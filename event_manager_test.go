package devcycle

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jarcoal/httpmock"
)

func TestEventManager_QueueEvent(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	c, err := NewClient(sdkKey, &Options{})
	require.NoError(t, err)
	defer c.Close()

	_, err = c.Track(User{UserId: "j_test", DeviceModel: "testing"},
		Event{Target: "customevent", Type_: "event"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEventManager_QueueEvent_100_DropEvent(t *testing.T) {

	sdkKey, _ := httpConfigMock(200)

	c, err := NewClient(sdkKey, &Options{MaxEventQueueSize: 100, FlushEventQueueSize: 10})
	require.NoError(t, err)
	defer c.Close()

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

func TestEventManager_QueueEvent_100_Flush(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	c, err := NewClient(sdkKey, &Options{
		MaxEventQueueSize:       100,
		FlushEventQueueSize:     10,
		ConfigPollingIntervalMS: time.Second,
		EventFlushIntervalMS:    time.Second,
	})
	require.NoError(t, err)
	defer c.Close()
	require.Eventually(t, func() bool {
		return c.isInitialized && c.hasConfig()
	}, 1*time.Second, 100*time.Millisecond)
	// Track up to FlushEventQueueSize events
	for i := 0; i < c.DevCycleOptions.FlushEventQueueSize; i++ {
		_, err = c.Track(User{UserId: "j_test", DeviceModel: "testing"},
			Event{Target: "customevent", Type_: "event"})
		if err != nil {
			t.Fatalf("Error tracking event: %v", err)
		}
	}
	// Wait for raw event queue to drain
	require.Eventually(t, func() bool {
		queueLength, _ := c.eventQueue.internalQueue.UserQueueLength()
		return queueLength >= 10
	}, 1*time.Second, 100*time.Millisecond)

	// Track one more event to trigger an automatic flush
	_, err = c.Track(User{UserId: "j_test", DeviceModel: "testing"}, Event{Target: "customevent", Type_: "event"})
	if err != nil {
		t.Fatalf("Error tracking event: %v", err)
	}

	// Wait for raw event queue to drain
	require.Eventually(t, func() bool {
		flushed, reported, dropped := c.eventQueue.Metrics()
		return flushed == 1 && reported == 1 && dropped == 0
	}, 1*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		fmt.Println(httpmock.GetCallCountInfo())
		return httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"] >= 1
	}, 1*time.Second, 100*time.Millisecond)

}
