package devcycle

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
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

func (e *EventQueue) FlushEvents(ctx context.Context, doReq func(request *http.Request) (*http.Response, error)) error {
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
		req, err = http.NewRequestWithContext(ctx, "POST", "https://events.devcycle.com/v1/events/batch", bytes.NewReader(body))

		req.Header.Set("Authorization", ctx.Value("APIKey").(string))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = doReq(req)
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
