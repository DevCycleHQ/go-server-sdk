package devcycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/util"
)

type EventFlushCallback func(payloads map[string]FlushPayload) (*FlushResult, error)

type InternalEventQueue interface {
	QueueEvent(user User, event Event) error
	QueueAggregateEvent(config BucketedUserConfig, event Event) error
	FlushEventQueue(EventFlushCallback) error
	UserQueueLength() (int, error)
}

type EventManager struct {
	internalQueue  InternalEventQueue
	flushMutex     *sync.Mutex
	sdkKey         string
	options        *Options
	cfg            *HTTPConfiguration
	closed         bool
	flushStop      chan bool
	forceFlush     chan bool
	eventsFlushed  atomic.Int32
	eventsReported atomic.Int32
	eventsDropped  atomic.Int32
}

type FlushResult struct {
	SuccessPayloads          []string
	FailurePayloads          []string
	FailureWithRetryPayloads []string
}

func NewEventManager(options *Options, localBucketing InternalEventQueue, cfg *HTTPConfiguration, sdkKey string) (eventQueue *EventManager, err error) {
	e := &EventManager{
		flushMutex: &sync.Mutex{},
	}
	e.options = options
	e.internalQueue = localBucketing
	e.cfg = cfg
	e.sdkKey = sdkKey

	e.flushStop = make(chan bool, 1)
	e.forceFlush = make(chan bool, 1)

	// Disable automatic flushing of events if all sources of events are disabled
	// DisableAutomaticEventLogging is passed into the WASM to disable events
	// from being emitted, so we don't need to flush them.
	if e.options.DisableAutomaticEventLogging && e.options.DisableCustomEventLogging {
		return e, nil
	}

	ticker := time.NewTicker(e.options.EventFlushIntervalMS)

	go func() {
		for {
			select {
			case <-ticker.C:
				err := e.FlushEvents()
				if err != nil {
					util.Warnf("Error flushing primary events queue: %s\n", err)
				}
			case <-e.forceFlush:
				err := e.FlushEvents()
				if err != nil {
					util.Warnf("Error flushing primary events queue: %s\n", err)
				}
			case <-e.flushStop:
				ticker.Stop()
				util.Infof("Stopping event flushing.")
			}
		}
	}()

	return e, nil
}

func (e *EventManager) QueueEvent(user User, event Event) error {
	if e.closed {
		return util.Errorf("DevCycle client was closed, no more events can be tracked.")
	}
	queueSize, err := e.internalQueue.UserQueueLength()
	if err != nil {
		return util.Errorf("Failed to check queue size, dropping event: %w", err)
	}

	if queueSize >= e.options.FlushEventQueueSize {
		select {
		case e.forceFlush <- true:
			util.Debugf("FlushEventQueueSize of %d reached: %d, flushing events", e.options.FlushEventQueueSize, queueSize)
		default:
		}
	}
	err = e.internalQueue.QueueEvent(user, event)
	if err != nil && errors.Is(err, ErrQueueFull) {
		e.eventsDropped.Add(1)
		return util.Errorf("event queue is full, dropping event: %+v", event)
	}
	return err
}

func (e *EventManager) QueueAggregateEvent(config BucketedUserConfig, event Event) error {
	return e.internalQueue.QueueAggregateEvent(config, event)
}

func (e *EventManager) FlushEvents() (err error) {
	e.flushMutex.Lock()
	defer e.flushMutex.Unlock()

	util.Debugf("Started flushing events")

	defer func() {
		if r := recover(); r != nil {
			// get the stack trace and potentially log it here
			err = util.Errorf("recovered from panic in FlushEvents: %v", r)
		}
	}()

	err = e.internalQueue.FlushEventQueue(func(payloads map[string]FlushPayload) (result *FlushResult, err error) {
		// TODO: use sum of event count
		e.eventsFlushed.Add(int32(len(payloads)))

		return e.flushEventPayloads(payloads)
	})

	if err != nil {
		return err
	}

	util.Debugf("Finished flushing events")

	return
}

