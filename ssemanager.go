package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/launchdarkly/eventsource"
	"time"
)

type SSEManager struct {
	configManager    *EnvironmentConfigManager
	options          *Options
	stream           *eventsource.Stream
	eventChannel     chan eventsource.Event
	url              string
	errorHandler     eventsource.StreamErrorHandler
	context          context.Context
	stopEventHandler context.CancelFunc
	cfg              *HTTPConfiguration
	Started          bool
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

func newSSEManager(configManager *EnvironmentConfigManager, options *Options, cfg *HTTPConfiguration) *SSEManager {
	if options == nil {
		options = &Options{}
		options.CheckDefaults()
	}
	sseManager := &SSEManager{
		configManager: configManager,
		options:       options,
		errorHandler: func(err error) eventsource.StreamErrorHandlerResult {
			util.Warnf("SSE - Error: %v\n", err)
			return eventsource.StreamErrorHandlerResult{
				CloseNow: false,
			}
		},
		cfg: cfg,
	}
	sseManager.context, sseManager.stopEventHandler = context.WithCancel(context.Background())

	return sseManager
}

func (m *SSEManager) connectSSE(url string) (err error) {
	if m.stream != nil {
		m.stream.Close()
	}
	sseClientEvent := api.ClientEvent{
		EventType: api.ClientEventType_InternalSSEConnected,
		EventData: "Connected to SSE stream: " + url,
		Status:    "success",
		Error:     nil,
	}

	defer func() {
		m.configManager.InternalClientEvents <- sseClientEvent
	}()
	sse, err := eventsource.SubscribeWithURL(url,
		eventsource.StreamOptionReadTimeout(m.options.AdvancedOptions.RealtimeUpdatesTimeout),
		eventsource.StreamOptionCanRetryFirstConnection(m.options.AdvancedOptions.RealtimeUpdatesTimeout),
		eventsource.StreamOptionErrorHandler(m.errorHandler),
		eventsource.StreamOptionUseBackoff(m.options.AdvancedOptions.RealtimeUpdatesBackoff),
		eventsource.StreamOptionUseJitter(0.25),
		eventsource.StreamOptionHTTPClient(m.cfg.HTTPClient))
	if err != nil {
		sseClientEvent.EventType = api.ClientEventType_InternalSSEFailure
		sseClientEvent.Status = "failure"
		sseClientEvent.Error = err
		sseClientEvent.EventData = "Error connecting to SSE stream: " + url
		return
	}
	if m.stream != nil {
		m.stream.Close()
	}
	m.stream = sse
	m.eventChannel = m.stream.Events
	m.Started = sseClientEvent.Error == nil
	go m.receiveSSEMessages()
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
		if m.stream == nil || m.context.Err() != nil {
			return
		}
		err := func() error {
			select {
			case <-m.context.Done():
				return fmt.Errorf("SSE - Stopping SSE polling")
			case event, ok := <-m.eventChannel:
				if !ok {
					return nil
				}
				message, err := m.parseMessage([]byte(event.Data()))
				if err != nil {
					util.Debugf("SSE - Error unmarshalling message: %v\n", err)
					return nil
				}
				if message.Type_ == "refetchConfig" || message.Type_ == "" {
					m.configManager.InternalClientEvents <- api.ClientEvent{
						EventType: api.ClientEventType_InternalNewConfigAvailable,
						EventData: time.UnixMilli(int64(message.LastModified)),
						Status:    "",
						Error:     nil,
					}
				}
			}
			return nil
		}()
		if err != nil {
			return
		}
	}
}

func (m *SSEManager) StartSSEOverride(url string) error {
	m.url = url
	return m.connectSSE(url)
}

func (m *SSEManager) StopSSE() {
	if m.stream != nil {
		m.stream.Close()
		m.stopEventHandler()
	}
}
