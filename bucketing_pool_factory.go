package devcycle

import (
	"context"
	"fmt"
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
	panic(fmt.Errorf("local bucketing borrow pool should never destroy objects"))
	return nil
}

func (f *BucketingPoolFactory) ValidateObject(ctx context.Context, object *pool.PooledObject) bool {
	// do validate
	return true
}

func (f *BucketingPoolFactory) checkForWork(ctx context.Context, object *pool.PooledObject) error {
	bucketing := object.Object.(*BucketingPoolObject)
	if bucketing.flushResultChan != nil {
		select {
		case result := <-bucketing.flushResultChan:
			for _, payloadId := range result.SuccessPayloads {
				if err := bucketing.localBucketing.onPayloadSuccess(payloadId); err != nil {
					_ = errorf("failed to mark event payloads as successful", err)
				}
			}
			for _, payloadId := range result.FailurePayloads {
				if err := bucketing.localBucketing.onPayloadFailure(payloadId, false); err != nil {
					_ = errorf("failed to mark event payloads as failed", err)

				}
			}
			for _, payloadId := range result.FailureWithRetryPayloads {
				if err := bucketing.localBucketing.onPayloadFailure(payloadId, true); err != nil {
					_ = errorf("failed to mark event payloads as failed", err)
				}
			}
		}
		bucketing.flushResultChan = nil
	}
	// no-op if it already has the right config
	_ = bucketing.StoreConfig(f.pool.configData)
	_ = bucketing.SetClientCustomData(f.pool.clientCustomData)

	if f.pool.isFlushing {
		bucketing.FlushEvents(f.pool.lastFlushTime)
	}

	return nil
}

func (f *BucketingPoolFactory) ActivateObject(ctx context.Context, object *pool.PooledObject) error {
	return nil
}

func (f *BucketingPoolFactory) PassivateObject(ctx context.Context, object *pool.PooledObject) error {
	return f.checkForWork(ctx, object)
}
