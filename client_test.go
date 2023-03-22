package devcycle

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jarcoal/httpmock"
)

func TestDVCClient_AllFeatures_Local(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})
	fatalErr(t, err)

	features, err := c.AllFeatures(
		DVCUser{UserId: "j_test", DeviceModel: "testing"})
	fatalErr(t, err)

	fmt.Println(features)
}

func TestDVCClient_AllVariablesLocal(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})
	fatalErr(t, err)

	variables, err := c.AllVariables(
		DVCUser{UserId: "j_test", DeviceModel: "testing"})
	fatalErr(t, err)

	fmt.Println(variables)
}

func TestDVCClient_VariableCloud(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpBucketingAPIMock()
	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{EnableCloudBucketing: true, ConfigPollingIntervalMS: 10 * time.Second})

	variable, err := c.Variable(
		DVCUser{UserId: "j_test", DeviceModel: "testing"},
		"test", true)
	fatalErr(t, err)

	fmt.Println(variable)
}

func TestDVCClient_VariableLocalNumber(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)

	c, err := NewDVCClient(test_environmentKey, &DVCOptions{})

	variable, err := c.Variable(
		DVCUser{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}},
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

func TestDVCClient_VariableLocal(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})

	variable, err := c.Variable(
		DVCUser{UserId: "j_test", DeviceModel: "testing"},
		"test", true)
	fatalErr(t, err)

	fmt.Println(variable)
}

func TestDVCClient_VariableLocalProtobuf(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})

	variable, err := c.Variable(
		DVCUser{UserId: "j_test", DeviceModel: "testing"},
		"test", true)
	fatalErr(t, err)

	expected := Variable{
		baseVariable: baseVariable{
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

func TestDVCClient_VariableLocalProtobuf_UserWithCustomData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})

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
		DVCUser{
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
		baseVariable: baseVariable{
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

func TestDVCClient_VariableLocal_403(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(403)

	_, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})
	if err == nil {
		t.Fatal("Expected error from configmanager")
	}
}

func TestDVCClient_VariableLocalProtobuf_StringEncoding(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_config_special_characters_var)

	options := &DVCOptions{
		UseDebugWASM: true,
	}

	c, err := NewDVCClient("dvc_server_token_hash", options)

	variable, err := c.Variable(
		DVCUser{
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
		baseVariable: baseVariable{
			Key:   "test",
			Type_: "String",
			Value: "ö",
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

func TestDVCClient_TrackLocal_QueueEvent(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	dvcOptions := DVCOptions{ConfigPollingIntervalMS: 10 * time.Second}

	c, err := NewDVCClient(test_environmentKey, &dvcOptions)

	track, err := c.Track(DVCUser{UserId: "j_test", DeviceModel: "testing"}, DVCEvent{
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
	user := DVCUser{UserId: "test"}
	if environmentKey == "" {
		t.Skip("DVC_SERVER_KEY not set. Not using production tests.")
	}
	dvcOptions := DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}
	client, err := NewDVCClient(environmentKey, &dvcOptions)
	if err != nil {
		t.Fatal(err)
	}

	variables, err := client.AllVariables(user)
	fatalErr(t, err)

	if len(variables) == 0 {
		t.Fatal("No variables returned")
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

func BenchmarkDVCClient_VariableSerial(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	if benchmarkDisableLogs {
		log.SetOutput(io.Discard)
	}

	options := &DVCOptions{
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

	client, err := NewDVCClient(test_environmentKey, options)
	if err != nil {
		b.Errorf("Failed to initialize client: %v", err)
	}

	user := DVCUser{UserId: "dontcare"}

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

func BenchmarkDVCClient_VariableParallel(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	if benchmarkDisableLogs {
		log.SetOutput(io.Discard)
	}

	options := &DVCOptions{
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
		infof("Enabling event logging")
		options.DisableAutomaticEventLogging = false
		options.DisableCustomEventLogging = false
		options.EventFlushIntervalMS = time.Millisecond * 500
	}

	client, err := NewDVCClient(test_environmentKey, options)
	if err != nil {
		b.Errorf("Failed to initialize client: %v", err)
	}

	user := DVCUser{UserId: "dontcare"}

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
					err = client.configManager.setConfig([]byte(test_large_config))
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
