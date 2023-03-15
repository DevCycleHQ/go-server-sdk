package devcycle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	"sync/atomic"
)

var (
	workerId = atomic.Int32{}
)

type LocalBucketingWorker struct {
	localBucketing *DevCycleLocalBucketing
	configEtag     string
	configData     []byte
	// channel to submit a job external to the pool that must be processed by this specific worker
	// used for things like storing the new config across every worker in the pool (when the worker is free)
	storeConfigChan                 chan *[]byte
	storeConfigResponseChan         chan error
	setClientCustomDataChan         chan *[]byte
	setClientCustomDataResponseChan chan error
	flushResultChannel              chan *FlushResult

	flushInProgress atomic.Bool
	id              int32
}

type WorkerPayloadType int32

const (
	VariableForUser WorkerPayloadType = iota
	StoreConfig
	SetClientCustomData
	FlushEvents
)

type WorkerPoolPayload struct {
	Type_              WorkerPayloadType
	VariableEvalParams *[]byte
	ConfigData         *[]byte
	ClientCustomData   *[]byte
	VariableType       VariableTypeCode
}

type WorkerPoolResponse struct {
	Variable *proto.SDKVariable_PB
	Events   *PayloadsAndChannel
	ImHere   bool
	Err      error
}

type FlushResult struct {
	SuccessPayloads          []string
	FailurePayloads          []string
	FailureWithRetryPayloads []string
}

func (w *LocalBucketingWorker) Initialize(wasmMain *WASMMain, sdkKey string, options *DVCOptions) (err error) {
	w.localBucketing = &DevCycleLocalBucketing{}
	err = w.localBucketing.Initialize(wasmMain, sdkKey, options)
	w.storeConfigChan = make(chan *[]byte, 1)
	w.storeConfigResponseChan = make(chan error, 1)
	w.setClientCustomDataChan = make(chan *[]byte, 1)
	w.setClientCustomDataResponseChan = make(chan error, 1)

	var eventQueueOpt []byte
	eventQueueOpt, err = json.Marshal(options.eventQueueOptions())
	if err != nil {
		return fmt.Errorf("error marshalling event queue options: %w", err)
	}
	err = w.localBucketing.initEventQueue(eventQueueOpt)
	if err != nil {
		return fmt.Errorf("error initializing worker event queue: %w", err)
	}

	w.flushInProgress = atomic.Bool{}
	w.flushResultChannel = make(chan *FlushResult)

	w.id = workerId.Add(1)
	return
}

func (w *LocalBucketingWorker) flushEvents() (*PayloadsAndChannel, error) {
	payloads, err := w.localBucketing.flushEventQueue()
	if err != nil {
		return nil, err
	}
	if len(payloads) == 0 {
		return nil, nil
	}

	w.flushInProgress.Store(true)

	return &PayloadsAndChannel{
		payloads: payloads,
		channel:  &w.flushResultChannel,
	}, nil
}

func (w *LocalBucketingWorker) Process(payload interface{}) interface{} {
	var workerPayload = payload.(*WorkerPoolPayload)

	if workerPayload.Type_ == StoreConfig {
		err := w.storeConfig(*workerPayload.ConfigData)
		return WorkerPoolResponse{
			Err: err,
		}
	} else if workerPayload.Type_ == SetClientCustomData {
		err := w.setClientCustomData(*workerPayload.ClientCustomData)
		return WorkerPoolResponse{
			Err: err,
		}
	} else if workerPayload.Type_ == FlushEvents {
		debugf("Flushing events from worker %d", w.id)
		events, err := w.flushEvents()
		return WorkerPoolResponse{
			Events: events,
			Err:    err,
		}
	}

	variable, err := w.variableForUser(
		workerPayload.VariableEvalParams,
	)

	return WorkerPoolResponse{
		Variable: variable,
		Err:      err,
	}
}

func (w *LocalBucketingWorker) variableForUser(params *[]byte) (*proto.SDKVariable_PB, error) {
	return w.localBucketing.VariableForUser_PB(*params)
}

func (w *LocalBucketingWorker) storeConfig(configData []byte) error {
	return w.localBucketing.StoreConfig(configData)
}

func (w *LocalBucketingWorker) setClientCustomData(customData []byte) error {
	return w.localBucketing.SetClientCustomData(customData)
}

/**
 * Called by the thread pool each time a job is completed.
 *	When the function returns, the worker will be returned to the pool and given a new job when needed.
 * We use this moment to check for any external state updates that have come in (eg. a new config or client custom data)
 *  and process them since we are not currently busy with a job.
 */
func (w *LocalBucketingWorker) BlockUntilReady() {
	if w.flushInProgress.Load() {
		select {
		case result := <-w.flushResultChannel:
			for _, payloadId := range result.SuccessPayloads {
				if err := w.localBucketing.onPayloadSuccess(payloadId); err != nil {
					_ = errorf("failed to mark event payloads as successful", err)
				}
			}
			for _, payloadId := range result.FailurePayloads {
				if err := w.localBucketing.onPayloadFailure(payloadId, false); err != nil {
					_ = errorf("failed to mark event payloads as failed", err)

				}
			}
			for _, payloadId := range result.FailureWithRetryPayloads {
				if err := w.localBucketing.onPayloadFailure(payloadId, true); err != nil {
					_ = errorf("failed to mark event payloads as failed", err)
				}
			}
		}
		w.flushInProgress.Store(false)
	}
}

func (w *LocalBucketingWorker) Interrupt() {}
func (w *LocalBucketingWorker) Terminate() {}
