package devcycle

import (
	_ "embed"
	"golang.org/x/exp/rand"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
)

var (
	//test_environmentKey = "dvc_server_token_hash"

	//go:embed testdata/fixture_small_config.json
	test_config string

	//go:embed testdata/fixture_small_config_special_characters.json
	test_config_special_characters_var string

	//go:embed testdata/fixture_large_config.json
	test_large_config          string
	test_large_config_variable = "v-key-25"

	//go:embed testdata/fixture_small_config_sse.json
	test_small_config_sse string
	test_options          = &Options{
		// use defaults that will be set by the CheckDefaults
		EventFlushIntervalMS:    time.Second * 30,
		ConfigPollingIntervalMS: time.Second * 10,
	}
	test_options_sse = &Options{
		// use defaults that will be set by the CheckDefaults
		EventFlushIntervalMS:    time.Second * 30,
		ConfigPollingIntervalMS: time.Second * 10,
		EnableRealtimeUpdates:   true,
	}
)

func TestMain(t *testing.M) {
	// Remove newlines in configs
	test_config = strings.ReplaceAll(test_config, "\n", "")
	test_small_config_sse = strings.ReplaceAll(test_small_config_sse, "\n", "")
	test_config_special_characters_var = strings.ReplaceAll(test_config_special_characters_var, "\n", "")
	test_large_config = strings.ReplaceAll(test_large_config, "\n", "")

	// Set default options
	test_options.CheckDefaults()
	test_options_sse.CheckDefaults()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
}

func httpBucketingAPIMock() {
	httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test",
		func(req *http.Request) (*http.Response, error) {

			resp := httpmock.NewStringResponse(200, `{"value": true, "_id": "614ef6ea475129459160721a", "key": "test", "type": "Boolean"}`)
			resp.Header.Set("Etag", "TESTING")
			resp.Header.Set("Last-Modified", "LAST-MODIFIED")
			return resp, nil
		},
	)
}

func httpEventsApiMock() {
	httpmock.RegisterResponder("POST", "https://events.devcycle.com/v1/events/batch",
		httpmock.NewStringResponder(201, `{}`))
}

func httpConfigMock(respcode int) (sdkKey string, responder httpmock.Responder) {
	sdkKey = generateTestSDKKey()
	responder = httpCustomConfigMock(sdkKey, respcode, test_config)
	return
}

func httpCustomConfigMock(sdkKey string, respcode int, config string) httpmock.Responder {
	responder := func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(respcode, config)
		resp.Header.Set("Etag", "TESTING")
		resp.Header.Set("Last-Modified", "LAST-MODIFIED")
		return resp, nil
	}
	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+sdkKey+".json", responder)
	return responder
}

func httpSSEConfigMock(respCode int, sdkKeys ...string) (sdkKey string, responder httpmock.Responder) {
	if len(sdkKeys) == 0 {
		sdkKey = generateTestSDKKey()
	} else {
		sdkKey = sdkKeys[0]
	}
	responder = httpCustomConfigMock(sdkKey, respCode, test_small_config_sse)
	return
}

func sseResponseBody() string {
	timestamp := strconv.FormatInt(time.Now().Add(time.Second*-2).UnixMilli(), 10)
	return `{
				"id":"S!e7drup1fABYuqU54493238:^e7d7zTfiQBYtpH28211230@1708618753666-0^mWw1Zg",
				"event":"message",
				"data":{
					"id":"WYc5JQA38b07:0",
					"timestamp":` + timestamp + `,
					"channel":"dvc_server_hashed_token_v1",
					"data":"{\"etag\":\"\\\"1d0be8bbc8e607590b11131237d608c0\\\"\",\"lastModified\":` + timestamp + `}",
					"name":"change"
				}
			}`
}

func httpSSEConnectionMock() {
	httpmock.RegisterResponder("GET", "https://sse.devcycle.com/v1/sse",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, sseResponseBody())
			resp.Header.Set("Content-Type", "text/event-stream")

			return resp, nil
		},
	)
}

func generateTestSDKKey() string {
	return "dvc_server_TESTING" + strconv.FormatInt(rand.Int63(), 10) + "_hash"
}
