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
	cfg               *HTTPConfiguration
	context           context.Context
	closed            bool
	ticker            *time.Ticker
	flushStop         chan bool
	eventsChan        chan []FlushPayload
}

func (e *EventQueue) initialize(eventsChan chan []FlushPayload, options *DVCOptions, localBucketing *DevCycleLocalBucketing, cfg *HTTPConfiguration) (err error) {
	e.context = context.Background()
	e.cfg = cfg
	e.options = options
	e.flushStop = make(chan bool, 1)
	e.eventsChan = eventsChan

	if !e.options.EnableCloudBucketing && localBucketing != nil {
		e.localBucketing = localBucketing
		var eventQueueOpt []byte
		eventQueueOpt, err = json.Marshal(options.eventQueueOptions())
		if err != nil {
			return err
		}
		err = e.localBucketing.initEventQueue(eventQueueOpt)
		ticker := time.NewTicker(e.options.EventFlushIntervalMS)

		go func() {
			for {
				select {
				case payloads := <-eventsChan:
					err = e.flushEventPayloads(payloads, true)
					if err != nil {
						warnf("Error flushing worker events: %s\n", err)
					}
				case <-ticker.C:
					err = e.FlushEvents()
					if err != nil {
						warnf("Error flushing primary events queue: %s\n", err)
					}
				case <-e.flushStop:
					ticker.Stop()
					infof("Stopping event flushing.")
					return
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

func (e *EventQueue) QueueAggregateEvent(config BucketedUserConfig, event DVCEvent) error {
	if q, err := e.checkEventQueueSize(); err != nil || q {
		return errorf("Max event queue size reached, dropping aggregate event")
	}
	if !e.options.EnableCloudBucketing {
		eventstring, err := json.Marshal(event)
		err = e.localBucketing.queueAggregateEvent(string(eventstring), config)
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
	e.localBucketing.startFlushEvents()
	defer e.localBucketing.finishFlushEvents()
	payloads, err := e.localBucketing.flushEventQueue()
	if err != nil {
		return err
	}

	return e.flushEventPayloads(payloads, false)
}

func (e *EventQueue) flushEventPayloads(payloads []FlushPayload, useChannel bool) (err error) {
	eventsHost := e.cfg.EventsAPIBasePath
	for _, payload := range payloads {
		var req *http.Request
		var resp *http.Response
		requestBody, err := json.Marshal(BatchEventsBody{Batch: payload.Records})
		if err != nil {
			_ = errorf("Failed to marshal batch events body: %s", err)
			e.reportPayloadFailure(payload, false, useChannel)
			continue
		}
		req, err = http.NewRequest("POST", eventsHost+"/v1/events/batch", bytes.NewReader(requestBody))
		if err != nil {
			_ = errorf("Failed to create request to events api: %s", err)
			e.reportPayloadFailure(payload, false, useChannel)
			continue
		}

		req.Header.Set("Authorization", e.localBucketing.sdkKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = e.cfg.HTTPClient.Do(req)

		if err != nil {
			_ = errorf("Failed to make request to events api: %s", err)
			e.reportPayloadFailure(payload, false, useChannel)
			continue
		}

		if resp.StatusCode >= 500 {
			warnf("Events API Returned a 5xx error, retrying later.")
			e.reportPayloadFailure(payload, true, useChannel)
			continue
		}

		if resp.StatusCode >= 400 {
			e.reportPayloadFailure(payload, false, useChannel)
			responseBody, readError := io.ReadAll(resp.Body)
			if readError != nil {
				_ = errorf("Failed to read response body %s", readError)
				continue
			}
			resp.Body.Close()

			_ = errorf("Error sending events - Response: %s", string(responseBody))

			continue
		}

		if resp.StatusCode == 201 {
			err = e.reportPayloadSuccess(payload, useChannel)
			if err != nil {
				_ = errorf("failed to mark payload as success %s", err)
				continue
			}
		}
	}
	return err
}

func (e *EventQueue) reportPayloadSuccess(payload FlushPayload, useChannel bool) (err error) {
	if useChannel {
		// No need to do anything here, the message has been removed from the channel already
		return nil
	}
	err = e.localBucketing.onPayloadSuccess(payload.PayloadId)
	if err != nil {
		_ = errorf("Failed to mark payload as failed: %s", err)
	}
	return
}

func (e *EventQueue) reportPayloadFailure(payload FlushPayload, retry bool, useChannel bool) {
	if useChannel {
		// Feed failed payload back into channel for re-processing
		// TODO: This needs to respect the maximum queue size
		e.eventsChan <- []FlushPayload{payload}
		return
	}
	err := e.localBucketing.onPayloadFailure(payload.PayloadId, retry)
	if err != nil {
		_ = errorf("Failed to mark payload as failed: %s", err)
	}
	return
}

func (e *EventQueue) Close() (err error) {
	e.flushStop <- true
	e.closed = true
	err = e.FlushEvents()
	return err
}
