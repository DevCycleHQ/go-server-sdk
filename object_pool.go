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
