package native_bucketing

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
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(b, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{MaxEventQueueSize: b.N + 10})
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
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(b, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{MaxEventQueueSize: b.N / 2})
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eq.QueueEvent(api.User{UserId: "testing"}, event)
	}
	b.StopTimer()
}
func TestEventQueue_MergeAggEventQueueKeys(t *testing.T) {
	// should not panic/error.
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	// Parsing the large config should succeed without an error
	err = SetConfig(test_config, "test", "")
	require.NoError(t, err)
	config, err := getConfig("test")
	require.NoError(t, err)

	eq.MergeAggEventQueueKeys(config)
}

func TestEventQueue_FlushEvents(t *testing.T) {
	// Test flush events by ensuring that all events are flushed
	// and that the number of events flushed is equal to the number
	// of events reported.
	err := SetConfig(test_config, "test", "")
	require.NoError(t, err)
	config, err := getConfig("test")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
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
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	err = eq.processUserEvent(event)
	require.NoError(t, err)
}

func TestEventQueue_ProcessAggregateEvent(t *testing.T) {
	event := aggEventData{
		event: &api.Event{
			Type_:      api.EventType_VariableEvaluated,
			Target:     "somevariablekey",
			CustomType: "testingtype",
			UserId:     "testing",
		},
		variableVariationMap: map[string]api.FeatureVariation{
			"somevariablekey": {
				Feature:   "featurekey",
				Variation: "somevariation",
			},
		},
		aggregateByVariation: false,
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
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
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	err = eq.QueueEvent(api.User{UserId: "testing"}, event)
	require.NoError(t, err)
}

func TestEventQueue_AddToAggQueue(t *testing.T) {
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	popu := api.User{UserId: "testing"}.GetPopulatedUser(platformData)
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{FlushEventsInterval: time.Hour})
	require.NoError(t, err)
	bucketedConfig, err := GenerateBucketedConfig("dvc_server_token_hash", popu, nil)
	require.NoError(t, err)
	err = eq.QueueAggregateEvent(*bucketedConfig, event)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(eq.aggEventQueue) == 1 }, 10*time.Second, time.Millisecond)
	require.Equal(t, 1, len(eq.aggEventQueue))
}

func TestEventQueue_UserMaxQueueDrop(t *testing.T) {
	user := api.User{UserId: "testing"}
	event := api.Event{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	// Replace inbound event queue so nothing will read from the other side
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
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := NewEventQueue("dvc_server_token_hash", &api.EventQueueOptions{
		FlushEventsInterval: time.Hour,
	})
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
	require.Eventually(t, func() bool { return len(eq.userEventQueue) == 2 }, 10*time.Second, time.Millisecond)
	require.Equal(t, 2, len(eq.userEventQueue))
	require.Equal(t, 0, len(eq.userEventQueueRaw))

	payloads, err := eq.FlushEventQueue()
	require.NoError(t, err)
	require.Equal(t, 2, len(payloads))
	require.Equal(t, 0, len(eq.userEventQueue))
	require.Equal(t, 2, len(eq.pendingPayloads))
}
