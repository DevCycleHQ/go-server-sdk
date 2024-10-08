package devcycle

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

type recordingConfigReceiver struct {
	configureCount int
	etag           string
	rayId          string
	lastModified   string
	config         []byte
	mutex          sync.Mutex
}

func (r *recordingConfigReceiver) StoreConfig(c []byte, etag, rayId, lastModified string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.configureCount++
	r.etag = etag
	r.rayId = rayId
	r.lastModified = lastModified
	r.config = c

	return nil
}

func (r *recordingConfigReceiver) HasConfig() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.configureCount > 0
}

func (r *recordingConfigReceiver) GetETag() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.etag
}

func (r *recordingConfigReceiver) GetRayId() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.rayId
}

func (r *recordingConfigReceiver) GetRawConfig() []byte {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.config
}

func (r *recordingConfigReceiver) GetLastModified() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.lastModified
}

const CONFIG_SDK_URL = "https://config-cdn.devcycle.com/config/v2/server/%s.json"

func TestEnvironmentConfigManager_fetchConfig_success(t *testing.T) {

	sdkKey, _ := httpConfigMock(200)
	localBucketing := &recordingConfigReceiver{}
	testOptionsWithHandler := *test_options

	testOptionsWithHandler.ClientEventHandler = make(chan api.ClientEvent, 10)
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, &testOptionsWithHandler, NewConfiguration(&testOptionsWithHandler))
	defer manager.Close()
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
	event1 := <-testOptionsWithHandler.ClientEventHandler
	if event1.Status != "success" {
		fmt.Println(event1)
		t.Fatal("event1.Status != success")
	}
}

func TestEnvironmentConfigManager_fetchConfig_refuseOld(t *testing.T) {
	sdkKey := generateTestSDKKey()
	initialHeaders := map[string]string{
		"Etag":          "INITIAL-ETAG",
		"Last-Modified": time.Now().Add(-time.Hour).Format(time.RFC1123),
		"Cf-Ray":        "INITIAL-CF-RAY",
	}
	olderHeaders := map[string]string{
		"Etag":          "OLDER-ETAG",
		"Last-Modified": time.Now().Add(-time.Hour * 2).Format(time.RFC1123),
		"Cf-Ray":        "OLDER-CF-RAY",
	}
	newestHeaders := map[string]string{
		"Etag":          "NEWEST-ETAG",
		"Last-Modified": time.Now().Add(time.Hour * 3).Format(time.RFC1123),
		"Cf-Ray":        "NEWEST-CF-RAY",
	}
	firstResponse := httpCustomConfigMock(sdkKey, 200, test_config, true, initialHeaders)
	secondResponse := httpCustomConfigMock(sdkKey, 200, test_config, true, olderHeaders)
	thirdResponse := httpCustomConfigMock(sdkKey, 200, test_config, true, newestHeaders)

	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		firstResponse.Then(secondResponse).Then(thirdResponse),
	)
	localBucketing := &recordingConfigReceiver{}
	testOptionsWithHandler := *test_options
	testOptionsWithHandler.ConfigPollingIntervalMS = time.Second * 1
	testOptionsWithHandler.ClientEventHandler = make(chan api.ClientEvent, 10)
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, &testOptionsWithHandler, NewConfiguration(&testOptionsWithHandler))
	defer manager.Close()
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
	if manager.GetETag() != "INITIAL-ETAG" {
		t.Fatal("cm.configEtag != INITIAL-ETAG")
	}
	if manager.GetLastModified() != initialHeaders["Last-Modified"] {
		t.Fatal("cm.lastModified != " + initialHeaders["Last-Modified"])
	}
	event1 := <-testOptionsWithHandler.ClientEventHandler
	if event1.Status != "success" {
		fmt.Println(event1)
		t.Fatal("event1.Status != success")
	}

	require.Never(t, func() bool {
		if manager.GetETag() == "OLDER-ETAG" || manager.GetLastModified() == olderHeaders["Last-Modified"] {
			return true
		}
		return false
	}, 2*time.Second, 1*time.Second)

	require.Eventually(t, func() bool {
		return manager.GetLastModified() == newestHeaders["Last-Modified"]
	}, 3*time.Second, 500*time.Millisecond)
}

func TestEnvironmentConfigManager_fetchConfig_success_sse(t *testing.T) {

	sdkKey, _ := httpSSEConfigMock(200)
	httpSSEConnectionMock()

	localBucketing := &recordingConfigReceiver{}

	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options_sse, NewConfiguration(test_options_sse))
	defer manager.Close()
	err := manager.initialFetch()
	require.NoError(t, err)
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
	require.Eventually(t, func() bool {
		return manager.sseManager.Connected.Load()
	}, 3*time.Second, 10*time.Millisecond)

}

func TestEnvironmentConfigManager_fetchConfig_retries500(t *testing.T) {
	sdkKey := generateTestSDKKey()

	error500Response := httpCustomConfigMock(sdkKey, 500, "Connection error", true)

	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		errorResponseChain(sdkKey, error500Response, CONFIG_RETRIES),
	)

	localBucketing := &recordingConfigReceiver{}
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))
	defer manager.Close()
	err := manager.initialFetch()
	require.NoError(t, err)
	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.GetETag() != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
	if _, err = time.Parse(time.RFC1123, manager.GetLastModified()); err != nil {
		t.Fatal("cm.lastModified was not validly parsed")
	}
}

