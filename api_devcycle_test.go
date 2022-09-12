package devcycle

import (
	"context"
	"fmt"
	"github.com/jarcoal/httpmock"
	"net/http"
	"testing"
	"time"
)

func TestDVCClientService_AllFeatures_Local(t *testing.T) {
	auth := context.WithValue(context.Background(), ContextAPIKey, APIKey{
		Key: "dvc_server_token_hash",
	})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	c := NewDVCClient("dvc_server_token_hash", &DVCOptions{PollingInterval: 10 * time.Second})

	c.configManager.SDKEvents = make(chan SDKEvent, 100)
	go func() {
		for {
			v := <-c.configManager.SDKEvents
			fmt.Println(v.Message, v.Error, v.FirstInitialization, v.Success)
		}
	}()
	features, err := c.DevCycleApi.AllFeatures(auth,
		UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"})
	if err != nil {
		t.Fatal(err)
		return
	}

	fmt.Println(features)

}
func TestDVCClientService_AllVariablesLocal(t *testing.T) {
	auth := context.WithValue(context.Background(), ContextAPIKey, APIKey{
		Key: "dvc_server_token_hash",
	})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	c := NewDVCClient("dvc_server_token_hash", &DVCOptions{PollingInterval: 10 * time.Second})

	c.configManager.SDKEvents = make(chan SDKEvent, 100)
	go func() {
		for {
			v := <-c.configManager.SDKEvents
			fmt.Println(v.Message, v.Error, v.FirstInitialization, v.Success)
		}
	}()
	variables, err := c.DevCycleApi.AllVariables(auth,
		UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"})
	if err != nil {
		t.Fatal(err)
		return
	}

	fmt.Println(variables)
}

func TestDVCClientService_VariableCloud(t *testing.T) {
	auth := context.WithValue(context.Background(), ContextAPIKey, APIKey{
		Key: "dvc_server_token_hash",
	})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpBucketingAPIMock()

	c := NewDVCClient("dvc_server_token_hash", &DVCOptions{DisableLocalBucketing: true, PollingInterval: 10 * time.Second})

	variable, err := c.DevCycleApi.Variable(auth,
		UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"},
		"test", true)
	if err != nil {
		t.Fatal(err)
		return
	}

	fmt.Println(variable)
}

func TestDVCClientService_VariableLocal(t *testing.T) {
	auth := context.WithValue(context.Background(), ContextAPIKey, APIKey{
		Key: "dvc_server_token_hash",
	})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	c := NewDVCClient("dvc_server_token_hash", &DVCOptions{PollingInterval: 10 * time.Second})

	c.configManager.SDKEvents = make(chan SDKEvent, 100)
	go func() {
		for {
			v := <-c.configManager.SDKEvents
			fmt.Println(v.Message, v.Error, v.FirstInitialization, v.Success)
		}
	}()

	variable, err := c.DevCycleApi.Variable(auth,
		UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"},
		"test", true)
	if err != nil {
		t.Fatal(err)
		return
	}

	fmt.Println(variable)
}

func TestDVCClientService_TrackLocal_QueueEvent(t *testing.T) {
	auth := context.WithValue(context.Background(), ContextAPIKey, APIKey{
		Key: "dvc_server_token_hash",
	})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	c := NewDVCClient("dvc_server_token_hash", &DVCOptions{PollingInterval: 10 * time.Second})

	track, err := c.DevCycleApi.Track(auth, UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"}, DVCEvent{
		Type_:       "customEvent",
		Target:      "",
		CustomType:  "",
		UserId:      "",
		Date:        0,
		ClientDate:  0,
		Value:       0,
		FeatureVars: nil,
		MetaData:    nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(track)
}

func httpBucketingAPIMock() {
	httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test",
		func(req *http.Request) (*http.Response, error) {

			resp := httpmock.NewStringResponse(200, `{"value": true, "_id": "614ef6ea475129459160721a", "key": "test", "type": "Boolean"}`)
			resp.Header.Set("Etag", "TESTING")
			return resp, nil
		},
	)
}
