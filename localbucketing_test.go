//go:build !native_bucketing

package devcycle

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/devcyclehq/go-server-sdk/v2/proto"
	"github.com/jarcoal/httpmock"
)

func TestDevCycleLocalBucketing_Initialize(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	wasmMain := WASMMain{}
	err := wasmMain.Initialize(nil)
	fatalErr(t, err)

	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})
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
		err := wasmMain.Initialize(nil)
		if err != nil {
			b.Fatal(err)
		}
		localBucketing := WASMLocalBucketingClient{}
		err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})
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
	err := wasmMain.Initialize(nil)
	fatalErr(t, err)
	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig([]byte(test_config))
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformDataJSON([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		t.Fatal(err)
	}

	genConfig, err := localBucketing.GenerateBucketedConfigForUser(
		User{
			UserId:      "j_test",
			DeviceModel: "testing",
		},
	)
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
	err := wasmMain.Initialize(nil)
	fatalErr(t, err)
	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})

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
	err := wasmMain.Initialize(&Options{})
	if err != nil {
		b.Fatal(err)
	}
	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err = localBucketing.StoreConfig([]byte(test_large_config))
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
	err := wasmMain.Initialize(nil)
	fatalErr(t, err)
	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformDataJSON([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_GenerateBucketedConfigForUser(b *testing.B) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	wasmMain := WASMMain{}
	err := wasmMain.Initialize(nil)
	if err != nil {
		b.Fatal(err)
	}
	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.StoreConfig([]byte(test_config))
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.SetPlatformDataJSON([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := localBucketing.GenerateBucketedConfigForUser(
			User{
				UserId:      "j_test",
				DeviceModel: "testing",
			},
		)
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
	err := wasmMain.Initialize(nil)
	if err != nil {
		b.Fatal(err)
	}
	localBucketing := WASMLocalBucketingClient{}

	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{
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

	err = localBucketing.SetPlatformDataJSON([]byte(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`))
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

func TestDevCycleLocalBucketing_newAssemblyScriptNoPoolByteArray(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	wasmMain := WASMMain{}
	err := wasmMain.Initialize(nil)
	fatalErr(t, err)
	localBucketing := WASMLocalBucketingClient{}
	err = localBucketing.Initialize(&wasmMain, test_environmentKey, GeneratePlatformData(), &Options{})

	if err != nil {
		t.Fatal(err)
	}
	var memPtr int32
	memPtr, err = localBucketing.newAssemblyScriptNoPoolByteArray([]byte(test_config))

	if err != nil {
		t.Fatal(err)
	}

	if memPtr == 0 {
		t.Fatal("Pointer to byte array header is 0")
	}
}
