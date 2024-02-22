package devcycle

import "C"
import (
	"encoding/json"
	"github.com/launchdarkly/eventsource"
)

type SSEManager struct {
	ConfigManager *EnvironmentConfigManager
	Options       *Options
	Stream        *eventsource.Stream
	eventChannel  chan eventsource.Event
}

type sseMessage struct {
	Etag         string  `json:"etag,omitempty"`
	LastModified float64 `json:"lastModified,omitempty"`
	Type_        string  `json:"type,omitempty"`
}

func newSSEManager(configManager *EnvironmentConfigManager, options *Options) *SSEManager {
	return &SSEManager{
		ConfigManager: configManager,
		Options:       options,
	}
}

func (m *SSEManager) connectSSE(url string) (err error) {
	sse, err := eventsource.SubscribeWithURL(url,
		eventsource.StreamOptionReadTimeout(m.Options.AdvancedOptions.ServerSentEventsTimeout),
		eventsource.StreamOptionCanRetryFirstConnection(m.Options.AdvancedOptions.ServerSentEventsTimeout))
	if err != nil {
		return
	}
	m.Stream = sse
	m.eventChannel = make(chan eventsource.Event, m.Options.AdvancedOptions.ServerSentEventsQueueSize)
	sse.Events = m.eventChannel
	return
}

func (m *SSEManager) receiveSSEMessages() {
	for {
		select {
		case event := <-m.eventChannel:
			var message sseMessage
			err := json.Unmarshal([]byte(event.Data()), &message)
			if err != nil {
				m.Options.Logger.Warnf("SSE - Error unmarshalling message: %v\n", err)
				continue
			}
			if message.Type_ == "refetchConfig" || message.Type_ == "" {
				err = m.ConfigManager.fetchConfig(CONFIG_RETRIES)
				if err != nil {
					m.Options.Logger.Warnf("SSE - Error fetching config: %v\n", err)
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
	m.receiveSSEMessages()
	return nil
}
