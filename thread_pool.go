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
	eventsQueue chan []FlushPayload
	hasConfig   bool
	id          int32
}

type VariableForUserPayload struct {
	User         *[]byte
	Key          *string
	VariableType VariableTypeCode
}

type VariableForUserResponse struct {
	Variable *Variable
	Err      error
}

func (w *LocalBucketingWorker) Initialize(wasmMain *WASMMain, sdkKey string, eventsQueue chan []FlushPayload, options *DVCOptions) (err error) {
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
	w.flushStop = make(chan bool, 1)
	go w.eventLoop(ticker)

	w.id = workerId.Add(1)
	return
}

func (w *LocalBucketingWorker) eventLoop(ticker *time.Ticker) {
	for {
		select {
		case <-w.flushStop:
			ticker.Stop()
			infof("LocalBucketingWorker: Stopping event flushing.")
			return
		case <-ticker.C:
			err := w.flushEvents()
			if err != nil {
				warnf("LocalBucketingWorker: Error flushing events: %s\n", err)
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
	w.eventsQueue <- payloads
	for _, payload := range payloads {
		if err = w.localBucketing.onPayloadSuccess(payload.PayloadId); err != nil {
			// TODO: Need to handle this better: otherwise next flushEventQueue will fail
			return err
		}
	}
	return nil
}

func (w *LocalBucketingWorker) Process(payload interface{}) interface{} {
	var variableForUserPayload = payload.(*VariableForUserPayload)

	variable, err := w.variableForUser(
		variableForUserPayload.User,
		variableForUserPayload.Key,
		variableForUserPayload.VariableType,
		true,
	)

	return VariableForUserResponse{
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

/**
 * Called by the thread pool each time a job is completed.
 *	When the function returns, the worker will be returned to the pool and given a new job when needed.
 * We use this moment to check for any external state updates that have come in (eg. a new config or client custom data)
 *  and process them since we are not currently busy with a job.
 */
func (w *LocalBucketingWorker) BlockUntilReady() {
	for {
		select {
		case configData := <-w.storeConfigChan:
			err := w.storeConfig(*configData)
			w.storeConfigResponseChan <- err
		case customData := <-w.setClientCustomDataChan:
			err := w.setClientCustomData(*customData)
			w.setClientCustomDataResponseChan <- err
		default:
			// keep blocking this worker until it has a config
			if w.hasConfig {
				return
			}
		}
	}
}

func (w *LocalBucketingWorker) Interrupt() {}
func (w *LocalBucketingWorker) Terminate() {
	w.flushStop <- true
}
