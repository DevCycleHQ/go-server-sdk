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
	localBucketing   *DevCycleLocalBucketing
	flushResultChan  chan *FlushResult
	id               int32
	configData       *[]byte
	clientCustomData *[]byte
	lastFlushTime    int64
	flushChan        chan *PayloadsAndChannel
}

func (o *BucketingPoolObject) Initialize(wasmMain *WASMMain, sdkKey string, options *DVCOptions, flushChan chan *PayloadsAndChannel) (err error) {
	o.localBucketing = &DevCycleLocalBucketing{}
	err = o.localBucketing.Initialize(wasmMain, sdkKey, options)
	o.flushChan = flushChan

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

func (o *BucketingPoolObject) writeToFlushChannel(lastFlushTime int64, payloads *PayloadsAndChannel) {
	o.lastFlushTime = lastFlushTime
	o.flushChan <- payloads
}

func (o *BucketingPoolObject) FlushEvents(lastFlushTime int64) {
	if o.lastFlushTime > lastFlushTime {
		return
	}
	payloads, err := o.localBucketing.flushEventQueue()
	if err != nil {
		o.writeToFlushChannel(lastFlushTime, nil)
		return
	}

	if len(payloads) == 0 {
		o.writeToFlushChannel(lastFlushTime, nil)
		return
	}

	o.flushResultChan = make(chan *FlushResult)

	o.writeToFlushChannel(lastFlushTime, &PayloadsAndChannel{
		payloads: payloads,
		channel:  &o.flushResultChan,
	})

	return
}
