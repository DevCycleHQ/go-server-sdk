package devcycle

import (
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_AllFeatures_Local(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	features, err := c.AllFeatures(
		User{UserId: "j_test", DeviceModel: "testing"})
	fatalErr(t, err)

	fmt.Println(features)
}

func TestClient_AllVariablesLocal(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	c, err := NewClient(sdkKey, &Options{})
	require.NoError(t, err)

	variables, err := c.AllVariables(
		User{UserId: "j_test", DeviceModel: "testing"})
	require.NoError(t, err)

	require.Len(t, variables, 5)
}

func TestClient_AllVariablesLocal_WithSpecialCharacters(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_config_special_characters_var, false)
	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	variables, err := c.AllVariables(
		User{UserId: "j_test", DeviceModel: "testing"})
	fatalErr(t, err)

	fmt.Println(variables)
	if len(variables) != 1 {
		t.Error("Expected 1 variable, got", len(variables))
	}

	expected := Variable{
		BaseVariable: BaseVariable{
			Key:   "test",
			Type_: "String",
			Value: "öé 🐍 ¥",
		},
	}
	if variables["test"].Key != expected.Key {
		t.Fatal("Variable key to be equal to expected variable")
	}
	if variables["test"].Type_ != expected.Type_ {
		t.Fatal("Variable type to be equal to expected variable")
	}
	if variables["test"].Value != expected.Value {
		t.Fatal("Variable value to be equal to expected variable")
	}
}

func TestClient_VariableCloud(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpBucketingAPIMock()
	c, err := NewClient(sdkKey, &Options{EnableCloudBucketing: true, ConfigPollingIntervalMS: 10 * time.Second})
	fatalErr(t, err)

	user := User{UserId: "j_test", DeviceModel: "testing"}
	variable, err := c.Variable(user, "test", true)
	fatalErr(t, err)
	fmt.Println(variable)

	variableValue, err := c.VariableValue(user, "test", true)
	fatalErr(t, err)
	fmt.Println(variableValue)
}

func TestClient_VariableLocalNumber(t *testing.T) {

	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)

	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	user := User{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}
	fmt.Println(c.AllVariables(user))
	variable, err := c.Variable(user, "v-key-76", 69)
	fatalErr(t, err)

	if variable.IsDefaulted || variable.Value == 69 {
		t.Fatal("variable should not be defaulted")
	}
	fmt.Println(variable.Value)
	if variable.Value.(float64) != 60.0 {
		t.Fatal("variable should be 60")
	}
	fmt.Println(variable.IsDefaulted)
	fmt.Println(variable)

	variableValue, err := c.VariableValue(user, "v-key-76", 69)
	fatalErr(t, err)
	if variableValue.(float64) != 60.0 {
		t.Fatal("variableValue should be 60")
	}
	fmt.Println(variableValue)
}

func TestClient_VariableLocalNumberWithNilDefault(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)

	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	user := User{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}
	fmt.Println(c.AllVariables(user))
	variable, err := c.Variable(
		user,
		"v-key-76", nil)
	fatalErr(t, err)

	if variable.IsDefaulted || variable.Value == nil {
		t.Fatal("variable should not be defaulted")
	}
	fmt.Println(variable.Value)
	if variable.Value.(float64) != 60.0 {
		t.Fatal("variable should be 60")
	}
	fmt.Println(variable.IsDefaulted)
	fmt.Println(variable)

	variable, err = c.Variable(user, "nonsense-key", nil)
	fatalErr(t, err)
	fmt.Println(variable)
}

func TestClient_VariableEventIsQueued(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)
	httpEventsApiMock()

	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	user := User{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}
	fmt.Println(c.AllVariables(user))
	variable, err := c.Variable(user, "v-key-76", 69)
	fatalErr(t, err)

	if variable.IsDefaulted || variable.Value == 69 {
		t.Fatal("variable should not be defaulted")
	}
	fmt.Println(variable.Value)
	if variable.Value.(float64) != 60.0 {
		t.Fatal("variable should be 60")
	}
	fmt.Println(variable.IsDefaulted)
	fmt.Println(variable)
	err = c.eventQueue.FlushEvents()
	require.NoError(t, err)
}

