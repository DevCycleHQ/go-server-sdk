package devcycle

import (
	"context"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	pool "github.com/jolestar/go-commons-pool/v2"
	"sync/atomic"
	"time"
)

type BucketingPool struct {
	pool             *pool.ObjectPool
	ctx              context.Context
	factory          *BucketingPoolFactory
	configData       *[]byte
	clientCustomData *[]byte
	lastFlushTime    int64
	eventFlushChans  []chan *PayloadsAndChannel
	isFlushing       atomic.Bool
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

	bucketingPool.isFlushing = atomic.Bool{}

	bucketingPool.pool = pool.NewObjectPool(ctx, bucketingPool.factory, config)
	bucketingPool.eventFlushChans = make([]chan *PayloadsAndChannel, options.MaxWasmWorkers)

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
		err = p.pool.ReturnObject(p.ctx, bucketing)
		if err != nil {
			// TODO do we need to panic here?
			panic(err)
		}
	}()

	return nil
}

// borrow N objects and return them immediately. This will trigger the passivate handler for any idle objects, while
// assuming that any objects that weren't borrowed during this process were busy and thus passivate will be called anyway
func (p *BucketingPool) PokeAll() error {
	for i := 0; i < p.pool.Config.MaxTotal; i++ {
		err := p.pokeOne()

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *BucketingPool) SetConfig(config []byte) error {
	p.configData = &config
	return p.PokeAll()
}

func (p *BucketingPool) SetClientCustomData(customData []byte) error {
	p.clientCustomData = &customData
	return p.PokeAll()
}

func (p *BucketingPool) FlushEvents() (flushPayloads []PayloadsAndChannel, err error) {
	if p.isFlushing.Load() {
		return
	}
	p.lastFlushTime = time.Now().UnixMilli()
	p.isFlushing.Store(true)
	defer p.isFlushing.Store(false)

	// Trigger everyone to wake up and check the lastFlushTime
	err = p.PokeAll()
	if err != nil {
		return nil, err
	}

	for _, channel := range p.eventFlushChans {
		payloads := <-channel

		flushPayloads = append(flushPayloads, *payloads)
	}

	return
}

func (p *BucketingPool) RegisterFlushChannel(channel chan *PayloadsAndChannel) {
	p.eventFlushChans = append(p.eventFlushChans, channel)
	return
}
