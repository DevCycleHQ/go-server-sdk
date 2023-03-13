package devcycle

import (
	_ "embed"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/proto"
	"github.com/jarcoal/httpmock"
	"testing"
)

func TestDevCycleLocalBucketing_Initialize(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	wasmMain := WASMMain{}
	err := wasmMain.Initialize()

	localBucketing := DevCycleLocalBucketing{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_Initialize(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	for i := 0; i < b.N; i++ {
		wasmMain := WASMMain{}
		err := wasmMain.Initialize()
		localBucketing := DevCycleLocalBucketing{}
		err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDevCycleLocalBucketing_GenerateBucketedConfigForUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	wasmMain := WASMMain{}
	err := wasmMain.Initialize()
	localBucketing := DevCycleLocalBucketing{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig([]byte(test_config))
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformData([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		t.Fatal(err)
	}

	genConfig, err := localBucketing.GenerateBucketedConfigForUser(
		`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing" }`)
	if err != nil {
		return
	}
	fmt.Println(genConfig)
	if genConfig.Variables["test"].Value != nil && genConfig.Variables["test"].Value.(bool) == true {
		fmt.Println("Correctly bucketed user.")
	} else {
		t.Fatal("Incorrectly bucketed user.")
	}

}

func TestDevCycleLocalBucketing_StoreConfig(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	wasmMain := WASMMain{}
	err := wasmMain.Initialize()
	localBucketing := DevCycleLocalBucketing{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})

	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig([]byte(test_config))
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_StoreConfig(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	wasmMain := WASMMain{}
	err := wasmMain.Initialize()
	localBucketing := DevCycleLocalBucketing{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		err = localBucketing.StoreConfig([]byte(test_config))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDevCycleLocalBucketing_SetPlatformData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	wasmMain := WASMMain{}
	err := wasmMain.Initialize()
	localBucketing := DevCycleLocalBucketing{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformData([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_GenerateBucketedConfigForUser(b *testing.B) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	wasmMain := WASMMain{}
	err := wasmMain.Initialize()
	localBucketing := DevCycleLocalBucketing{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{})
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.StoreConfig([]byte(test_config))
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.SetPlatformData([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := localBucketing.GenerateBucketedConfigForUser(
			`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDevCycleLocalBucketing_VariableForUser_PB(b *testing.B) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	wasmMain := WASMMain{}
	err := wasmMain.Initialize()
	localBucketing := DevCycleLocalBucketing{}

	err = localBucketing.Initialize(&wasmMain, test_environmentKey, &DVCOptions{
		AdvancedOptions: AdvancedOptions{
			MaxMemoryAllocationBuckets: 1,
		},
	})

	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.StoreConfig([]byte(test_config))
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.SetPlatformData([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		b.Fatal(err)
	}

	userPB := &proto.DVCUser_PB{
		UserId: "userid",
	}

	// package everything into the root params object
	paramsPB := proto.VariableForUserParams_PB{
		SdkKey:           test_environmentKey,
		VariableKey:      "test",
		VariableType:     proto.VariableType_PB(localBucketing.VariableTypeCodes.Boolean),
		User:             userPB,
		ShouldTrackEvent: false,
	}

	paramsBuffer, err := paramsPB.MarshalVT()

	if err != nil {
		b.Fatal(err)
	}
	_, err = localBucketing.VariableForUser_PB(paramsBuffer)

	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.assemblyScriptCollect()

	if err != nil {
		b.Fatal(err)
	}

	_, err = localBucketing.VariableForUser_PB(paramsBuffer)

	if err != nil {
		b.Fatal(err)
	}
}
