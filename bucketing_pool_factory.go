package devcycle

import (
	"context"
	pool "github.com/jolestar/go-commons-pool/v2"
)

type BucketingPoolFactory struct {
	wasmMain *WASMMain
	sdkKey   string
	options  *DVCOptions
	pool     *BucketingPool
}

func MakeBucketingPoolFactory(wasmMain *WASMMain, sdkKey string, options *DVCOptions, pool *BucketingPool) *BucketingPoolFactory {
	return &BucketingPoolFactory{
		wasmMain: wasmMain,
		sdkKey:   sdkKey,
		options:  options,
		pool:     pool,
	}
}

func (f *BucketingPoolFactory) MakeObject(ctx context.Context) (*pool.PooledObject, error) {
	var bucketing = &BucketingPoolObject{}
	err := bucketing.Initialize(f.wasmMain, f.sdkKey, f.options, f.pool.eventFlushChan)
	if err != nil {
		return nil, err
	}

	return pool.NewPooledObject(
			bucketing),
		nil
}

func (f *BucketingPoolFactory) DestroyObject(ctx context.Context, object *pool.PooledObject) error {
	return nil
}

func (f *BucketingPoolFactory) ValidateObject(ctx context.Context, object *pool.PooledObject) bool {
	// do validate
	return true
}

func (f *BucketingPoolFactory) ActivateObject(ctx context.Context, object *pool.PooledObject) error {
	return nil
}

func (f *BucketingPoolFactory) PassivateObject(ctx context.Context, object *pool.PooledObject) error {
	return nil
}
