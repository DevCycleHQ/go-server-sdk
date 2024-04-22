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
	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.GetETag() != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
}

func TestEnvironmentConfigManager_fetchConfig_success_sse(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpSSEConfigMock(200)
	httpSSEConnectionMock()

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options_sse, NewConfiguration(test_options_sse))

	err := manager.StartSSE()
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
	if manager.sseManager == nil {
		t.Fatal("cm.sseManager == nil")
	}
	if manager.sseManager.Stream == nil {
		t.Fatal("cm.sseManager.Stream == nil")
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

func TestEnvironmentConfigManager_fetchConfig_retries_errors_sse(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpSSEConnectionMock()

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		errorResponseChain(connectionErrorResponse, CONFIG_RETRIES, httpSSEConfigMock),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options_sse, NewConfiguration(test_options_sse))

	err := manager.StartSSE()

	if err != nil {
		t.Fatal(err)
	}
	if !manager.HasConfig() {
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

func TestEnvironmentConfigManager_fetchConfig_returns_errors_sse(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/"+test_environmentKey+".json",
		errorResponseChain(connectionErrorResponse, CONFIG_RETRIES+1),
	)

	localBucketing := &recordingConfigReceiver{}
	manager := NewEnvironmentConfigManager(test_environmentKey, localBucketing, test_options_sse, NewConfiguration(test_options_sse))

	err := manager.StartSSE()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func errorResponseChain(errorResponse httpmock.Responder, count int, configMock ...func(respcode int) httpmock.Responder) httpmock.Responder {

	var successResponse httpmock.Responder
	if configMock != nil {
		successResponse = configMock[0](200)
	} else {
		successResponse = httpConfigMock(200)
	}
	response := errorResponse
	for i := 1; i < count; i++ {
		response = response.Then(errorResponse)
	}
	response = response.Then(successResponse)
	return response
}
