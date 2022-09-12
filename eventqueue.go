package devcycle

import (
	"encoding/json"
	"net/http"
)

var ()

func (e *EventQueue) initialize(options *DVCOptions, localBucketing *DevCycleLocalBucketing) error {
	e.httpClient = http.DefaultClient
	e.options = options

	if !e.options.DisableLocalBucketing && localBucketing != nil {
		e.localBucketing = localBucketing
		str, err := json.Marshal(e.eventQueueOptions)
		if err != nil {
			return err
		}
		err = e.localBucketing.initEventQueue(string(str))
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EventQueue) QueueEvent(user UserData, event DVCEvent) error {
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
	e.eventQueue <- event
	return nil
}

func (e *EventQueue) QueueAggregateEvent(event DVCEvent, bucketedConfig BucketedUserConfig) error {
	if !e.options.DisableLocalBucketing {
		eventstring, err := json.Marshal(event)
		err = e.localBucketing.queueAggregateEvent(string(eventstring), bucketedConfig)
		return err
	}
	e.aggregateQueue <- event
	return nil
}

func isRetryable(resp *http.Response) bool {
	return resp.StatusCode >= 500 && resp.StatusCode < 600
}
