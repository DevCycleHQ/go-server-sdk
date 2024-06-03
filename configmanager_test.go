package devcycle

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
)

type recordingConfigReceiver struct {
	configureCount int
	etag           string
	rayId          string
	lastModified   string
}

func (r *recordingConfigReceiver) StoreConfig(_ []byte, etag, rayId, lastModified string) error {
	r.configureCount++
	r.etag = etag
	r.rayId = rayId
	r.lastModified = lastModified
	return nil
}

func (r *recordingConfigReceiver) HasConfig() bool {
	return r.configureCount > 0
}

func (r *recordingConfigReceiver) GetETag() string {
	return r.etag
}

func (r *recordingConfigReceiver) GetRayId() string {
	return r.rayId
}

func (r *recordingConfigReceiver) GetRawConfig() []byte {
	return nil
}

func (r *recordingConfigReceiver) GetLastModified() string {
	return r.lastModified
}

func TestEnvironmentConfigManager_fetchConfig_success(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}

	if localBucketing.configureCount != 1 {
		t.Fatal("localBucketing.configureCount != 1")
	}
	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.GetETag() != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
}

func TestEnvironmentConfigManager_fetchConfig_retries500(t *testing.T) {
	sdkKey := generateTestSDKKey()

	error500Response := httpmock.NewStringResponder(http.StatusInternalServerError, "Internal Server Error")

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+sdkKey+".json",
		errorResponseChain(sdkKey, error500Response, CONFIG_RETRIES),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}
	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.GetETag() != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
	if manager.GetLastModified() != "LAST-MODIFIED" {
		t.Fatal("cm.lastModified != LAST-MODIFIED")
	}
}

func TestEnvironmentConfigManager_fetchConfig_retries_errors(t *testing.T) {
	sdkKey := generateTestSDKKey()
	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+sdkKey+".json",
		errorResponseChain(sdkKey, connectionErrorResponse, CONFIG_RETRIES),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err != nil {
		t.Fatal(err)
	}
	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.GetETag() != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
	if manager.GetLastModified() != "LAST-MODIFIED" {
		t.Fatal("cm.lastModified != LAST-MODIFIED")
	}
}

func TestEnvironmentConfigManager_fetchConfig_returns_errors(t *testing.T) {
	sdkKey := generateTestSDKKey()

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+sdkKey+".json",
		errorResponseChain(sdkKey, connectionErrorResponse, CONFIG_RETRIES+1),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))

	err := manager.initialFetch()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func errorResponseChain(sdkKey string, errorResponse httpmock.Responder, count int, configMock ...func(respcode int, sdkKeys ...string) (string, httpmock.Responder)) httpmock.Responder {

	var successResponse httpmock.Responder
	if configMock != nil {
		_, successResponse = configMock[0](200, sdkKey)
	} else {
		successResponse = httpCustomConfigMock(sdkKey, 200, test_config)
	}
	response := errorResponse
	for i := 1; i < count; i++ {
		response = response.Then(errorResponse)
	}
	response = response.Then(successResponse)
	return response
}
