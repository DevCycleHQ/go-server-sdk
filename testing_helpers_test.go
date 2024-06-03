package devcycle

import (
	_ "embed"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
)

var (
	//go:embed testdata/fixture_small_config.json
	test_config string

	//go:embed testdata/fixture_small_config_special_characters.json
	test_config_special_characters_var string

	//go:embed testdata/fixture_large_config.json
	test_large_config          string
	test_large_config_variable = "v-key-25"

	test_options = &Options{
		// use defaults that will be set by the CheckDefaults
		EventFlushIntervalMS:    time.Second * 30,
		ConfigPollingIntervalMS: time.Second * 10,
	}
	test_options_sse = &Options{
		// use defaults that will be set by the CheckDefaults
		EventFlushIntervalMS:    time.Second * 30,
		ConfigPollingIntervalMS: time.Second * 10,
	}
	benchmarkEnableEvents        bool
	benchmarkEnableConfigUpdates bool
	benchmarkDisableLogs         bool
)

func TestMain(t *testing.M) {
	httpmock.Activate()
	flag.BoolVar(&benchmarkEnableEvents, "benchEnableEvents", false, "Custom test flag that enables event logging in benchmarks")
	flag.BoolVar(&benchmarkEnableConfigUpdates, "benchEnableConfigUpdates", false, "Custom test flag that enables config updates in benchmarks")
	flag.BoolVar(&benchmarkDisableLogs, "benchDisableLogs", false, "Custom test flag that disables logging in benchmarks")
	flag.Parse()

	rand.NewSource(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Remove newlines in configs
	test_config = strings.ReplaceAll(test_config, "\n", "")
	test_config_special_characters_var = strings.ReplaceAll(test_config_special_characters_var, "\n", "")
	test_large_config = strings.ReplaceAll(test_large_config, "\n", "")

	// Set default options
	test_options.CheckDefaults()
	test_options_sse.CheckDefaults()
	httpBucketingAPIMock()
	httpEventsApiMock()

	os.Exit(t.Run())
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
		resp.Header.Set("Cf-Ray", "TESTING")
		return resp, nil
	}
	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+sdkKey+".json", responder)
	return responder
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

func fatalErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
