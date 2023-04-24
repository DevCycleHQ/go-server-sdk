package native_bucketing

import (
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEventQueue_MergeAggEventQueueKeys(t *testing.T) {
	eq, err := InitEventQueue("dvc_server_token_hash", &api.EventQueueOptions{})
	require.NoError(t, err)
	// Parsing the large config should succeed without an error
	err = SetConfig(test_config, "test", "")
	require.NoError(t, err)
	config, err := getConfig("test")
	require.NoError(t, err)

	eq.MergeAggEventQueueKeys(config)

}
