package devcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/launchdarkly/eventsource"
	"sync/atomic"
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
	Connected        atomic.Bool
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

func newSSEManager(configManager *EnvironmentConfigManager, options *Options, cfg *HTTPConfiguration) (*SSEManager, error) {
	if options == nil {
		return nil, fmt.Errorf("SSE - Options cannot be nil")
	}
	sseManager := &SSEManager{
		configManager: configManager,
		options:       options,
		errorHandler: func(err error) eventsource.StreamErrorHandlerResult {
			util.Debugf("SSE - Error: %v\n", err)
			return eventsource.StreamErrorHandlerResult{
				CloseNow: false,
			}
		},
		cfg: cfg,
	}
	sseManager.Connected.Store(false)

	sseManager.context, sseManager.stopEventHandler = context.WithCancel(context.Background())

	return sseManager, nil
}

func (m *SSEManager) connectSSE(url string) (err error) {
	// A stream is mutex locked - so we need to make sure we close it before we open a new one
	// This is to prevent multiple streams from being opened, and to prevent race conditions on accessing/reading from
	// the event stream
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
		eventsource.StreamOptionCanRetryFirstConnection(m.options.RequestTimeout),
		eventsource.StreamOptionErrorHandler(m.errorHandler),
		eventsource.StreamOptionUseBackoff(m.options.RequestTimeout),
		eventsource.StreamOptionUseJitter(0.25),
		eventsource.StreamOptionHTTPClient(m.cfg.HTTPClient))
	if err != nil {
		sseClientEvent.EventType = api.ClientEventType_InternalSSEFailure
		sseClientEvent.Status = "failure"
		sseClientEvent.Error = err
		sseClientEvent.EventData = "Error connecting to SSE stream: " + url
		return
	}
	m.Connected.Store(true)
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
		// If the stream is killed/stopped - we should stop polling
		if m.stream == nil || m.context.Err() != nil {
			m.Connected.Store(false)
			m.configManager.InternalClientEvents <- api.ClientEvent{
				EventType: api.ClientEventType_InternalSSEFailure,
				EventData: "SSE stream has been stopped",
				Status:    "failure",
				Error:     m.context.Err(),
			}
			return
		}
		err := func() error {
			select {
			case <-m.context.Done():
				m.Connected.Store(false)
				return fmt.Errorf("SSE - Stopping SSE polling")
			case event, ok := <-m.eventChannel:
				if !ok {
					return nil
				}

				if m.options.ClientEventHandler != nil {
					go func() {
						m.options.ClientEventHandler <- api.ClientEvent{
							EventType: api.ClientEventType_RealtimeUpdates,
							EventData: event,
							Status:    "info",
							Error:     nil,
						}
					}()
				}
				message, err := m.parseMessage([]byte(event.Data()))
				if err != nil {
					util.Debugf("SSE - Error unmarshalling message: %v\n", err)
					return nil
				}
				if message.Type_ == "refetchConfig" || message.Type_ == "" {
					util.Debugf("SSE - Received refetchConfig message: %v\n", message)
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
		// Close wraps `close` and is safe to call in threads - this also just explicitly sets the stream to nil
		m.stream = nil
	}
}

func (m *SSEManager) Close() {
	m.stopEventHandler()
}