func (e *EventManager) flushEventPayload(
	payload *FlushPayload,
	successes *[]string,
	failures *[]string,
	retryableFailures *[]string,
) {
	eventsHost := e.cfg.EventsAPIBasePath
	var req *http.Request
	var resp *http.Response
	requestBody, err := json.Marshal(BatchEventsBody{Batch: payload.Records})
	if err != nil {
		_ = util.Errorf("Failed to marshal batch events body: %s", err)
		e.reportPayloadFailure(payload, false, failures, retryableFailures)
		return
	}
	req, err = http.NewRequest("POST", eventsHost+"/v1/events/batch", bytes.NewReader(requestBody))
	if err != nil {
		_ = util.Errorf("Failed to create request to events api: %s", err)
		e.reportPayloadFailure(payload, false, failures, retryableFailures)
		return
	}

	req.Header.Set("Authorization", e.sdkKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err = e.cfg.HTTPClient.Do(req)

	if err != nil {
		_ = util.Errorf("Failed to make request to events api: %s", err)
		e.reportPayloadFailure(payload, false, failures, retryableFailures)
		return
	}

	// always ensure body is closed to avoid goroutine leak
	defer func() {
		_ = resp.Body.Close()
	}()

	// always read response body fully - from net/http docs:
	// If the Body is not both read to EOF and closed, the Client's
	// underlying RoundTripper (typically Transport) may not be able to
	// re-use a persistent TCP connection to the server for a subsequent
	// "keep-alive" request.
	responseBody, readError := io.ReadAll(resp.Body)
	if readError != nil {
		_ = util.Errorf("Failed to read response body: %v", readError)
		e.reportPayloadFailure(payload, false, failures, retryableFailures)
		return
	}

	if resp.StatusCode >= 500 {
		util.Warnf("Events API Returned a 5xx error, retrying later.")
		e.reportPayloadFailure(payload, true, failures, retryableFailures)
		return
	}

	if resp.StatusCode >= 400 {
		e.reportPayloadFailure(payload, false, failures, retryableFailures)
		_ = util.Errorf("Error sending events - Response: %s", string(responseBody))
		return
	}

	if resp.StatusCode == 201 {
		e.reportPayloadSuccess(payload, successes)
		e.eventsReported.Add(1)
		return
	}

	_ = util.Errorf("unknown status code when flushing events %d", resp.StatusCode)
	e.reportPayloadFailure(payload, false, failures, retryableFailures)
}

func (e *EventManager) flushEventPayloads(payloads map[string]FlushPayload) (result *FlushResult, err error) {
	e.eventsFlushed.Add(int32(len(payloads)))
	successes := make([]string, 0, len(payloads))
	failures := make([]string, 0)
	retryableFailures := make([]string, 0)

	for _, payload := range payloads {
		e.flushEventPayload(&payload, &successes, &failures, &retryableFailures)
	}

	return &FlushResult{
		SuccessPayloads:          successes,
		FailurePayloads:          failures,
		FailureWithRetryPayloads: retryableFailures,
	}, nil
}

func (e *EventManager) reportPayloadSuccess(payload *FlushPayload, successPayloads *[]string) {
	*successPayloads = append(*successPayloads, payload.PayloadId)
}

func (e *EventManager) reportPayloadFailure(
	payload *FlushPayload,
	retry bool,
	failurePayloads *[]string,
	retryableFailurePayloads *[]string,
) {
	if retry {
		*retryableFailurePayloads = append(*retryableFailurePayloads, payload.PayloadId)
	} else {
		*failurePayloads = append(*failurePayloads, payload.PayloadId)
	}
}

func (e *EventManager) Metrics() (int32, int32, int32) {
	return e.eventsFlushed.Load(), e.eventsReported.Load(), e.eventsDropped.Load()
}

func (e *EventManager) Close() (err error) {
	e.flushStop <- true
	e.closed = true
	err = e.FlushEvents()
	return err
}
