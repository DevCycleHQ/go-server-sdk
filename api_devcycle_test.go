package devcycle

import (
	"context"
	"fmt"
	"github.com/jarcoal/httpmock"
	"net/http"
	"testing"
)

func TestDVCClientService_AllFeatures_Local(t *testing.T) {
	auth := context.WithValue(context.Background(), ContextAPIKey, APIKey{
		Key: "dvc_server_fake_key",
	})
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "*",
		func(req *http.Request) (*http.Response, error) {
			fmt.Println(req.URL.String())
			resp, err := httpmock.NewJsonResponse(200, `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`)
			return resp, err
		},
	)

	c := NewDVCClient("dvc_server_fake_key")
	err := c.localBucketing.StoreConfig("dvc_server_fake_key", `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`)
	if err != nil {
		t.Fatal(err)
		return
	}
	c.configManager.SDKEvents = make(chan SDKEvent)
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

func TestDVCClientService_AllFeatures(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://config",
		func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Accept") != "application/json" {
				t.Errorf("Expected Accept: application/json header, got: %s", req.Header.Get("Accept"))
			}
			resp, err := httpmock.NewJsonResponse(200, map[string]interface{}{
				"value": "fixed",
			})
			return resp, err
		},
	)
}
