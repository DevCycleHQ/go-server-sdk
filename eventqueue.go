package devcycle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (e *EventQueue) initialize(options *DVCOptions, localBucketing *DevCycleLocalBucketing) error {
	e.httpClient = http.DefaultClient
	e.options = options

	if !e.options.DisableLocalBucketing && localBucketing != nil {
		e.localBucketing = localBucketing
		str, err := json.Marshal(e.eventQueueOptionsFromDVCOptions(options))
		if err != nil {
			return err
		}
		err = e.localBucketing.initEventQueue(string(str))
		return err
	}
	return nil
}

func (e *EventQueue) QueueEvent(user UserData, event DVCEvent) error {
	if q, err := e.checkEventQueueSize(); err != nil || q {
		return fmt.Errorf("event queue is full. Dropping event")
	}
	if !e.options.DisableLocalBucketing {
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

func (e *EventQueue) QueueAggregateEvent(event DVCEvent, bucketedConfig BucketedUserConfig) error {
	if q, err := e.checkEventQueueSize(); err != nil || q {
		return fmt.Errorf("event queue is full. Dropping aggregate event")
	}
	if !e.options.DisableLocalBucketing {
		eventstring, err := json.Marshal(event)
		err = e.localBucketing.queueAggregateEvent(string(eventstring), bucketedConfig)
		return err
	}
	return nil
}

func (e *EventQueue) checkEventQueueSize() (bool, error) {
	queueSize, err := e.localBucketing.checkEventQueueSize()
	if err != nil {
		return false, err
	}
	if queueSize >= e.options.MinEventsPerFlush {
		err = e.FlushEvents()
		if err != nil {
			return false, nil
		}
		if queueSize >= e.options.MaxEventsPerFlush {
			return true, nil
		}
	}
	return false, nil
}

func (e *EventQueue) FlushEvents() error {
	e.localBucketing.startFlushEvents()
	events, err := e.localBucketing.flushEventQueue()
	if err != nil {
		return err
	}

	for _, event := range events {
		var req *http.Request
		var resp *http.Response
		var body []byte
		body, err = json.Marshal(event)
		req, err = http.NewRequest("POST", "https://events.devcycle.com/v1/events/batch", bytes.NewReader(body))

		req.Header.Set("Authorization", e.localBucketing.sdkKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = e.httpClient.Do(req)
		if err != nil {
			err = e.localBucketing.onPayloadFailure(event.PayloadId, resp.StatusCode >= 500 && resp.StatusCode < 600)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println(err)
			continue
		}

		if resp.StatusCode == 201 {
			err = e.localBucketing.onPayloadSuccess(event.PayloadId)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}
	e.localBucketing.finishFlushEvents()
	return nil
}
