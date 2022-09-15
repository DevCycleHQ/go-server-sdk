package devcycle

import (
	"context"
	"github.com/jarcoal/httpmock"
	"log"
	"testing"
)

func TestEventQueue_QueueEvent(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	lb, err := InitializeLocalBucketing("dvc_server_token_hash", &DVCOptions{})
	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{}, lb)

	_, err = c.DevCycleApi.Track(context.Background(), UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"},
		DVCEvent{Target: "customevent"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEventQueue_QueueEvent_100_DropEvent(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	lb, err := InitializeLocalBucketing("dvc_server_token_hash", &DVCOptions{MaxEventsPerFlush: 100, MinEventsPerFlush: 10})

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{MaxEventsPerFlush: 100, MinEventsPerFlush: 10}, lb)

	errored := false
	for i := 0; i < 1000; i++ {
		log.Println(i)
		_, err = c.DevCycleApi.Track(context.Background(), UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"},
			DVCEvent{Target: "customevent"})
		if err != nil {
			errored = true
			log.Println(err)
			return
		}
	}
	if !errored {
		t.Fatal("Did not get dropped event warning.")
	}
}

func TestEventQueue_QueueEvent_100_Flush(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	httpEventsApiMock()
	lb, err := InitializeLocalBucketing("dvc_server_token_hash", &DVCOptions{MaxEventsPerFlush: 100, MinEventsPerFlush: 10})

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{MaxEventsPerFlush: 100, MinEventsPerFlush: 10}, lb)

	for i := 0; i < 101; i++ {
		log.Println(i)
		_, err = c.DevCycleApi.Track(context.Background(), UserData{UserId: "j_test", Platform: "golang-testing", SdkType: "server", PlatformVersion: "testing", DeviceModel: "testing", SdkVersion: "testing"},
			DVCEvent{Target: "customevent"})
		if err != nil {
			log.Println(err)
			break
		}
	}
	if httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"] != 10 {
		t.Fatal("Expected 10 flushes to be forced. Got ", httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"])
	}
}
