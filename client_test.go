package devcycle

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/devcyclehq/go-server-sdk/v2/util"
	"github.com/stretchr/testify/require"

	"github.com/jarcoal/httpmock"
)

func TestClient_AllFeatures_Local(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	features, err := c.AllFeatures(
		User{UserId: "j_test", DeviceModel: "testing"})
	fatalErr(t, err)

	fmt.Println(features)
}

func TestClient_AllVariablesLocal(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	variables, err := c.AllVariables(
		User{UserId: "j_test", DeviceModel: "testing"})
	fatalErr(t, err)

	fmt.Println(variables)
	if len(variables) != 1 {
		t.Error("Expected 1 variable, got", len(variables))
	}
}

func TestClient_AllVariablesLocal_WithSpecialCharacters(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_config_special_characters_var)
	c, err := NewClient(test_environmentKey, &Options{})
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
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpBucketingAPIMock()
	c, err := NewClient(test_environmentKey, &Options{EnableCloudBucketing: true, ConfigPollingIntervalMS: 10 * time.Second})
	fatalErr(t, err)

	variable, err := c.Variable(
		User{UserId: "j_test", DeviceModel: "testing"},
		"test", true)
	fatalErr(t, err)

	fmt.Println(variable)
}

func TestClient_VariableLocalNumber(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)

	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	user := User{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}
	fmt.Println(c.AllVariables(user))
	variable, err := c.Variable(
		user,
		"v-key-76", 69)
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
}

func TestClient_VariableEventIsQueued(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	user := DVCUser{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}
	fmt.Println(c.AllVariables(user))
	variable, err := c.Variable(
		user,
		"v-key-76", 69)
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

func TestClient_VariableLocal(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	variable, err := c.Variable(
		User{UserId: "j_test", DeviceModel: "testing"},
		"test", true)
	fatalErr(t, err)

	fmt.Println(variable)
}

func TestClient_VariableLocalFlush(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	variable, err := c.Variable(
		DVCUser{UserId: "j_test", DeviceModel: "testing"},
		"variableThatShouldBeDefaulted", true)
	fatalErr(t, err)
	err = c.FlushEvents()
	fatalErr(t, err)
	fmt.Println(variable)
}

func TestClient_VariableLocalProtobuf(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	variable, err := c.Variable(
		User{UserId: "j_test", DeviceModel: "testing"},
		"test", true)
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
}

func TestClient_VariableLocalProtobuf_UserWithCustomData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewClient(test_environmentKey, &Options{})
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

	variable, err := c.Variable(
		User{
			UserId:            "j_test",
			DeviceModel:       "testing",
			Name:              "Pedro Pascal",
			Email:             "pedro@pascal.com",
			AppBuild:          "1.0.0",
			CustomData:        customData,
			PrivateCustomData: customPrivateData,
		},
		"test", true)
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
}

func TestClient_VariableLocal_403(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(403)

	_, err := NewClient(test_environmentKey, &Options{})
	if err == nil {
		t.Fatal("Expected error from configmanager")
	}
}

func TestClient_VariableLocalProtobuf_StringEncoding(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_config_special_characters_var)

	options := &Options{
		UseDebugWASM: true,
	}

	c, err := NewClient(test_environmentKey, options)
	fatalErr(t, err)

	variable, err := c.Variable(
		User{
			UserId: "someuser",
		},
		"test", "default_value")
	fatalErr(t, err)

	fmt.Printf("Value: %v | bytes %v\n", variable.Value, []byte(variable.Value.(string)))
	fmt.Printf("Is Valid UTF-8 String: %v\n", utf8.ValidString(variable.Value.(string)))

	fmt.Println(variable)
	if variable.IsDefaulted {
		t.Fatal("Expected variable to return a value")
	}

	expected := Variable{
		BaseVariable: BaseVariable{
			Key:   "test",
			Type_: "String",
			Value: "öé 🐍 ¥",
		},
		DefaultValue: "default_value",
		IsDefaulted:  false,
	}
	if !reflect.DeepEqual(expected, variable) {
		fmt.Println("got", variable)
		fmt.Println("expected", expected)
		t.Fatal("Expected variable to be equal to expected variable")
	}
}