func TestClient_VariableLocalFlush(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)

	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	user := User{UserId: "j_test", DeviceModel: "testing"}
	variable, err := c.Variable(user, "variableThatShouldBeDefaulted", true)
	fatalErr(t, err)
	err = c.FlushEvents()
	fatalErr(t, err)
	fmt.Println(variable)
}

func TestClient_VariableLocal(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)

	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	user := User{UserId: "j_test", DeviceModel: "testing"}
	variable, err := c.Variable(user, "test", true)
	fatalErr(t, err)

	expected := Variable{
		BaseVariable: BaseVariable{
			Key:   "test",
			Type_: "Boolean",
			Value: true,
		},
		DefaultValue: true,
		IsDefaulted:  false,
	}
	if !reflect.DeepEqual(expected, variable) {
		fmt.Println("got", variable)
		fmt.Println("expected", expected)
		t.Fatal("Expected variable to be equal to expected variable")
	}
	fmt.Println(variable)

	variableValue, err := c.VariableValue(user, "test", true)
	fatalErr(t, err)
	if variableValue != true {
		t.Fatal("Expected variableValue to be true")
	}
}

func TestClient_VariableLocal_UserWithCustomData(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)

	c, err := NewClient(sdkKey, &Options{})
	fatalErr(t, err)

	customData := map[string]interface{}{
		"propStr":  "hello",
		"propInt":  1,
		"propBool": true,
		"propNull": nil,
	}
	customPrivateData := map[string]interface{}{
		"aPrivateValue": "asuh",
	}

	user := User{
		UserId:            "j_test",
		DeviceModel:       "testing",
		Name:              "Pedro Pascal",
		Email:             "pedro@pascal.com",
		AppBuild:          "1.0.0",
		CustomData:        customData,
		PrivateCustomData: customPrivateData,
	}
	variable, err := c.Variable(user, "test", true)
	fatalErr(t, err)

	expected := Variable{
		BaseVariable: BaseVariable{
			Key:   "test",
			Type_: "Boolean",
			Value: true,
		},
		DefaultValue: true,
		IsDefaulted:  false,
	}
	if !reflect.DeepEqual(expected, variable) {
		fmt.Println("got", variable)
		fmt.Println("expected", expected)
		t.Fatal("Expected variable to be equal to expected variable")
	}
	fmt.Println(variable)

	variableValue, err := c.VariableValue(user, "test", true)
	fatalErr(t, err)
	if variableValue != true {
		t.Fatal("Expected variableValue to be true")
	}
}

func TestClient_VariableLocal_403(t *testing.T) {
	sdkKey, _ := httpConfigMock(403)
	_, err := NewClient(sdkKey, &Options{})
	if err == nil {
		t.Fatal("Expected error from configmanager")
	}
}

func TestClient_TrackLocal_QueueEvent(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	dvcOptions := Options{ConfigPollingIntervalMS: 10 * time.Second}

	c, err := NewClient(sdkKey, &dvcOptions)
	fatalErr(t, err)

	track, err := c.Track(User{UserId: "j_test", DeviceModel: "testing"}, Event{
		Target:      "customEvent",
		Value:       0,
		Type_:       "someType",
		FeatureVars: nil,
		MetaData:    nil,
	})
	fatalErr(t, err)

	fmt.Println(track)
}

func TestClient_TrackLocal_QueueEventBeforeConfig(t *testing.T) {

	// Config will fail to load on HTTP 500 after several retries without an error
	sdkKey, _ := httpConfigMock(http.StatusInternalServerError)
	dvcOptions := Options{ConfigPollingIntervalMS: 10 * time.Second}

	// Expect initial retry to fail and return an error
	c, err := NewClient(sdkKey, &dvcOptions)

	track, err := c.Track(User{UserId: "j_test", DeviceModel: "testing"}, Event{
		Target:      "customEvent",
		Value:       0,
		Type_:       "someType",
		FeatureVars: nil,
		MetaData:    nil,
	})
	fatalErr(t, err)

	fmt.Println(track)
}

func TestProduction_Local(t *testing.T) {
	environmentKey := os.Getenv("DEVCYCLE_SERVER_SDK_KEY")
	if environmentKey == "" {
		t.Skip("DEVCYCLE_SERVER_SDK_KEY not set. Not using production tests.")
	}
	user := User{UserId: "test"}

	dvcOptions := Options{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}
	client, err := NewClient(environmentKey, &dvcOptions)
	if err != nil {
		t.Fatal(err)
	}

	variables, err := client.AllVariables(user)
	fatalErr(t, err)

	if len(variables) == 0 {
		t.Fatal("No variables returned")
	}
}

