package devcycle

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type EventQueue struct {
	localBucketing    *DevCycleLocalBucketing
	options           *DVCOptions
	eventQueue        chan Event
	aggregateQueue    chan Event
	eventQueueOptions *EventQueueOptions
	httpClient        *http.Client
}

type EventQueueOptions struct {
	FlushEventsInterval          time.Duration `json:"flushEventsMS"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging"`
}

func (e *EventQueue) initialize(localBucketing *DevCycleLocalBucketing, options *DVCOptions) error {
	e.httpClient = http.DefaultClient
	e.localBucketing = localBucketing
	e.options = options
	e.eventQueue = make(chan Event, 100)
	e.aggregateQueue = make(chan Event, 100)

	if !e.options.DisableLocalBucketing {
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

func (e *EventQueue) QueueEvent(user UserData, event Event) error {
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

func (e *EventQueue) QueueAggregateEvent(event Event, bucketedConfig BucketedUserConfig) error {
	if !e.options.DisableLocalBucketing {
		eventstring, err := json.Marshal(event)
		err = e.localBucketing.queueAggregateEvent(string(eventstring), bucketedConfig)
		return err
	}
	e.aggregateQueue <- event
	return nil
}

func (e *EventQueue) flushEvents() error {
	if !e.options.DisableLocalBucketing {
		payload, err := e.localBucketing.flushEventQueue()
		if err != nil {
			return err
		}
		for _, p := range payload {
			body := BatchEventsBody{Records: p.Records}
			bodystring, err := json.Marshal(body)
			resp, err := e.eventsPost("", string(bodystring), e.localBucketing.sdkKey)
			if err != nil || resp.StatusCode != 201 {
				// Could not post the event, or the status code is wrong.
				// TODO: Add logging.
				err := e.localBucketing.onPayloadFailure(p.PayloadId, isRetryable(resp))
				if err != nil {
					return err
				}
				continue
			}
			err = e.localBucketing.onPayloadSuccess(p.PayloadId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *EventQueue) eventFlushPolling() {
	ticker := time.NewTicker(e.options.PollingInterval)
	for {
		select {
		case <-ticker.C:
			e.flushEvents()
		}
	}
}

func (e *EventQueue) eventsPost(url string, body, envKey string) (resp *http.Response, err error) {

	bodyBuf := &bytes.Buffer{}
	bodyBuf.WriteString(body)
	req, err := http.NewRequest("POST", url, bodyBuf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", envKey)

	resp, err = e.httpClient.Do(req)
	return
}

func isRetryable(resp *http.Response) bool {
	return resp.StatusCode >= 500 && resp.StatusCode < 600
}
