package devcycle

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type EventQueue struct {
	localBucketing    *DevCycleLocalBucketing
	options           *DVCOptions
	eventQueueOptions *EventQueueOptions
	httpClient        *http.Client
	context           context.Context
	closed            bool
	ticker            *time.Ticker
	flushStop         chan bool
}

func (e *EventQueue) eventQueueOptionsFromDVCOptions(options *DVCOptions) *EventQueueOptions {
	return &EventQueueOptions{
		FlushEventsInterval:          options.EventFlushIntervalMS,
		DisableAutomaticEventLogging: options.DisableAutomaticEventLogging,
		DisableCustomEventLogging:    options.DisableCustomEventLogging,
	}
}

type EventQueueOptions struct {
	FlushEventsInterval          time.Duration `json:"flushEventsMS"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging"`
}

func (e *EventQueue) initialize(options *DVCOptions, localBucketing *DevCycleLocalBucketing) (err error) {
	e.context = context.Background()
	e.httpClient = localBucketing.cfg.HTTPClient
	e.options = options
	e.flushStop = make(chan bool, 1)

	if !e.options.EnableCloudBucketing && localBucketing != nil {
		e.localBucketing = localBucketing
		var eventQueueOpt []byte
		eventQueueOpt, err = json.Marshal(e.eventQueueOptionsFromDVCOptions(options))
		if err != nil {
			return err
		}
		err = e.localBucketing.initEventQueue(string(eventQueueOpt))
		ticker := time.NewTicker(e.options.EventFlushIntervalMS)

		go func() {
			for {
				select {
				case <-e.flushStop:
					ticker.Stop()
					infof("Stopping event flushing.")
					return
				case <-ticker.C:
					err = e.FlushEvents()
					if err != nil {
						warnf("Error flushing events: %s\n", err)
					}
				}
			}
		}()
		return err
	}
	return err
}

func (e *EventQueue) QueueEvent(user DVCUser, event DVCEvent) error {
	if e.closed {
		return errorf("DevCycle client was closed, no more events can be tracked.")
	}
	if q, err := e.checkEventQueueSize(); err != nil || q {
		return errorf("Max event queue size reached, dropping event")
	}
	if !e.options.EnableCloudBucketing {
		userstring, err := json.Marshal(user)
		if err != nil {
			return err
		}
		eventstring, err := json.Marshal(event)
		if err != nil {
			return err
		}
		err = e.localBucketing.queueEvent(string(userstring), string(eventstring))
		return err
	}
	return nil
}

func (e *EventQueue) QueueAggregateEvent(user BucketedUserConfig, event DVCEvent) error {
	if q, err := e.checkEventQueueSize(); err != nil || q {
		return errorf("Max event queue size reached, dropping aggregate event")
	}
	if !e.options.EnableCloudBucketing {
		eventstring, err := json.Marshal(event)
		err = e.localBucketing.queueAggregateEvent(string(eventstring), user)
		return err
	}
	return nil
}

func (e *EventQueue) checkEventQueueSize() (bool, error) {
	queueSize, err := e.localBucketing.checkEventQueueSize()
	if err != nil {
		return false, err
	}
	if queueSize >= e.options.FlushEventQueueSize {
		err = e.FlushEvents()
		if err != nil {
			return true, err
		}
		if queueSize >= e.options.MaxEventQueueSize {
			return true, nil
		}
	}
	return false, nil
}

func (e *EventQueue) FlushEvents() (err error) {
	eventsHost := e.localBucketing.cfg.EventsAPIBasePath
	e.localBucketing.startFlushEvents()
	defer e.localBucketing.finishFlushEvents()
	payloads, err := e.localBucketing.flushEventQueue()
	if err != nil {
		return err
	}

	for _, payload := range payloads {
		var req *http.Request
		var resp *http.Response
		requestBody, err := json.Marshal(BatchEventsBody{Batch: payload.Records})
		if err != nil {
			errorf("Failed to marshal batch events body: %s", err)
			reportPayloadFailure(e.localBucketing, payload.PayloadId, false)
			continue
		}
		req, err = http.NewRequest("POST", eventsHost+"/v1/events/batch", bytes.NewReader(requestBody))
		if err != nil {
			errorf("Failed to create request to events api: %s", err)
			reportPayloadFailure(e.localBucketing, payload.PayloadId, false)
			continue
		}

		req.Header.Set("Authorization", e.localBucketing.sdkKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = e.httpClient.Do(req)

		if err != nil {
			errorf("Failed to make request to events api: %s", err)
			_ = reportPayloadFailure(e.localBucketing, payload.PayloadId, false)
			continue
		}

		if resp.StatusCode >= 500 {
			warnf("Events API Returned a 5xx error, retrying later.")
			_ = reportPayloadFailure(e.localBucketing, payload.PayloadId, true)
			continue
		}

		if resp.StatusCode >= 400 {
			_ = reportPayloadFailure(e.localBucketing, payload.PayloadId, false)
			responseBody, readError := io.ReadAll(resp.Body)
			if readError != nil {
				errorf("Failed to read response body %s", readError)
				continue
			}
			resp.Body.Close()

			errorf("Error sending events - Response: %s", string(responseBody))

			continue
		}

		if resp.StatusCode == 201 {
			err = e.localBucketing.onPayloadSuccess(payload.PayloadId)
			if err != nil {
				errorf("failed to mark payload as success %s", err)
				continue
			}
			debugf("Flushed %d events\n", payload.EventCount)
		}
	}
	return err
}

func reportPayloadFailure(localBucketing *DevCycleLocalBucketing, payloadId string, retry bool) (err error) {
	err = localBucketing.onPayloadFailure(payloadId, retry)
	if err != nil {
		errorf("Failed to mark payload as failed: %s", err)
	}
	return
}

func (e *EventQueue) Close() (err error) {
	e.flushStop <- true
	e.closed = true
	err = e.FlushEvents()
	return err
}