func TestEnvironmentConfigManager_fetchConfig_retries_until_abort(t *testing.T) {
	sdkKey := generateTestSDKKey()
	initialHeaders := map[string]string{
		"Etag":          "INITIAL-ETAG",
		"Last-Modified": time.Now().Add(-time.Hour).Format(time.RFC1123),
		"Cf-Ray":        "INITIAL-CF-RAY",
	}
	olderHeaders := map[string]string{
		"Etag":          "OLDER-ETAG",
		"Last-Modified": time.Now().Add(-time.Hour * 2).Format(time.RFC1123),
		"Cf-Ray":        "OLDER-CF-RAY",
	}
	firstResponse := httpCustomConfigMock(sdkKey, 200, test_config, true, initialHeaders)
	secondResponse := httpCustomConfigMock(sdkKey, 200, test_config, true, olderHeaders)

	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		firstResponse.Then(secondResponse).Then(secondResponse).Then(secondResponse),
	)
	localBucketing := &recordingConfigReceiver{}
	testOptionsWithHandler := *test_options
	testOptionsWithHandler.ConfigPollingIntervalMS = time.Second * 1
	testOptionsWithHandler.ClientEventHandler = make(chan api.ClientEvent, 10)
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, &testOptionsWithHandler, NewConfiguration(&testOptionsWithHandler))
	defer manager.Close()
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
	if manager.GetETag() != "INITIAL-ETAG" {
		t.Fatal("cm.configEtag != INITIAL-ETAG")
	}
	if manager.GetLastModified() != initialHeaders["Last-Modified"] {
		t.Fatal("cm.lastModified != " + initialHeaders["Last-Modified"])
	}
	event1 := <-testOptionsWithHandler.ClientEventHandler
	if event1.Status != "success" {
		fmt.Println(event1)
		t.Fatal("event1.Status != success")
	}

	// Config fetch will abort and return an error before the third retry.
	require.Never(t, func() bool {
		if manager.GetETag() == "OLDER-ETAG" || manager.GetLastModified() == olderHeaders["Last-Modified"] {
			return true
		}
		return false
	}, 2*time.Second, 1*time.Second)
	require.Eventually(t, func() bool {
		event := <-manager.InternalClientEvents
		fmt.Println(event)
		return event.EventType == api.ClientEventType_Error
	}, 3*time.Second, 500*time.Millisecond)
}

func TestEnvironmentConfigManager_fetchConfig_retries_errors(t *testing.T) {
	sdkKey := generateTestSDKKey()
	connectionErrorResponse := httpCustomConfigMock(sdkKey, 500, "Connection error", true)
	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		errorResponseChain(sdkKey, connectionErrorResponse, CONFIG_RETRIES),
	)

	localBucketing := &recordingConfigReceiver{}
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))
	defer manager.Close()

	err := manager.initialFetch()
	require.NoError(t, err)
	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
	if manager.GetETag() != "TESTING" {
		t.Fatal("cm.configEtag != TESTING")
	}
	if _, err = time.Parse(time.RFC1123, manager.GetLastModified()); err != nil {
		t.Fatal("cm.lastModified was not validly parsed")
	}
}

func TestEnvironmentConfigManager_fetchConfig_retries_errors_sse(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpSSEConnectionMock()

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))
	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		errorResponseChain(sdkKey, connectionErrorResponse, CONFIG_RETRIES, httpSSEConfigMock),
	)

	localBucketing := &recordingConfigReceiver{}
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options_sse, NewConfiguration(test_options_sse))
	defer manager.Close()
	err := manager.initialFetch()
	require.NoError(t, err)

	if !manager.HasConfig() {
		t.Fatal("cm.hasConfig != true")
	}
}

func TestEnvironmentConfigManager_fetchConfig_returns_errors(t *testing.T) {

	sdkKey := generateTestSDKKey()
	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))

	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		errorResponseChain(sdkKey, connectionErrorResponse, CONFIG_RETRIES+1),
	)

	localBucketing := &recordingConfigReceiver{}
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options, NewConfiguration(test_options))
	defer manager.Close()
	err := manager.initialFetch()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestEnvironmentConfigManager_fetchConfig_returns_errors_sse(t *testing.T) {

	connectionErrorResponse := httpmock.NewErrorResponder(fmt.Errorf("connection error"))
	sdkKey := generateTestSDKKey()
	httpmock.RegisterResponder("GET", fmt.Sprintf(CONFIG_SDK_URL, sdkKey),
		errorResponseChain(sdkKey, connectionErrorResponse, CONFIG_RETRIES+1),
	)

	localBucketing := &recordingConfigReceiver{}
	manager, _ := NewEnvironmentConfigManager(sdkKey, localBucketing, nil, test_options_sse, NewConfiguration(test_options_sse))
	defer manager.Close()

	err := manager.initialFetch()
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if manager.HasConfig() {
		t.Fatal("manager.hasConfig == true")
	}
	if manager.sseManager.Started {
		t.Fatal("manager.sseManager.Started == true")
	}

}

func errorResponseChain(sdkKey string, errorResponse httpmock.Responder, count int, configMock ...func(respcode int, sdkKeys ...string) (string, httpmock.Responder)) httpmock.Responder {

	var successResponse httpmock.Responder
	if configMock != nil {
		_, successResponse = configMock[0](200, sdkKey)
	} else {
		successResponse = httpCustomConfigMock(sdkKey, 200, test_config, true)
	}
	response := errorResponse
	for i := 1; i < count; i++ {
		response = response.Then(errorResponse)
	}
	response = response.Then(successResponse)
	return response
}
