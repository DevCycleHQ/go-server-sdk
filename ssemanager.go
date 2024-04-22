package devcycle

import "C"
import (
	"encoding/json"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/launchdarkly/eventsource"
	"time"
)

type SSEManager struct {
	ConfigManager *EnvironmentConfigManager
	Options       *Options
	Stream        *eventsource.Stream
	eventChannel  chan eventsource.Event
	errorHandler  eventsource.StreamErrorHandler
}

type sseEvent struct {
	Id        string  `json:"id"`
	Timestamp float64 `json:"timestamp"`
	Channel   string  `json:"channel"`
	Data      string  `json:"data"`
	Name      string  `json:"name"`
}
type sseMessage struct {
	Etag         string  `json:"etag,omitempty"`
	LastModified float64 `json:"lastModified,omitempty"`
	Type_        string  `json:"type,omitempty"`
}

func (m *sseMessage) LastModifiedDuration() time.Duration {
	return time.Duration(m.LastModified) * time.Millisecond
}

func newSSEManager(configManager *EnvironmentConfigManager, options *Options) *SSEManager {
	if options == nil {
		options = &Options{}
		options.CheckDefaults()
	}
	return &SSEManager{
		ConfigManager: configManager,
		Options:       options,
		errorHandler: func(err error) eventsource.StreamErrorHandlerResult {
			util.Warnf("SSE - Error: %v\n", err)
			return eventsource.StreamErrorHandlerResult{
				CloseNow: false,
			}
		},
	}
}

func (m *SSEManager) connectSSE(url string) (err error) {
	sse, err := eventsource.SubscribeWithURL(url,
		eventsource.StreamOptionReadTimeout(m.Options.AdvancedOptions.ServerSentEventsTimeout),
		eventsource.StreamOptionCanRetryFirstConnection(m.Options.AdvancedOptions.ServerSentEventsTimeout),
		eventsource.StreamOptionErrorHandler(m.errorHandler),
		eventsource.StreamOptionUseBackoff(m.Options.AdvancedOptions.ServerSentEventsBackoff),
		eventsource.StreamOptionUseJitter(0.25),
		eventsource.StreamOptionHTTPClient(m.ConfigManager.httpClient))

	if err != nil {
		return
	}
	m.Stream = sse
	m.eventChannel = sse.Events

	return
}

func (m *SSEManager) parseMessage(rawMessage []byte) (message sseMessage, err error) {
	event := sseEvent{}
	err = json.Unmarshal(rawMessage, &event)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(event.Data), &message)
	return
}

func (m *SSEManager) receiveSSEMessages() {
	for {
		select {
		case event, ok := <-m.Stream.Events:
			if !ok {
				break
			}
			message, err := m.parseMessage([]byte(event.Data()))
			if err != nil {
				util.Debugf("SSE - Error unmarshalling message: %v\n", err)
				continue
			}
			if message.Type_ == "refetchConfig" || message.Type_ == "" {
				err = m.ConfigManager.fetchConfig(CONFIG_RETRIES)
				if err != nil {
					util.Warnf("SSE - Error fetching config: %v\n", err)
				}
			}
		}
	}
}

func (m *SSEManager) StartSSE() error {
	err := m.connectSSE(m.Options.AdvancedOptions.ServerSentEventsURI)
	if err != nil {
		return err
	}
	go m.receiveSSEMessages()
	return nil
}
