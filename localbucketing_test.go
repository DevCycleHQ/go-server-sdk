package devcycle

import (
	_ "embed"
	"fmt"
	"github.com/jarcoal/httpmock"
	"net/http"
	"testing"
	"time"
)

func TestDevCycleLocalBucketing_Initialize(t *testing.T) {
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevCycleLocalBucketing_GenerateBucketedConfigForUser(t *testing.T) {
	environmentKey := "dvc_server_token_hash"
	config := `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(environmentKey, config)
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}

	genConfig, err := localBucketing.GenerateBucketedConfigForUser(environmentKey,
		`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing" }`)
	if err != nil {
		return
	}
	fmt.Println(genConfig)
}

func TestDevCycleLocalBucketing_StoreConfig(t *testing.T) {

	environmentKey := "dvc_server_token_hash"
	config := `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(environmentKey, config)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_StoreConfig(b *testing.B) {

	environmentKey := "dvc_server_token_hash"
	config := `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		err = localBucketing.StoreConfig(environmentKey, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDevCycleLocalBucketing_SetPlatformData(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_GenerateBucketedConfigForUser(b *testing.B) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	environmentKey := "dvc_server_token_hash"
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize()
	if err != nil {
		b.Fatal(err)
	}

	config := `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`

	err = localBucketing.StoreConfig(environmentKey, config)
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := localBucketing.GenerateBucketedConfigForUser(environmentKey,
			`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestEnvironmentConfigManager_Initialize(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	environmentKey := "dvc_server_token_hash"
	var err error

	localBucketing := DevCycleLocalBucketing{}

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.configManager.Initialize(environmentKey, &DVCOptions{PollingInterval: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}
	localBucketing.configManager.cancel()
	localBucketing.configManager.context.Done()
	fmt.Println("done")
}

func TestEnvironmentConfigManager_LocalBucketing(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock()

	environmentKey := "dvc_server_token_hash"
	var err error

	localBucketing := DevCycleLocalBucketing{}

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.configManager.Initialize(environmentKey, &DVCOptions{PollingInterval: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}
	user, err := localBucketing.GenerateBucketedConfigForUser(environmentKey,
		`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}

	if user.Project.Id != "6216420c2ea68943c8833c09" {
		t.Fatalf("Project does not match. %s, %s", user.Project, "6216420c2ea68943c8833c09")
	}

	localBucketing.configManager.cancel()
	localBucketing.configManager.context.Done()
	fmt.Println("done")
}

func httpConfigMock() {
	httpmock.RegisterResponder("GET", "https://config-cdn.devcycle.com/config/v1/server/dvc_server_token_hash.json",
		func(req *http.Request) (*http.Response, error) {

			resp := httpmock.NewStringResponse(200, `{"project":{"_id":"6216420c2ea68943c8833c09","key":"default","a0_organization":"org_NszUFyWBFy7cr95J"},"environment":{"_id":"6216420c2ea68943c8833c0b","key":"development"},"features":[{"_id":"6216422850294da359385e8b","key":"test","type":"release","variations":[{"variables":[{"_var":"6216422850294da359385e8d","value":true}],"name":"Variation On","key":"variation-on","_id":"6216422850294da359385e8f"},{"variables":[{"_var":"6216422850294da359385e8d","value":false}],"name":"Variation Off","key":"variation-off","_id":"6216422850294da359385e90"}],"configuration":{"_id":"621642332ea68943c8833c4a","targets":[{"distribution":[{"percentage":0.5,"_variation":"6216422850294da359385e8f"},{"percentage":0.5,"_variation":"6216422850294da359385e90"}],"_audience":{"_id":"621642332ea68943c8833c4b","filters":{"operator":"and","filters":[{"values":[],"type":"all","filters":[]}]}},"_id":"621642332ea68943c8833c4d"}],"forcedUsers":{}}}],"variables":[{"_id":"6216422850294da359385e8d","key":"test","type":"Boolean"}],"variableHashes":{"test":2447239932}}`)
			resp.Header.Set("Etag", "TESTING")
			return resp, nil
		},
	)
}
