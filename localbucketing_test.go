package devcycle

import (
	_ "embed"
	"fmt"
	"github.com/jarcoal/httpmock"
	"testing"
	"time"
)

func TestDevCycleLocalBucketing_Initialize(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	localBucketing := DevCycleLocalBucketing{}
	var err error
	localBucketing.SetSDKToken(test_environmentKey)
	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevCycleLocalBucketing_GenerateBucketedConfigForUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	localBucketing := DevCycleLocalBucketing{}
	var err error

	localBucketing.SetSDKToken(test_environmentKey)

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(test_environmentKey, test_config)
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
	localBucketing.SetSDKToken(test_environmentKey)

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.StoreConfig(test_environmentKey, test_config)
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

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		err = localBucketing.StoreConfig(test_environmentKey, test_config)
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

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
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

	localBucketing.SetSDKToken(test_environmentKey)
	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		b.Fatal(err)
	}

	err = localBucketing.StoreConfig(test_environmentKey, test_config)
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

func TestEnvironmentConfigManager_Initialize(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	var err error

	localBucketing := DevCycleLocalBucketing{}
	localBucketing.SetSDKToken(test_environmentKey)

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.configManager.Initialize(test_environmentKey, &DVCOptions{PollingInterval: 500 * time.Millisecond})
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
	httpConfigMock(200)

	var err error

	localBucketing := DevCycleLocalBucketing{}
	localBucketing.SetSDKToken(test_environmentKey)

	err = localBucketing.Initialize(test_environmentKey, &DVCOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = localBucketing.configManager.Initialize(test_environmentKey, &DVCOptions{PollingInterval: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	err = localBucketing.SetPlatformData(`{"platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}
	user, err := localBucketing.GenerateBucketedConfigForUser(
		`{"user_id": "j_test", "platform": "golang-testing", "sdkType": "server", "platformVersion": "testing", "deviceModel": "testing", "sdkVersion":"testing"}`)
	if err != nil {
		t.Fatal(err)
	}

	if user.Project.Id != "6216420c2ea68943c8833c09" {
		t.Fatalf("Project does not match. %s, %s", user.Project.Id, "6216420c2ea68943c8833c09")
	}

	localBucketing.configManager.cancel()
	localBucketing.configManager.context.Done()
	fmt.Println("done")
}
