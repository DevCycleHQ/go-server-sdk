package devcycle

import (
	"net/http"
	"time"
)

type EventQueue struct {
	localBucketing    *DevCycleLocalBucketing
	options           *DVCOptions
	eventQueueOptions *EventQueueOptions
	httpClient        *http.Client
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