func TestClient_TrackLocal_QueueEvent(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	dvcOptions := Options{ConfigPollingIntervalMS: 10 * time.Second}

	c, err := NewClient(test_environmentKey, &dvcOptions)
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
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Config will fail to load on HTTP 500 after several retries without an error
	httpConfigMock(http.StatusInternalServerError)
	dvcOptions := Options{ConfigPollingIntervalMS: 10 * time.Second}

	c, err := NewClient(test_environmentKey, &dvcOptions)
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

func TestProduction_Local(t *testing.T) {
	environmentKey := os.Getenv("DVC_SERVER_KEY")
	user := User{UserId: "test"}
	if environmentKey == "" {
		t.Skip("DVC_SERVER_KEY not set. Not using production tests.")
	}
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

func TestClient_Validate_OnInitializedChannel_EnableCloudBucketing_Options(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	onInitialized := make(chan bool)

	// Try each of the combos to make sure they all act as expected and don't hang
	dvcOptions := Options{OnInitializedChannel: onInitialized, EnableCloudBucketing: true}
	c, err := NewClient(test_environmentKey, &dvcOptions)
	fatalErr(t, err)
	val := <-onInitialized
	if !val {
		t.Fatal("Expected true from onInitialized channel")
	}

	if c.isInitialized {
		// isInitialized is only relevant when using Local Bucketing
		t.Fatal("Expected isInitialized to be false")
	}

	dvcOptions = Options{OnInitializedChannel: onInitialized, EnableCloudBucketing: false}
	c, err = NewClient(test_environmentKey, &dvcOptions)
	fatalErr(t, err)
	val = <-onInitialized
	if !val {
		t.Fatal("Expected true from onInitialized channel")
	}

	if !c.isInitialized {
		t.Fatal("Expected isInitialized to be true")
	}

	if !c.hasConfig() {
		t.Fatal("Expected config to be loaded")
	}

	dvcOptions = Options{OnInitializedChannel: nil, EnableCloudBucketing: true}
	c, err = NewClient(test_environmentKey, &dvcOptions)
	fatalErr(t, err)

	if c.isInitialized {
		// isInitialized is only relevant when using Local Bucketing
		t.Fatal("Expected isInitialized to be false")
	}

	dvcOptions = Options{OnInitializedChannel: nil, EnableCloudBucketing: false}
	c, err = NewClient(test_environmentKey, &dvcOptions)
	fatalErr(t, err)

	if !c.isInitialized {
		t.Fatal("Expected isInitialized to be true")
	}

	if !c.hasConfig() {
		t.Fatal("Expected config to be loaded")
	}
}

func fatalErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var (
	benchmarkEnableEvents        bool
	benchmarkEnableConfigUpdates bool
	benchmarkNumWorkers          int
	benchmarkDisableLogs         bool
)

func init() {
	flag.BoolVar(&benchmarkEnableEvents, "benchEnableEvents", false, "Custom test flag that enables event logging in benchmarks")
	flag.BoolVar(&benchmarkEnableConfigUpdates, "benchEnableConfigUpdates", false, "Custom test flag that enables config updates in benchmarks")
	flag.IntVar(&benchmarkNumWorkers, "benchNumWorkers", runtime.NumCPU(), "Custom test flag that sets the number of WASM workers in benchmarks")
	flag.BoolVar(&benchmarkDisableLogs, "benchDisableLogs", false, "Custom test flag that disables logging in benchmarks")
}

func BenchmarkClient_VariableSerial(b *testing.B) {
	util.SetLogger(util.DiscardLogger{})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	if benchmarkDisableLogs {
		log.SetOutput(io.Discard)
	}

	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		ConfigPollingIntervalMS:      time.Minute,
		EventFlushIntervalMS:         time.Minute,
		AdvancedOptions: AdvancedOptions{
			MaxWasmWorkers: benchmarkNumWorkers,
		},
	}

	if benchmarkEnableEvents {
		options.DisableAutomaticEventLogging = false
		options.DisableCustomEventLogging = false
		options.EventFlushIntervalMS = 0
	}

	client, err := NewClient(test_environmentKey, options)
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
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	if benchmarkDisableLogs {
		log.SetOutput(io.Discard)
	}

	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		ConfigPollingIntervalMS:      time.Minute,
		EventFlushIntervalMS:         time.Minute,
		AdvancedOptions: AdvancedOptions{
			MaxWasmWorkers: benchmarkNumWorkers,
		},
	}
	if benchmarkEnableEvents {
		util.Infof("Enabling event logging")
		options.DisableAutomaticEventLogging = false
		options.DisableCustomEventLogging = false
		options.EventFlushIntervalMS = time.Millisecond * 500
	}

	client, err := NewClient(test_environmentKey, options)
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
					err = client.configManager.setConfig([]byte(test_large_config), "")
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

	b.ReportMetric(float64(benchmarkNumWorkers), "workers")
	b.ReportMetric(float64(setConfigCount.Load()), "reconfigs")
	b.ReportMetric(float64(opNanos.Load())/float64(b.N), "ns")
	eventsFlushed, eventsReported := client.eventQueue.Metrics()
	b.ReportMetric(float64(eventsFlushed), "eventsFlushed")
	b.ReportMetric(float64(eventsReported), "eventsReported")
}
