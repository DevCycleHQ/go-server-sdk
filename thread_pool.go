package devcycle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
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
	// used to signal to the event flushing to stop
	flushStop chan bool
	// channel for passing back event payloads to the main event queue
	eventsQueue     chan PayloadsAndChannel
	flushEventsChan chan bool
	hasConfig       bool
	jobInProgress   atomic.Bool
	id              int32
}

type WorkerPoolPayload struct {
	// TODO make this an enum
	Type_            string
	User             *[]byte
	Key              *string
	ConfigData       *[]byte
	ClientCustomData *[]byte
	VariableType     VariableTypeCode
}

type WorkerPoolResponse struct {
	Variable *Variable
	Err      error
}

type FlushResult struct {
	SuccessPayloads          []string
	FailurePayloads          []string
	FailureWithRetryPayloads []string
}

func (w *LocalBucketingWorker) Initialize(wasmMain *WASMMain, sdkKey string, eventsQueue chan PayloadsAndChannel, options *DVCOptions) (err error) {
	w.localBucketing = &DevCycleLocalBucketing{}
	err = w.localBucketing.Initialize(wasmMain, sdkKey, options)
	w.storeConfigChan = make(chan *[]byte, 1)
	w.storeConfigResponseChan = make(chan error, 1)
	w.setClientCustomDataChan = make(chan *[]byte, 1)
	w.setClientCustomDataResponseChan = make(chan error, 1)
	w.eventsQueue = eventsQueue

	var eventQueueOpt []byte
	eventQueueOpt, err = json.Marshal(options.eventQueueOptions())
	if err != nil {
		return fmt.Errorf("error marshalling event queue options: %w", err)
	}
	err = w.localBucketing.initEventQueue(eventQueueOpt)
	if err != nil {
		return fmt.Errorf("error initializing worker event queue: %w", err)
	}

	ticker := time.NewTicker(options.EventFlushIntervalMS)
	checkForWorkTimer := time.NewTicker(1 * time.Second)

	w.flushStop = make(chan bool, 1)
	w.flushEventsChan = make(chan bool, 1)

	w.jobInProgress = atomic.Bool{}

	go w.tickers(ticker, checkForWorkTimer)

	w.id = workerId.Add(1)
	return
}

func (w *LocalBucketingWorker) tickers(flushTicker *time.Ticker, checkForWorkTimer *time.Ticker) {
	for {
		select {
		case <-w.flushStop:
			flushTicker.Stop()
			infof("LocalBucketingWorker: Stopping event flushing.")
			return
		case <-checkForWorkTimer.C:
			w.checkForExternalWork()
		case <-flushTicker.C:
			select {
			// write non-blockingly to notify that we want to flush still
			case w.flushEventsChan <- true:
			default:
			}
		}
	}
}

func (w *LocalBucketingWorker) flushEvents() error {
	payloads, err := w.localBucketing.flushEventQueue()
	if err != nil {
		return err
	}
	if len(payloads) == 0 {
		return nil
	}
	var responseChannel = make(chan FlushResult, 1)

	w.eventsQueue <- PayloadsAndChannel{
		payloads: payloads,
		channel:  &responseChannel,
	}

	var result = <-responseChannel
	for _, payloadId := range result.SuccessPayloads {
		if err = w.localBucketing.onPayloadSuccess(payloadId); err != nil {
			return err
		}
	}
	for _, payloadId := range result.FailurePayloads {
		if err = w.localBucketing.onPayloadSuccess(payloadId); err != nil {
			return err
		}
	}
	return nil
}

func (w *LocalBucketingWorker) Process(payload interface{}) interface{} {
	w.jobInProgress.Store(true)
	defer w.jobInProgress.Store(false)

	var workerPayload = payload.(*WorkerPoolPayload)

	if workerPayload.Type_ == "storeConfig" {
		err := w.storeConfig(*workerPayload.ConfigData)
		return WorkerPoolResponse{
			Err: err,
		}
	} else if workerPayload.Type_ == "setClientCustomData" {
		err := w.setClientCustomData(*workerPayload.ClientCustomData)
		return WorkerPoolResponse{
			Err: err,
		}
	} else if workerPayload.Type_ == "flushEvents" {
		err := w.flushEvents()
		return WorkerPoolResponse{
			Err: err,
		}
	}

	variable, err := w.variableForUser(
		workerPayload.User,
		workerPayload.Key,
		workerPayload.VariableType,
		true,
	)

	return WorkerPoolResponse{
		Variable: &variable,
		Err:      err,
	}
}

func (w *LocalBucketingWorker) variableForUser(user *[]byte, key *string, variableType VariableTypeCode, shouldTrackEvents bool) (Variable, error) {
	return w.localBucketing.VariableForUser(*user, *key, variableType, shouldTrackEvents)
}

func (w *LocalBucketingWorker) storeConfig(configData []byte) error {
	w.hasConfig = true
	return w.localBucketing.StoreConfig(configData)
}

func (w *LocalBucketingWorker) setClientCustomData(customData []byte) error {
	return w.localBucketing.SetClientCustomData(customData)
}

func (w *LocalBucketingWorker) checkForExternalWork() {
	//if w.jobInProgress.Load() {
	//	return
	//}
	//
	//for {
	//	select {
	//	case configData := <-w.storeConfigChan:
	//		err := w.storeConfig(*configData)
	//		w.storeConfigResponseChan <- err
	//	case customData := <-w.setClientCustomDataChan:
	//		err := w.setClientCustomData(*customData)
	//		w.setClientCustomDataResponseChan <- err
	//	case <-w.flushEventsChan:
	//		err := w.flushEvents()
	//		if err != nil {
	//			warnf("LocalBucketingWorker: Error flushing events: %s\n", err)
	//		}
	//	default:
	//		// keep blocking this worker until it has a config
	//		if w.hasConfig {
	//			return
	//		}
	//	}
	//}
}

/**
 * Called by the thread pool each time a job is completed.
 *	When the function returns, the worker will be returned to the pool and given a new job when needed.
 * We use this moment to check for any external state updates that have come in (eg. a new config or client custom data)
 *  and process them since we are not currently busy with a job.
 */
func (w *LocalBucketingWorker) BlockUntilReady() {
	w.checkForExternalWork()
}

func (w *LocalBucketingWorker) Interrupt() {}
func (w *LocalBucketingWorker) Terminate() {
	w.flushStop <- true
}
