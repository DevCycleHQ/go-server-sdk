package devcycle

import (
	"context"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	pool "github.com/jolestar/go-commons-pool/v2"
	"sync"
	"sync/atomic"
	"time"
)

type BucketingPool struct {
	pool1            *pool.ObjectPool
	pool2            *pool.ObjectPool
	currentPool      atomic.Pointer[pool.ObjectPool]
	poolSwapMutex    sync.Mutex
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
	bucketingPool.poolSwapMutex = sync.Mutex{}

	bucketingPool.pool1 = pool.NewObjectPool(ctx, bucketingPool.factory, config)
	bucketingPool.pool2 = pool.NewObjectPool(ctx, bucketingPool.factory, config)
	bucketingPool.eventFlushChan = make(chan *PayloadsAndChannel)

	bucketingPool.currentPool.Store(bucketingPool.pool1)

	for i := 0; i < options.MaxWasmWorkers; i++ {
		err := bucketingPool.pool1.AddObject(ctx)
		if err != nil {
			return nil, err
		}
		err = bucketingPool.pool2.AddObject(ctx)
		if err != nil {
			return nil, err
		}
	}
	return bucketingPool, nil
}

func (p *BucketingPool) VariableForUser(paramsBuffer []byte) (*proto.SDKVariable_PB, error) {
	currentPool := p.currentPool.Load()
	bucketing, err := currentPool.BorrowObject(p.ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = currentPool.ReturnObject(p.ctx, bucketing)
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

func (p *BucketingPool) pokeOne(currentPool *pool.ObjectPool) error {
	bucketing, err := currentPool.BorrowObject(p.ctx)
	if err != nil {
		return err
	}

	defer func() {
		go func() {
			err = currentPool.ReturnObject(p.ctx, bucketing)
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
	currentPool := p.currentPool.Load()

	for i := 0; i < currentPool.Config.MaxTotal; i++ {
		err := p.pokeOne(currentPool)

		if err != nil {
			return err
		}
	}

	return nil
}

// ProcessAll will func the "process" handler function on every object in the pool. It will block until the operation
// has completed for every object, or there was an error. It naively grabs the longest idle object from the pool each
// time and checks if it has seen it before. If it has, it will immediately return it and try again.
func (p *BucketingPool) ProcessAll(
	operationName string,
	process func(object *BucketingPoolObject) error,
	timeout time.Duration,
) (err error) {
	p.poolSwapMutex.Lock()
	defer p.poolSwapMutex.Unlock()

	currentPool := p.currentPool.Load()

	inactivePool := p.pool1
	if currentPool == p.pool1 {
		inactivePool = p.pool2
	}

	idMap := make(map[int32]bool)
	start := time.Now()
	ctx, cancel := context.WithTimeout(p.ctx, timeout)

	defer cancel()

	wrongObjectCount := 0

	processAll := func(processPool *pool.ObjectPool) error {
		for len(idMap) < processPool.Config.MaxTotal && ctx.Err() == nil {
			var curObj *BucketingPoolObject
			for curObj == nil {
				borrowed, err := processPool.BorrowObject(ctx)
				curObj = borrowed.(*BucketingPoolObject)
				if err != nil {
					_ = processPool.ReturnObject(p.ctx, curObj)
					if ctx.Err() != nil {
						return fmt.Errorf("timed out after %s while processing %s on all workers", timeout, operationName)
					}
					return err
				}

				if idMap[curObj.id] {
					_ = processPool.ReturnObject(p.ctx, curObj)
					wrongObjectCount += 1
					curObj = nil
				}
			}

			idMap[curObj.id] = true

			err := process(curObj)
			_ = processPool.ReturnObject(p.ctx, curObj)

			if err != nil {
				return err
			}
		}

		return nil
	}

	err = processAll(inactivePool)
	if err != nil {
		return err
	}

	p.currentPool.Swap(inactivePool)
	err = processAll(currentPool)

	//debugf("Borrowing all objects from pool for %s (max %d)", operationName, p.pool.Config.MaxTotal)

	debugf("Borrowed all objects from pool for %s in %s with %d repeated objects", operationName, time.Since(start), wrongObjectCount)

	return nil
}

func (p *BucketingPool) SetConfig(config []byte) error {
	p.configData = &config
	debugf("Setting config on all workers")
	return p.ProcessAll("SetConfig", func(object *BucketingPoolObject) error {
		return object.StoreConfig(&config)
	}, 5*time.Second)
}

func (p *BucketingPool) SetClientCustomData(customData []byte) error {
	p.clientCustomData = &customData
	debugf("Setting client custom data on all workers")
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
