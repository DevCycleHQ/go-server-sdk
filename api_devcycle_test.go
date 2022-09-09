package devcycle

import (
	"context"
	"fmt"
	"github.com/jarcoal/httpmock"
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
func TestDVCClientService_AllVariables(t *testing.T) {
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

func TestDVCClientService_Variable(t *testing.T) {
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
