package devcycle

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
)

type recordingConfigReceiver struct {
	configureCount int
}

func (r *recordingConfigReceiver) StoreConfig([]byte, string) error {
	r.configureCount++
	return nil
}

func TestEnvironmentConfigManager_fetchConfig_success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpConfigMock(200)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}

	if localBucketing.configureCount != 1 {
		t.Fatal("localBucketing.configureCount != 1")
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

	error500Response := httpmock.NewStringResponder(http.StatusInternalServerError, "Internal Server Error")

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		errorResponseChain(error500Response, CONFIG_RETRIES),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}
	if !manager.hasConfig.Load() {
		t.Fatal("cm.hasConfig != true")
	}
}

func TestEnvironmentConfigManager_fetchConfig_retries_errors(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		errorResponseChain(connectionErrorResponse, CONFIG_RETRIES),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}
	if !manager.hasConfig.Load() {
		t.Fatal("cm.hasConfig != true")
	}
}

func TestEnvironmentConfigManager_fetchConfig_returns_errors(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		errorResponseChain(connectionErrorResponse, CONFIG_RETRIES+1),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func errorResponseChain(errorResponse httpmock.Responder, count int) httpmock.Responder {
	successResponse := httpConfigMock(200)
	response := errorResponse
	for i := 1; i < count; i++ {
		response = response.Then(errorResponse)
	}
	response = response.Then(successResponse)
	return response
}
