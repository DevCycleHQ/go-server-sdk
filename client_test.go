package devcycle

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

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

func BenchmarkDVCClient_Variable(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	options := &DVCOptions{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		ConfigPollingIntervalMS:      time.Minute,
		EventFlushIntervalMS:         time.Minute,
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

func BenchmarkDVCClient_Variable_Protobuf(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_large_config)
	httpEventsApiMock()

	options := &DVCOptions{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		ConfigPollingIntervalMS:      time.Minute,
		EventFlushIntervalMS:         time.Minute,
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
