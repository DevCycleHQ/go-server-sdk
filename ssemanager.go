package devcycle

import (
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/launchdarkly/eventsource"
	"time"
)

type SSEManager struct {
	configManager *EnvironmentConfigManager
	options       *Options
	stream        *eventsource.Stream
	URL           string
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
		configManager: configManager,
		options:       options,
		errorHandler: func(err error) eventsource.StreamErrorHandlerResult {
			util.Warnf("SSE - Error: %v\n", err)
			return eventsource.StreamErrorHandlerResult{
				CloseNow: false,
			}
		},
	}
}

func (m *SSEManager) connectSSE(url string) (err error) {
	sseClientEvent := api.ClientEvent{
		EventType: api.ClientEventType_RealtimeUpdates,
		EventData: "Connected to SSE stream: " + url,
		Status:    "success",
		Error:     nil,
	}

	defer func() {
		if m.options.ClientEventHandler != nil {
			go func() {
				m.options.ClientEventHandler <- sseClientEvent
			}()
		}
	}()
	sse, err := eventsource.SubscribeWithURL(url,
		eventsource.StreamOptionReadTimeout(m.options.AdvancedOptions.RealtimeUpdatesTimeout),
		eventsource.StreamOptionCanRetryFirstConnection(m.options.AdvancedOptions.RealtimeUpdatesTimeout),
		eventsource.StreamOptionErrorHandler(m.errorHandler),
		eventsource.StreamOptionUseBackoff(m.options.AdvancedOptions.RealtimeUpdatesBackoff),
		eventsource.StreamOptionUseJitter(0.25),
		eventsource.StreamOptionHTTPClient(m.configManager.httpClient))

	if err != nil {
		sseClientEvent.Status = "error"
		sseClientEvent.Error = err
		sseClientEvent.EventData = "Error connecting to SSE stream: " + url
		return
	}
	m.stream = sse
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
	//nolint:all
	for {
		select {
		case event, ok := <-m.stream.Events:
			if !ok {
				break
			}
			message, err := m.parseMessage([]byte(event.Data()))
			if err != nil {
				util.Debugf("SSE - Error unmarshalling message: %v\n", err)
				continue
			}
			if message.Type_ == "refetchConfig" || message.Type_ == "" {
				err = m.configManager.fetchConfig(CONFIG_RETRIES)
				if err != nil {
					util.Warnf("SSE - Error fetching config: %v\n", err)
				}
			}
		}
	}
}

func (m *SSEManager) StartSSEOverride(url string) error {
	m.URL = url
	return m.StartSSE()
}

func (m *SSEManager) StartSSE() error {
	if m.URL == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	err := m.connectSSE(m.URL)
	if err != nil {
		return err
	}
	go m.receiveSSEMessages()
	return nil
}
