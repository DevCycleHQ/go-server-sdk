package devcycle

import "C"
import (
	"encoding/json"
	"github.com/launchdarkly/eventsource"
	"time"
)

type SSEManager struct {
	ConfigManager *EnvironmentConfigManager
	Options       *Options
	Stream        *eventsource.Stream
	eventChannel  chan eventsource.Event
}

type sseMessage struct {
	Etag         string        `json:"etag,omitempty"`
	LastModified time.Duration `json:"lastModified,omitempty"`
	Type_        string        `json:"type,omitempty"`
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
				m.Options.Logger.Warnf("Error unmarshalling sse message: %v", err)
				continue
			}
			/*
			 val innerData = JSONObject(data.get("data") as String)
			            val lastModified = if (innerData.has("lastModified")) {
			                (innerData.get("lastModified") as Long)
			            } else null
			            val type = if (innerData.has("type")) {
			                (innerData.get("type") as String).toLong()
			            } else ""
			            val etag = if (innerData.has("etag")) {
			                (innerData.get("etag") as String)
			            } else null

			            if (type == "refetchConfig" || type == "") { // Refetch the config if theres no type
			                refetchConfig(true, lastModified, etag)
			            }
			*/
			if message.Type_ == "refetchConfig" || message.Type_ == "" {
				err = m.ConfigManager.fetchConfig(CONFIG_RETRIES)
				if err != nil {
					m.Options.Logger.Warnf("Error fetching config: %s\n", err)
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
}
