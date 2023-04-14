package native_bucketing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetConfig(t *testing.T) {
	err := SetConfig(test_config, "test", "test_etag")
	require.NoError(t, err)

	setConfig, err := getConfig("test")
	baseConfig := configBody{}
	err = json.Unmarshal(test_config, &baseConfig)
	require.NoError(t, err)
	baseConfig.compile("test_etag")

	require.True(t, setConfig.Equals(baseConfig))
}

func TestGetConfig_Unset(t *testing.T) {
	config, err := getConfig("test2")
	require.Error(t, err)
	require.Nil(t, config)
}

func TestGetConfig_Set(t *testing.T) {
	err := SetConfig(test_config, "test3", "test_etag")
	require.NoError(t, err)

	config, err := getConfig("test3")
	require.NoError(t, err)
	require.NotNil(t, config)
}

func TestSetConfigError(t *testing.T) {
	err := SetConfig([]byte(`{"hello": "world"}`), "test", "test_etag")
	require.Error(t, err)
}
