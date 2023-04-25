package devcycle

import (
	"context"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"sync"
	"sync/atomic"

	"github.com/devcyclehq/go-server-sdk/v2/proto"
	pool "github.com/jolestar/go-commons-pool/v2"
)

type BucketingPool struct {
	pool1         *pool.ObjectPool
	pool2         *pool.ObjectPool
	currentPool   atomic.Pointer[pool.ObjectPool]
	poolSwapMutex sync.Mutex
	ctx           context.Context
	factory       *BucketingPoolFactory
	closed        atomic.Bool
}

func NewBucketingPool(ctx context.Context, wasmMain *WASMMain, sdkKey string, platformData *PlatformData, options *DVCOptions) (*BucketingPool, error) {
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

	bucketingPool.factory = MakeBucketingPoolFactory(wasmMain, sdkKey, platformData, options, bucketingPool)

	bucketingPool.poolSwapMutex = sync.Mutex{}
	bucketingPool.closed = atomic.Bool{}

	bucketingPool.pool1 = pool.NewObjectPool(ctx, bucketingPool.factory, config)
	bucketingPool.pool2 = pool.NewObjectPool(ctx, bucketingPool.factory, config)

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
	if p.closed.Load() {
		return nil, util.Errorf("Cannot evaluate variable on closed pool")
	}
	currentPool := p.currentPool.Load()
	bucketing, err := currentPool.BorrowObject(p.ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = currentPool.ReturnObject(p.ctx, bucketing)
	}()

	b := bucketing.(*BucketingPoolObject)
	variablePB, err := b.localBucketing.VariableForUser_PB(paramsBuffer)

	if err != nil {
		return nil, err
	}

	return variablePB, nil
}

// ProcessAll will func the "process" handler function on every object in the pool. It will block until the operation
// has completed for every object, or there was an error. It naively grabs the longest idle object from the pool each
// time and checks if it has seen it before. If it has, it will immediately return it and try again.
func (p *BucketingPool) ProcessAll(
	operationName string,
	process func(object *BucketingPoolObject) error,
) (err error) {
	if p.closed.Load() {
		return util.Errorf("Cannot process task on closed pool")
	}
	p.poolSwapMutex.Lock()
	defer p.poolSwapMutex.Unlock()

	currentPool := p.currentPool.Load()

	inactivePool := p.pool1
	if currentPool == p.pool1 {
		inactivePool = p.pool2
	}

	processAll := func(processPool *pool.ObjectPool) error {
		i := 0
		curObjects := make([]*BucketingPoolObject, processPool.Config.MaxTotal)
		for i < processPool.Config.MaxTotal {
			borrowed, err := processPool.BorrowObject(p.ctx)
			if err != nil {
				return err
			}

			curObj := borrowed.(*BucketingPoolObject)

			err = process(curObj)

			if err != nil {
				_ = processPool.ReturnObject(p.ctx, curObj)
				return err
			}
			curObjects[i] = curObj
			i += 1
		}

		for _, curObj := range curObjects {
			if curObj != nil {
				_ = processPool.ReturnObject(p.ctx, curObj)
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

	return err
}

func (p *BucketingPool) StoreConfig(config []byte) error {
	if p.closed.Load() {
		return util.Errorf("Cannot set config on closed pool")
	}
	util.Debugf("Setting config on all workers")
	return p.ProcessAll("StoreConfig", func(object *BucketingPoolObject) error {
		return object.StoreConfig(&config)
	})
}

func (p *BucketingPool) SetClientCustomData(customData []byte) error {
	if p.closed.Load() {
		return util.Errorf("Cannot set client custom data on closed pool")
	}
	util.Debugf("Setting client custom data on all workers")
	return p.ProcessAll("SetClientCustomData", func(object *BucketingPoolObject) error {
		return object.SetClientCustomData(&customData)
	})
}

func (p *BucketingPool) Close() {
	p.pool1.Close(p.ctx)
	p.pool2.Close(p.ctx)
	p.closed.Store(true)
}
