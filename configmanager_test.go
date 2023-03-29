package devcycle

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
)

type recordingConfigReceiver struct {
	configureCount int
}

func (r *recordingConfigReceiver) StoreConfig([]byte) error {
	r.configureCount++
	return nil
}

func TestEnvironmentConfigManager_fetchConfig_success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, test_config)
			resp.Header.Set("Etag", "TESTING")
			return resp, nil
		},
	)

	localBucketing, bucketingPool := &recordingConfigReceiver{}, &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, bucketingPool, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}

	if localBucketing.configureCount != 1 {
		t.Fatal("localBucketing.configureCount != 1")
	}
	if bucketingPool.configureCount != 1 {
		t.Fatal("bucketingPool.configureCount != 1")
	}
	if !manager.hasConfig.Load() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.configETag != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
}

func TestEnvironmentConfigManager_fetchConfig_retries500(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	successResponse := httpConfigMock(200)
	errorResponse := httpmock.NewStringResponder(http.StatusInternalServerError, "Internal Server Error")

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		errorResponse.Then(successResponse),
	)

	localBucketing, bucketingPool := &recordingConfigReceiver{}, &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, bucketingPool, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}
	if !manager.hasConfig.Load() {
		t.Fatal("cm.hasConfig != true")
	}
}
