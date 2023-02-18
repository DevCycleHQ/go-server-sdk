package devcycle

import (
	"fmt"
	"os"
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
	if err != nil {
		t.Fatal(err)
	}
}
