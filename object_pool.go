package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	pool "github.com/jolestar/go-commons-pool/v2"
	"sync/atomic"
)

var (
	objectId = atomic.Int32{}
)

type BucketingPoolFactory struct {
	wasmMain *WASMMain
	sdkKey   string
	options  *DVCOptions
}

func (f *BucketingPoolFactory) MakeObject(ctx context.Context) (*pool.PooledObject, error) {
	var bucketing = &BucketingPoolObject{}
	err := bucketing.Initialize(f.wasmMain, f.sdkKey, f.options)
	if err != nil {
		return nil, err
	}

	return pool.NewPooledObject(
			bucketing),
		nil
}

func (f *BucketingPoolFactory) DestroyObject(ctx context.Context, object *pool.PooledObject) error {
	// do destroy
	return nil
}

func (f *BucketingPoolFactory) ValidateObject(ctx context.Context, object *pool.PooledObject) bool {
	// do validate
	return true
}

func (f *BucketingPoolFactory) ActivateObject(ctx context.Context, object *pool.PooledObject) error {
	// do activate
	return nil
}

func (f *BucketingPoolFactory) PassivateObject(ctx context.Context, object *pool.PooledObject) error {
	// do passivate
	return nil
}

type BucketingPoolObject struct {
	localBucketing *DevCycleLocalBucketing
	id             int32
}

func (o *BucketingPoolObject) Initialize(wasmMain *WASMMain, sdkKey string, options *DVCOptions) (err error) {
	o.localBucketing = &DevCycleLocalBucketing{}
	err = o.localBucketing.Initialize(wasmMain, sdkKey, options)
	//o.storeConfigChan = make(chan *[]byte, 1)
	//o.storeConfigResponseChan = make(chan error, 1)
	//o.setClientCustomDataChan = make(chan *[]byte, 1)
	//o.setClientCustomDataResponseChan = make(chan error, 1)

	var eventQueueOpt []byte
	eventQueueOpt, err = json.Marshal(options.eventQueueOptions())
	if err != nil {
		return fmt.Errorf("error marshalling event queue options: %w", err)
	}
	err = o.localBucketing.initEventQueue(eventQueueOpt)
	if err != nil {
		return fmt.Errorf("error initializing worker event queue: %w", err)
	}

	//o.flushInProgress = atomic.Bool{}
	//o.flushResultChannel = make(chan *FlushResult)

	o.id = objectId.Add(1)
	return
}

type BucketingPool struct {
	pool *pool.ObjectPool
	ctx  context.Context
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
	bucketingPool.pool = pool.NewObjectPool(ctx, &BucketingPoolFactory{
		wasmMain: wasmMain,
		sdkKey:   sdkKey,
		options:  options,
	}, config)

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

func (p *BucketingPool) SetConfig(config []byte) error {
	for i := 0; i < p.pool.Config.MaxTotal; i++ {
		bucketing, err := p.pool.BorrowObject(p.ctx)
		if err != nil {
			return err
		}

		defer func() {
			err = p.pool.ReturnObject(p.ctx, bucketing)
			if err != nil {
				// TODO do we need to panic here?
				panic(err)
			}
		}()

		err = bucketing.(*BucketingPoolObject).localBucketing.StoreConfig(config)

		if err != nil {
			return err
		}
	}

	return nil
}
