package devcycle

import (
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
