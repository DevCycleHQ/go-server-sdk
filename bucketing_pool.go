package devcycle

import (
	"context"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	pool "github.com/jolestar/go-commons-pool/v2"
	"sync"
	"time"
)

type BucketingPool struct {
	pool             *pool.ObjectPool
	ctx              context.Context
	factory          *BucketingPoolFactory
	configData       *[]byte
	clientCustomData *[]byte
	lastFlushTime    int64
	eventFlushChan   chan *PayloadsAndChannel
	isFlushingMutex  sync.Mutex
	isFlushing       bool
}

func NewBucketingPool(ctx context.Context, wasmMain *WASMMain, sdkKey string, options *DVCOptions) (*BucketingPool, error) {
	bucketingPool := &BucketingPool{
		ctx: ctx,
	}
	config := pool.NewDefaultPoolConfig()
	config.LIFO = false
	config.MaxTotal = options.MaxWasmWorkers
	config.MaxIdle = options.MaxWasmWorkers
	// disable idle evictions, we want these objects to stay active forever
	config.MinEvictableIdleTime = -1
	config.TimeBetweenEvictionRuns = -1

	bucketingPool.factory = MakeBucketingPoolFactory(wasmMain, sdkKey, options, bucketingPool)

	bucketingPool.isFlushingMutex = sync.Mutex{}

	bucketingPool.pool = pool.NewObjectPool(ctx, bucketingPool.factory, config)
	bucketingPool.eventFlushChan = make(chan *PayloadsAndChannel)

	for i := 0; i < options.MaxWasmWorkers; i++ {
		err := bucketingPool.pool.AddObject(ctx)
		if err != nil {
			return nil, err
		}
	}
	return bucketingPool, nil
}

func (p *BucketingPool) VariableForUser(paramsBuffer []byte) (*proto.SDKVariable_PB, error) {
	bucketing, err := p.pool.BorrowObject(p.ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = p.pool.ReturnObject(p.ctx, bucketing)
		if err != nil {
			// TODO do we need to panic here?
			panic(err)
		}
	}()

	b := bucketing.(*BucketingPoolObject)
	variablePB, err := b.localBucketing.VariableForUser_PB(paramsBuffer)

	if err != nil {
		return nil, err
	}

	return variablePB, nil
}

func (p *BucketingPool) pokeOne() error {
	bucketing, err := p.pool.BorrowObject(p.ctx)
	if err != nil {
		return err
	}

	defer func() {
		go func() {
			err = p.pool.ReturnObject(p.ctx, bucketing)
			if err != nil {
				// TODO do we need to panic here?
				panic(err)
			}
		}()
	}()

	return nil
}

// borrow N objects and return them immediately. This will trigger the passivate handler for any idle objects, while
// assuming that any objects that weren't borrowed during this process were busy and thus passivate will be called anyway
func (p *BucketingPool) PokeAll() error {
	printf("Poking all workers")

	for i := 0; i < p.pool.Config.MaxTotal; i++ {
		err := p.pokeOne()

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *BucketingPool) ProcessAll(
	operationName string,
	process func(object *BucketingPoolObject) error,
	timeout time.Duration,
) (err error) {
	idMap := make(map[int32]bool)
	//start := time.Now()
	ctx, cancel := context.WithTimeout(p.ctx, timeout)

	defer cancel()

	//debugf("Borrowing all objects from pool for %s (max %d)", operationName, p.pool.Config.MaxTotal)
	for len(idMap) < p.pool.Config.MaxTotal && ctx.Err() == nil {
		var curObj *BucketingPoolObject
		for curObj == nil {
			borrowed, err := p.pool.BorrowObject(ctx)
			curObj = borrowed.(*BucketingPoolObject)
			if err != nil {
				_ = p.pool.ReturnObject(ctx, curObj)
				return err
			}

			if idMap[curObj.id] {
				_ = p.pool.ReturnObject(p.ctx, curObj)
				curObj = nil
			}
		}

		idMap[curObj.id] = true

		err := process(curObj)
		_ = p.pool.ReturnObject(ctx, curObj)

		if err != nil {
			return err
		}
	}

	if ctx.Err() != nil {
		return fmt.Errorf("timed out after %s while processing %s on all workers", timeout, operationName)
	}

	//debugf("Borrowed all objects from pool for %s in %s", operationName, time.Since(start))

	return nil
}

func (p *BucketingPool) SetConfig(config []byte) error {
	p.configData = &config
	debugf("Setting config on all workers (max %d)", p.pool.Config.MaxTotal)
	return p.ProcessAll("SetConfig", func(object *BucketingPoolObject) error {
		return object.StoreConfig(&config)
	}, 5*time.Second)
}

func (p *BucketingPool) SetClientCustomData(customData []byte) error {
	p.clientCustomData = &customData
	debugf("Setting client custom data on all workers (max %d)", p.pool.Config.MaxTotal)
	return p.ProcessAll("SetClientCustomData", func(object *BucketingPoolObject) error {
		return object.SetClientCustomData(&customData)
	}, 5*time.Second)
}

//func (p *BucketingPool) FlushEvents() (flushPayloads []PayloadsAndChannel, err error) {
//	p.isFlushingMutex.Lock()
//	defer p.isFlushingMutex.Unlock()
//	if p.isFlushing {
//		return
//	}
//	printf("Starting to flush events")
//	p.lastFlushTime = time.Now().UnixMilli()
//	p.isFlushing = true
//	defer func() { p.isFlushing = false }()
//
//	// Trigger everyone to wake up and check the lastFlushTime
//	err = p.PokeAll()
//	if err != nil {
//		errorf("error", err)
//		return nil, err
//	}
//
//	printf("Waiting for all channels to report in to flush events")
//	count := 0
//	flushPayloads = make([]PayloadsAndChannel, 0)
//	for count < p.pool.Config.MaxTotal {
//		select {
//		case payload := <-p.eventFlushChan:
//			printf("received events on channel!")
//			if payload != nil {
//				flushPayloads = append(flushPayloads, *payload)
//			}
//			count += 1
//		case <-time.After(60 * time.Second):
//			return nil, fmt.Errorf("Timed out while waiting for events to flush")
//		}
//	}
//
//	printf("Finished receiving events on channels")
//
//	return
//}
