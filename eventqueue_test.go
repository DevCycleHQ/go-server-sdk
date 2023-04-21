package devcycle

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jarcoal/httpmock"
)

func TestEventQueue_QueueEvent(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewClient("dvc_server_token_hash", &Options{})
	fatalErr(t, err)

	_, err = c.Track(User{UserId: "j_test", DeviceModel: "testing"},
		Event{Target: "customevent", Type_: "event"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEventQueue_QueueEvent_100_DropEvent(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)

	c, err := NewClient("dvc_server_token_hash", &Options{MaxEventQueueSize: 100, FlushEventQueueSize: 10})
	fatalErr(t, err)

	errored := false
	for i := 0; i < 1000; i++ {
		log.Println(i)
		_, err = c.Track(User{UserId: "j_test", DeviceModel: "testing"},
			Event{Target: "customevent"})
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
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpConfigMock(200)
	httpEventsApiMock()
	c, err := NewClient("dvc_server_token_hash", &Options{MaxEventQueueSize: 100, FlushEventQueueSize: 10})
	fatalErr(t, err)

	for i := 0; i <= 100; i++ {
		_, err = c.Track(User{UserId: "j_test", DeviceModel: "testing"},
			Event{Target: "customevent", Type_: "event"})
		if err != nil {
			log.Println(err)
			break
		}
	}

	require.Eventually(t, func() bool {
		return httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"] == 10
	}, 1*time.Second, 100*time.Millisecond)

}
