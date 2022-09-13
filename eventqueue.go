package devcycle

import (
	"encoding/json"
	"net/http"
)

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
		return err
	}

	e.EventQueue = make(chan DVCEvent, 10000)
	e.AggregateEventQueue = make(chan DVCEvent, 10000)
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
	select {
	case e.EventQueue <- event:
	default:
		break
	}
	return nil
}

func (e *EventQueue) QueueAggregateEvent(event DVCEvent, bucketedConfig BucketedUserConfig) error {
	if !e.options.DisableLocalBucketing {
		eventstring, err := json.Marshal(event)
		err = e.localBucketing.queueAggregateEvent(string(eventstring), bucketedConfig)
		return err
	}
	select {
	case e.AggregateEventQueue <- event:
	default:
		break
	}
	return nil
}