func TestClient_CloudBucketingHandler(t *testing.T) {

	sdkKey := generateTestSDKKey()
	httpBucketingAPIMock()
	clientEventHandler := make(chan api.ClientEvent, 10)
	c, err := NewClient(sdkKey, &Options{EnableCloudBucketing: true, ClientEventHandler: clientEventHandler})
	fatalErr(t, err)
	init := <-clientEventHandler

	if init.EventType != api.ClientEventType_Initialized {
		t.Fatal("Expected initialized event")
	}
	if !c.isInitialized {
		t.Fatal("Expected client to be initialized")
	}
}

func TestClient_LocalBucketingHandler(t *testing.T) {

	sdkKey, _ := httpConfigMock(200)
	clientEventHandler := make(chan api.ClientEvent, 10)
	c, err := NewClient(sdkKey, &Options{ClientEventHandler: clientEventHandler})
	fatalErr(t, err)
	event1 := <-clientEventHandler
	event2 := <-clientEventHandler
	switch event1.EventType {
	case api.ClientEventType_Initialized:
		if event2.EventType != api.ClientEventType_ConfigUpdated {
			t.Fatal("Expected config updated event and initialized events")
		}
	case api.ClientEventType_ConfigUpdated:
		if event2.EventType != api.ClientEventType_Initialized {
			t.Fatal("Expected initialized and config updated events")
		}
	}
	if !c.isInitialized {
		t.Fatal("Expected client to be initialized")
	}
	if !c.hasConfig() {
		t.Fatal("Expected client to have config")
	}
}

func TestClient_ConfigUpdatedEvent(t *testing.T) {
	responder := func(req *http.Request) (*http.Response, error) {
		reqBody, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(string(reqBody), api.EventType_SDKConfig) {
			return nil, fmt.Errorf("Expected config updated event")
		}

		return httpmock.NewStringResponse(201, `{}`), nil
	}
	httpmock.RegisterResponder("POST", "https://config-updated.devcycle.com/v1/events/batch", responder)
	sdkKey, _ := httpConfigMock(200)
	c, err := NewClient(sdkKey, &Options{EventsAPIURI: "https://config-updated.devcycle.com", EventFlushIntervalMS: 500 * time.Millisecond})
	fatalErr(t, err)
	if !c.isInitialized {
		t.Fatal("Expected client to be initialized")
	}
	if !c.hasConfig() {
		t.Fatal("Expected client to have config")
	}
	require.Eventually(t, func() bool {
		return httpmock.GetCallCountInfo()["POST https://config-updated.devcycle.com/v1/events/batch"] >= 1
	}, 1*time.Second, 100*time.Millisecond)
}
func TestClient_ConfigUpdatedEvent_Detail(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	responder := func(req *http.Request) (*http.Response, error) {
		reqBody, err := io.ReadAll(req.Body)
		fmt.Println(string(reqBody))
		if err != nil {
			return httpmock.NewStringResponse(500, `{}`), err
		}
		if !strings.Contains(string(reqBody), api.EventType_SDKConfig) {
			t.Fatal("Expected config updated event in request body")
		}
		return httpmock.NewStringResponse(201, `{}`), nil
	}
	httpmock.RegisterResponder("POST", "https://config-updated.devcycle.com/v1/events/batch", responder)
	c, err := NewClient(sdkKey, &Options{EventsAPIURI: "https://config-updated.devcycle.com", EventFlushIntervalMS: 500 * time.Millisecond})
	fatalErr(t, err)
	if !c.isInitialized {
		t.Fatal("Expected client to be initialized")
	}
	if !c.hasConfig() {
		t.Fatal("Expected client to have config")
	}

	require.Eventually(t, func() bool {
		return httpmock.GetCallCountInfo()["POST https://config-updated.devcycle.com/v1/events/batch"] >= 1
	}, 1*time.Second, 100*time.Millisecond)
}

