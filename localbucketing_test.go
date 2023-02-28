package devcycle

import (
	_ "embed"
	"fmt"
	"github.com/jarcoal/httpmock"
	"testing"
)

func TestDevCycleLocalBucketing_Initialize(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	localBucketing := DevCycleLocalBucketing{}
	var err error
	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_Initialize(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	for i := 0; i < b.N; i++ {
		localBucketing := DevCycleLocalBucketing{}
		var err error
		err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDevCycleLocalBucketing_GenerateBucketedConfigForUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(test_config)
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}

	genConfig, err := localBucketing.GenerateBucketedConfigForUser(
		`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing" }`)
	if err != nil {
		return
	}
	fmt.Println(genConfig)
	if genConfig.Variables["test"].Value.(bool) == true {
		fmt.Println("Correctly bucketed user.")
	} else {
		t.Fatal("Incorrectly bucketed user.")
	}

}

func TestDevCycleLocalBucketing_StoreConfig(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(test_config)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDevCycleLocalBucketing_StoreConfig(b *testing.B) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		err = localBucketing.StoreConfig(test_config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDevCycleLocalBucketing_SetPlatformData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
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
	httpConfigMock(200)

	localBucketing := DevCycleLocalBucketing{}
	var err error

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{}, NewConfiguration(&DVCOptions{}))
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.StoreConfig(test_config)
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
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
