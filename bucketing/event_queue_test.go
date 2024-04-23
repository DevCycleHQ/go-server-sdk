package bucketing

import (
	"fmt"
	"testing"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/stretchr/testify/require"
)

func BenchmarkEventQueue_QueueEvent(b *testing.B) {
	util.SetLogger(util.DiscardLogger{})
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(b, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{MaxEventQueueSize: b.N + 10}, (&api.PlatformData{}).Default())
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eq.QueueEvent(api.User{UserId: "testing"}, event)
	}
	b.StopTimer()
}

func BenchmarkEventQueue_QueueEvent_WithDrop(b *testing.B) {
	util.SetLogger(util.DiscardLogger{})
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(b, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{MaxEventQueueSize: b.N / 2}, (&api.PlatformData{}).Default())
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eq.QueueEvent(api.User{UserId: "testing"}, event)
	}
	b.StopTimer()
}
func TestEventQueue_MergeAggEventQueueKeys(t *testing.T) {
	// should not panic/error.
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	// Parsing the large config should succeed without an error
	err = SetConfig(test_config, "test", "", "")
	require.NoError(t, err)
	config, err := getConfig("test")
	require.NoError(t, err)

	eq.MergeAggEventQueueKeys(config)
}

func TestEventQueue_FlushEvents(t *testing.T) {
	// Test flush events by ensuring that all events are flushed
	// and that the number of events flushed is equal to the number
	// of events reported.
	err := SetConfig(test_config, "test", "", "")
	require.NoError(t, err)
	config, err := getConfig("test")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	eq.MergeAggEventQueueKeys(config)

}

func TestEventQueue_ProcessUserEvent(t *testing.T) {
	event := userEventData{
		event: &api.Event{
			Type_:      api.EventType_VariableEvaluated,
			Target:     "somevariablekey",
			CustomType: "testingtype",
			UserId:     "testing",
		},
		user: &api.User{
			UserId: "testing",
		},
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	err = eq.processUserEvent(event)
	require.NoError(t, err)
}

func TestEventQueue_ProcessAggregateEvent(t *testing.T) {
	event := aggEventData{
		eventType:   api.EventType_AggVariableEvaluated,
		variableKey: "somevariablekey",
		featureId:   "featurekey",
		variationId: "somevariation",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	err = eq.processAggregateEvent(event)
	require.NoError(t, err)
}

func TestEventQueue_AddToUserQueue(t *testing.T) {
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	err = eq.QueueEvent(api.User{UserId: "testing"}, event)
	require.NoError(t, err)
}

func TestEventQueue_AddToAggQueue(t *testing.T) {
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{FlushEventsInterval: time.Hour}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	err = eq.QueueVariableEvaluatedEvent("somevariablekey", "featureId", "variationId")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return eq.aggQueueLength() == 1 }, 10*time.Second, time.Millisecond)
}

func TestEventQueue_UserMaxQueueDrop(t *testing.T) {
	user := api.User{UserId: "testing"}
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
	}, (&api.PlatformData{}).Default())
	require.NoError(t, err)
	// Replace the user event queue with a channel that can only hold 3 events
	eq.userEventQueueRaw = make(chan userEventData, 3)
	hasErrored := false
	for i := 0; i <= 3; i++ {
		event.Target = fmt.Sprintf("somevariablekey%d", i)
		err = eq.QueueEvent(user, event)
		if err != nil {
			hasErrored = true
			break
		}
	}
	require.True(t, hasErrored)
	require.Errorf(t, err, "dropping")
}

func TestEventQueue_QueueAndFlush(t *testing.T) {
	user := api.User{UserId: "testing"}
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{
		FlushEventsInterval: time.Hour,
	}, api.PlatformData{}.Default())
	require.NoError(t, err)
	hasErrored := false
	for i := 0; i < 2; i++ {
		event.Target = fmt.Sprintf("somevariablekey%d", i)
		user.UserId = fmt.Sprintf("testing%d", i)
		err = eq.QueueEvent(user, event)
		if err != nil {
			hasErrored = true
			break
		}
	}
	require.False(t, hasErrored)
	require.NoError(t, err)

	// Wait for the events to progress through the background worker
	require.Eventually(t, func() bool { return eq.UserQueueLength() == 2 }, 10*time.Second, time.Millisecond)

	require.Equal(t, 2, len(eq.userEventQueue))
	require.Equal(t, 0, len(eq.userEventQueueRaw))

	payloads, err := eq.FlushEventQueue("", "", "")
	require.NoError(t, err)
	require.Equal(t, 2, len(payloads))
	require.Equal(t, 0, len(eq.userEventQueue))
	require.Equal(t, 2, len(eq.pendingPayloads))
}
