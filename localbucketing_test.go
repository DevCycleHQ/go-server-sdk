package devcycle

import (
	"fmt"
	"os"
	"testing"
)

func TestDevCycleLocalBucketing_Initialize(t *testing.T) {
	localBucketing := DevCycleLocalBucketing{}
	var err error

	wasmFile, err := os.ReadFile("bucketing-lib.release.wasm")
	if err != nil {
		t.Fatal(err)
	}
	localBucketing.wasm = wasmFile

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

	wasmFile, err := os.ReadFile("bucketing-lib.release.wasm")
	if err != nil {
		t.Fatal(err)
	}
	localBucketing.wasm = wasmFile

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

	wasmFile, err := os.ReadFile("bucketing-lib.release.wasm")
	if err != nil {
		t.Fatal(err)
	}
	localBucketing.wasm = wasmFile

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(environmentKey, config)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevCycleLocalBucketing_SetPlatformData(t *testing.T) {

	localBucketing := DevCycleLocalBucketing{}
	var err error

	wasmFile, err := os.ReadFile("bucketing-lib.release.wasm")
	if err != nil {
		t.Fatal(err)
	}
	localBucketing.wasm = wasmFile

	err = localBucketing.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}
}
