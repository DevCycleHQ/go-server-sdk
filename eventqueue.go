package devcycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var flushStop = make(chan bool, 1)

type EventQueue struct {
	localBucketing    *DevCycleLocalBucketing
	options           *DVCOptions
	eventQueueOptions *EventQueueOptions
	httpClient        *http.Client
	context           context.Context
	closed            bool
	ticker            *time.Ticker
}

func (e *EventQueue) eventQueueOptionsFromDVCOptions(options *DVCOptions) *EventQueueOptions {
	return &EventQueueOptions{
		FlushEventsInterval:          options.EventsFlushInterval,
		DisableAutomaticEventLogging: options.DisableAutomaticEventLogging,
		DisableCustomEventLogging:    options.DisableCustomEventLogging,
	}
}

type EventQueueOptions struct {
	FlushEventsInterval          time.Duration `json:"flushEventsMS"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging"`
}

func (e *EventQueue) initialize(options *DVCOptions, localBucketing *DevCycleLocalBucketing) error {
	e.context = context.Background()
	e.httpClient = http.DefaultClient
	e.options = options
	if !e.options.EnableCloudBucketing && localBucketing != nil {
		e.localBucketing = localBucketing
		str, err := json.Marshal(e.eventQueueOptionsFromDVCOptions(options))
		if err != nil {
			return err
		}
		err = e.localBucketing.initEventQueue(string(str))
		ticker := time.NewTicker(e.options.EventsFlushInterval)

		go func(ctx context.Context) {
			for {
				select {
				case <-flushStop:
				case <-ctx.Done():
					ticker.Stop()
					log.Println("Stopping event flushing.")
					return
				case <-ticker.C:
					err = e.FlushEvents()
					if err != nil {
						log.Printf("Error flushing events: %s\n", err)
					}
				}
			}
		}(e.context)
		return err
	}
	return nil
}

func (e *EventQueue) QueueEvent(user UserData, event DVCEvent) error {
	if e.closed {
		log.Println("DevCycle client was closed, no more events can be tracked.")
		return fmt.Errorf("DevCycle client was closed, no more events can be tracked.")
	}
	if q, err := e.checkEventQueueSize(); err != nil || q {
		fmt.Println(err)
		log.Println("Max event queue size reached, dropping event")
		return fmt.Errorf("Max event queue size reached, dropping event")
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
		fmt.Println(err)
		log.Println("Max event queue size reached, dropping aggregate event")
		return fmt.Errorf("Max event queue size reached, dropping aggregate event")
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
	eventsHost := e.options.EventsAPIOverride
	e.localBucketing.startFlushEvents()
	defer e.localBucketing.finishFlushEvents()
	events, err := e.localBucketing.flushEventQueue()
	if err != nil {
		return err
	}

	for _, event := range events {
		var req *http.Request
		var resp *http.Response
		var body []byte
		body, err = json.Marshal(event)
		req, err = http.NewRequest("POST", eventsHost+"/v1/events/batch", bytes.NewReader(body))

		req.Header.Set("Authorization", e.localBucketing.sdkKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = e.httpClient.Do(req)
		if err != nil {
			if resp != nil {
				err = e.localBucketing.onPayloadFailure(event.PayloadId, resp.StatusCode >= 500 && resp.StatusCode < 600)
				if err != nil {
					log.Println(err)
					continue
				}
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
			log.Printf("Flushed %d events\n", event.EventCount)
		}
	}
	return err
}

func (e *EventQueue) Close() (err error) {
	flushStop <- true
	e.closed = true
	err = e.FlushEvents()
	return err
}
