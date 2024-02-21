package devcycle

import (
	"github.com/launchdarkly/eventsource"
)

type SSEManager struct {
	Options *Options
	Stream  *eventsource.Stream
}

func (m *SSEManager) connectSSE(url string) (err error) {
	sse, err := eventsource.SubscribeWithURL(url,
		eventsource.StreamOptionReadTimeout(m.Options.AdvancedOptions.ServerSentEventsTimeout),
		eventsource.StreamOptionCanRetryFirstConnection(m.Options.AdvancedOptions.ServerSentEventsTimeout))
	if err != nil {
		return
	}
	m.Stream = sse
	return
}
func (m *SSEManager) receiveSSEMessage() {

}
