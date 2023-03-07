package devcycle

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/jarcoal/httpmock"
	"bytes"
)

var (
	test_environmentKey = "dvc_server_token_hash"

	//go:embed testdata/fixture_small_config.json
	test_config []byte

	//go:embed testdata/fixture_large_config.json
	test_large_config          string
	test_large_config_variable = "v-key-25"
)

func init() {
	// Remove newlines in configs
	test_config = bytes.ReplaceAll(test_config, []byte("\n"), []byte(""))
	test_large_config = strings.ReplaceAll(test_large_config, "\n", "")
}

func httpBucketingAPIMock() {
	httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test",
		func(req *http.Request) (*http.Response, error) {

			resp := httpmock.NewStringResponse(200, `{"value": true, "_id": "614ef6ea475129459160721a", "key": "test", "type": "Boolean"}`)
			resp.Header.Set("Etag", "TESTING")
			return resp, nil
		},
	)
}

func httpEventsApiMock() {
	httpmock.RegisterResponder("POST", "https://events.devcycle.com/v1/events/batch",
		httpmock.NewStringResponder(201, `{}`))
}

func httpConfigMock(respcode int) {
	httpCustomConfigMock(test_environmentKey, respcode, string(test_config))
}

func httpCustomConfigMock(sdkKey string, respcode int, config string) {
	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+sdkKey+".json",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(respcode, config)
			resp.Header.Set("Etag", "TESTING")
			return resp, nil
		},
	)
}
