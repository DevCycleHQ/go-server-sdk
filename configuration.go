package devcycle

import (
	"net/http"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

// contextKeys are used to identify the type of value in the context.
// Since these are string, it is possible to get a short description of the
// context key for logging and debugging using key.String().

type EventQueueOptions = api.EventQueueOptions

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	// ContextOAuth2 takes a oauth2.TokenSource as authentication for the request.
	ContextOAuth2 = contextKey("token")

	// ContextBasicAuth takes BasicAuth as authentication for the request.
	ContextBasicAuth = contextKey("basic")

	// ContextAccessToken takes a string oauth2 access token as authentication for the request.
	ContextAccessToken = contextKey("accesstoken")

	// ContextAPIKey takes an APIKey as authentication for the request
	ContextAPIKey = contextKey("apikey")
)

// BasicAuth provides basic http authentication to a request passed via context using ContextBasicAuth
type BasicAuth struct {
	UserName string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
}

// APIKey provides API key based authentication to a request passed via context using ContextAPIKey
type APIKey struct {
	Key    string
	Prefix string
}

type AdvancedOptions struct {
	OverridePlatformData      *api.PlatformData
	ServerSentEventsURI       string
	ServerSentEventsTimeout   time.Duration
	ServerSentEventsQueueSize int
	ServerSentEventsBackoff   time.Duration
}

type Options struct {
	EnableEdgeDB                 bool          `json:"enableEdgeDb,omitempty"`
	EnableCloudBucketing         bool          `json:"enableCloudBucketing,omitempty"`
	EventFlushIntervalMS         time.Duration `json:"eventFlushIntervalMS,omitempty"`
	ConfigPollingIntervalMS      time.Duration `json:"configPollingIntervalMS,omitempty"`
	RequestTimeout               time.Duration `json:"requestTimeout,omitempty"`
	DisableAutomaticEventLogging bool          `json:"disableAutomaticEventLogging,omitempty"`
	DisableCustomEventLogging    bool          `json:"disableCustomEventLogging,omitempty"`
	DisableServerSentEvents      bool          `json:"disableServerSentEvents,omitempty"`
	MaxEventQueueSize            int           `json:"maxEventsPerFlush,omitempty"`
	FlushEventQueueSize          int           `json:"minEventsPerFlush,omitempty"`
	ConfigCDNURI                 string
	EventsAPIURI                 string
	OnInitializedChannel         chan bool
	BucketingAPIURI              string
	Logger                       util.Logger
	AdvancedOptions
}

func (o *Options) eventQueueOptions() *EventQueueOptions {
	return &EventQueueOptions{
		FlushEventsInterval:          o.EventFlushIntervalMS,
		DisableAutomaticEventLogging: o.DisableAutomaticEventLogging,
		DisableCustomEventLogging:    o.DisableCustomEventLogging,
		MaxEventQueueSize:            o.MaxEventQueueSize,
		FlushEventQueueSize:          o.FlushEventQueueSize,
		EventRequestChunkSize:        100, // TODO: make this configurable
		EventsAPIBasePath:            o.EventsAPIURI,
	}
}

func (o *Options) CheckDefaults() {
	if o.ConfigCDNURI == "" {
		o.ConfigCDNURI = "https://config-cdn.devcycle.com"
	}
	if o.EventsAPIURI == "" {
		o.EventsAPIURI = "https://events.devcycle.com"
	}
	if o.BucketingAPIURI == "" {
		o.BucketingAPIURI = "https://bucketing-api.devcycle.com"
	}

	if o.EventFlushIntervalMS < time.Millisecond*500 || o.EventFlushIntervalMS > time.Minute*1 {
		util.Warnf("EventFlushIntervalMS cannot be less than 500ms or longer than 1 minute. Defaulting to 30 seconds.")
		o.EventFlushIntervalMS = time.Second * 30
	}
	if o.ConfigPollingIntervalMS < time.Second*1 {
		util.Warnf("ConfigPollingIntervalMS cannot be less than 1 second. Defaulting to 10 seconds.")
		o.ConfigPollingIntervalMS = time.Second * 10
	}
	if o.RequestTimeout <= time.Second*5 {
		o.RequestTimeout = time.Second * 5
	}
	if o.MaxEventQueueSize <= 0 {
		o.MaxEventQueueSize = 10000
	} else if o.MaxEventQueueSize > 50000 {
		o.MaxEventQueueSize = 50000
	}

	if o.FlushEventQueueSize <= 0 {
		o.FlushEventQueueSize = 1000
	} else if o.FlushEventQueueSize > 50000 {
		o.FlushEventQueueSize = 50000
	}
	if o.AdvancedOptions.ServerSentEventsQueueSize <= 0 {
		o.AdvancedOptions.ServerSentEventsQueueSize = 100
	}
	if o.AdvancedOptions.ServerSentEventsTimeout <= time.Second*5 {
		o.AdvancedOptions.ServerSentEventsTimeout = time.Second * 5
	}
	if o.AdvancedOptions.ServerSentEventsBackoff <= time.Minute*1 {
		o.AdvancedOptions.ServerSentEventsBackoff = time.Minute * 1
	}
}

type HTTPConfiguration struct {
	BasePath          string            `json:"basePath,omitempty"`
	ConfigCDNBasePath string            `json:"configCDNBasePath,omitempty"`
	EventsAPIBasePath string            `json:"eventsAPIBasePath,omitempty"`
	Host              string            `json:"host,omitempty"`
	Scheme            string            `json:"scheme,omitempty"`
	DefaultHeader     map[string]string `json:"defaultHeader,omitempty"`
	UserAgent         string            `json:"userAgent,omitempty"`
	HTTPClient        *http.Client
}

func NewConfiguration(options *Options) *HTTPConfiguration {

	cfg := &HTTPConfiguration{
		BasePath:          options.BucketingAPIURI,
		ConfigCDNBasePath: options.ConfigCDNURI,
		EventsAPIBasePath: options.EventsAPIURI,
		DefaultHeader:     make(map[string]string),
		UserAgent:         "DevCycle-Server-SDK/" + VERSION + "/go",
		HTTPClient: &http.Client{
			// Set an explicit timeout so that we don't wait forever on a request
			Timeout: options.RequestTimeout,
		},
	}
	return cfg
}

func (c *HTTPConfiguration) AddDefaultHeader(key string, value string) {
	c.DefaultHeader[key] = value
}
