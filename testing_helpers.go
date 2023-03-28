package devcycle

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/jarcoal/httpmock"
)

var (
	test_environmentKey = "dvc_server_token_hash"

	//go:embed bench/testdata/fixture_small_config.json
	test_config string

	//go:embed bench/testdata/fixture_small_config_special_characters.json
	test_config_special_characters_var string

	//go:embed bench/testdata/fixture_large_config.json
	test_large_config          string
	test_large_config_variable = "v-key-25"
)

func init() {
	// Remove newlines in configs
	test_config = strings.ReplaceAll(test_config, "\n", "")
	test_config_special_characters_var = strings.ReplaceAll(test_config_special_characters_var, "\n", "")
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
	httpCustomConfigMock(test_environmentKey, respcode, test_config)
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
