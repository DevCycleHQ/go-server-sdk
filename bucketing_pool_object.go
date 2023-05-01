//go:build !native_bucketing

package devcycle

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

var (
	objectId = atomic.Int32{}
)

type BucketingPoolObject struct {
	localBucketing   *WASMLocalBucketingClient
	id               int32
	configData       *[]byte
	clientCustomData *[]byte
}

func (o *BucketingPoolObject) Initialize(wasmMain *WASMMain, sdkKey string, platformData *PlatformData, options *DVCOptions) (err error) {
	o.localBucketing = &WASMLocalBucketingClient{}
	err = o.localBucketing.Initialize(wasmMain, sdkKey, platformData, options)

	if err != nil {
		return
	}

	var eventQueueOpt []byte
	eventQueueOpt, err = json.Marshal(options.eventQueueOptions())
	if err != nil {
		return fmt.Errorf("error marshalling event queue options: %w", err)
	}
	err = o.localBucketing.initEventQueue(eventQueueOpt)
	if err != nil {
		return fmt.Errorf("error initializing worker event queue: %w", err)
	}

	o.id = objectId.Add(1)
	return
}

func (o *BucketingPoolObject) StoreConfig(config *[]byte) (err error) {
	if o.configData == config {
		return nil
	}

	err = o.localBucketing.StoreConfig(*config)
	if err != nil {
		return err
	}
	o.configData = config
	return
}

func (o *BucketingPoolObject) SetClientCustomData(clientCustomData *[]byte) (err error) {
	if o.clientCustomData == clientCustomData {
		return nil
	}

	err = o.localBucketing.SetClientCustomData(*clientCustomData)
	if err != nil {
		return err
	}

	o.clientCustomData = clientCustomData
	return
}

func (o *BucketingPoolObject) FlushEvents() ([]FlushPayload, error) {
	return o.localBucketing.flushEventQueue()
}

func (o *BucketingPoolObject) HandleFlushResults(result *FlushResult) {
	o.localBucketing.HandleFlushResults(result)
}
