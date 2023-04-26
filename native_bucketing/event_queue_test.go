package native_bucketing

import (
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEventQueue_MergeAggEventQueueKeys(t *testing.T) {
	// should not panic/error.
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
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
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	eq.MergeAggEventQueueKeys(config)

}

func TestEventQueue_ProcessUserEvent(t *testing.T) {
	event := userEventData{
		event: &api.DVCEvent{
			Type_:      api.EventType_VariableEvaluated,
			Target:     "somevariablekey",
			CustomType: "testingtype",
			UserId:     "testing",
		},
		user: &api.DVCUser{
			UserId: "testing",
		},
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	err = eq.processUserEvent(event)
	require.NoError(t, err)
	fmt.Println(eq.userEventQueue)
}

func TestEventQueue_ProcessAggregateEvent(t *testing.T) {
	event := aggEventData{
		event: &api.DVCEvent{
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
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	err = eq.processAggregateEvent(event)
	require.NoError(t, err)
	fmt.Println(eq.aggEventQueue)
}

func TestEventQueue_AddToUserQueue(t *testing.T) {
	event := api.DVCEvent{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	err = eq.QueueEvent(DVCUser{UserId: "testing"}, event)
	require.NoError(t, err)
	fmt.Println(len(eq.userEventQueueRaw))
	fmt.Println(len(eq.userEventQueue))
}

func TestEventQueue_AddToAggQueue(t *testing.T) {
	event := api.DVCEvent{
		Type_:      api.EventType_VariableEvaluated,
		Target:     "somevariablekey",
		CustomType: "testingtype",
		UserId:     "testing",
	}
	popu := DVCUser{UserId: "testing"}.GetPopulatedUser(platformData)
	err := SetConfig(test_config, "dvc_server_token_hash", "")
	require.NoError(t, err)
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	bucketedConfig, err := GenerateBucketedConfig("dvc_server_token_hash", popu, nil)
	require.NoError(t, err)
	err = eq.QueueAggregateEvent(*bucketedConfig, event)
	require.NoError(t, err)
	fmt.Println(len(eq.aggEventQueueRaw))
	fmt.Println(len(eq.aggEventQueue))
}
