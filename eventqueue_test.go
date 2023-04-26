package devcycle

import (
	"log"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestEventQueue_QueueEvent(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{})
	fatalErr(t, err)

	_, err = c.Track(DVCUser{UserId: "j_test", DeviceModel: "testing"},
		DVCEvent{Target: "customevent", Type_: "event"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEventQueue_QueueEvent_100_DropEvent(t *testing.T) {
	skipIfNative(t)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{MaxEventQueueSize: 100, FlushEventQueueSize: 10})
	fatalErr(t, err)

	errored := false
	for i := 0; i < 1000; i++ {
		log.Println(i)
		_, err = c.Track(DVCUser{UserId: "j_test", DeviceModel: "testing"},
			DVCEvent{Target: "customevent"})
		if err != nil {
			errored = true
			log.Println(err)
			break
		}
	}
	if !errored {
		t.Fatal("Did not get dropped event warning.")
	}
}

func TestEventQueue_QueueEvent_100_Flush(t *testing.T) {
	//skipIfNative(t)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	httpEventsApiMock()
	c, err := NewDVCClient("dvc_server_token_hash", &DVCOptions{MaxEventQueueSize: 100, FlushEventQueueSize: 10})
	fatalErr(t, err)

	for i := 0; i <= 100; i++ {
		log.Println(i)
		_, err = c.Track(DVCUser{UserId: "j_test", DeviceModel: "testing"},
			DVCEvent{Target: "customevent", Type_: "event"})
		if err != nil {
			log.Println(err)
			break
		}
	}
	if httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"] != 10 {
		t.Fatal("Expected 10 flushes to be forced. Got ", httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"])
	}
}