func TestClient_ConfigUpdatedEvent_VariableEval(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	responder := func(req *http.Request) (*http.Response, error) {
		reqBody, err := io.ReadAll(req.Body)
		fmt.Println(string(reqBody))
		if err != nil {
			return httpmock.NewStringResponse(500, `{}`), err
		}
		if !strings.Contains(string(reqBody), api.EventType_SDKConfig) || !strings.Contains(string(reqBody), api.EventType_AggVariableDefaulted) {
			fmt.Println("Expected config updated event and defaulted event in request body")
		}
		return httpmock.NewStringResponse(201, `{}`), nil
	}
	httpmock.RegisterResponder("POST", "https://config-updated.devcycle.com/v1/events/batch", responder)
	c, err := NewClient(sdkKey, &Options{EventsAPIURI: "https://config-updated.devcycle.com", EventFlushIntervalMS: time.Millisecond * 500})
	fatalErr(t, err)
	if !c.isInitialized {
		t.Fatal("Expected client to be initialized")
	}
	if !c.hasConfig() {
		t.Fatal("Expected client to have config")
	}

	user := User{UserId: "j_test", DeviceModel: "testing"}
	variable, _ := c.Variable(user, "variableThatShouldBeDefaulted", true)

	if !variable.IsDefaulted {
		t.Fatal("Expected variable to be defaulted")
	}

	require.Eventually(t, func() bool {
		return httpmock.GetCallCountInfo()["POST https://config-updated.devcycle.com/v1/events/batch"] >= 1
	}, 1*time.Second, 100*time.Millisecond)
}

func BenchmarkClient_VariableSerial(b *testing.B) {
	util.SetLogger(util.DiscardLogger{})

	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)

	if benchmarkDisableLogs {
		log.SetOutput(io.Discard)
	}

	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		ConfigPollingIntervalMS:      time.Minute,
		EventFlushIntervalMS:         time.Minute,
	}

	if benchmarkEnableEvents {
		options.DisableAutomaticEventLogging = false
		options.DisableCustomEventLogging = false
		options.EventFlushIntervalMS = 0
	}

	client, err := NewClient(sdkKey, options)
	if err != nil {
		b.Errorf("Failed to initialize client: %v", err)
	}

	user := User{UserId: "dontcare"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		variable, err := client.Variable(user, test_large_config_variable, false)
		if err != nil {
			b.Errorf("Failed to retrieve variable: %v", err)
		}
		if variable.IsDefaulted {
			b.Fatal("Expected variable to return a value")
		}
	}
}

func BenchmarkClient_VariableParallel(b *testing.B) {
	util.SetLogger(util.DiscardLogger{})

	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)

	if benchmarkDisableLogs {
		log.SetOutput(io.Discard)
	}

	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		ConfigPollingIntervalMS:      time.Minute,
		EventFlushIntervalMS:         time.Minute,
	}
	if benchmarkEnableEvents {
		util.Infof("Enabling event logging")
		options.DisableAutomaticEventLogging = false
		options.DisableCustomEventLogging = false
		options.EventFlushIntervalMS = time.Millisecond * 500
	}

	client, err := NewClient(sdkKey, options)
	if err != nil {
		b.Errorf("Failed to initialize client: %v", err)
	}

	user := User{UserId: "dontcare"}

	b.ResetTimer()
	b.ReportAllocs()

	setConfigCount := atomic.Uint64{}
	configCounter := atomic.Uint64{}

	errors := make(chan error, b.N)

	var opNanos atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()
			variable, err := client.Variable(user, test_large_config_variable, false)
			duration := time.Since(start)
			opNanos.Add(duration.Nanoseconds())

			if err != nil {
				errors <- fmt.Errorf("Failed to retrieve variable: %v", err)
			}
			if benchmarkEnableConfigUpdates && configCounter.Add(1)%10000 == 0 {
				go func() {
					err = client.configManager.setConfig([]byte(test_large_config), "", "", "")
					setConfigCount.Add(1)
				}()
			}
			if variable.IsDefaulted {
				errors <- fmt.Errorf("Expected variable to return a value")
			}
		}
	})

	select {
	case err := <-errors:
		b.Error(err)
	default:
	}
	b.ReportMetric(float64(setConfigCount.Load()), "reconfigs")
	b.ReportMetric(float64(opNanos.Load())/float64(b.N), "ns")
	eventsFlushed, eventsReported, eventsDropped := client.eventQueue.Metrics()
	b.ReportMetric(float64(eventsFlushed), "eventsFlushed")
	b.ReportMetric(float64(eventsReported), "eventsReported")
	b.ReportMetric(float64(eventsDropped), "eventsDropped")
}
