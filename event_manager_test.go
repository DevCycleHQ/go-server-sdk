package devcycle

import (
	"encoding/json"
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/api"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jarcoal/httpmock"
)

func TestEventManager_QueueEvent(t *testing.T) {
	sdkKey, _ := httpConfigMock(200)
	c, err := NewClient(sdkKey, &Options{})
	require.NoError(t, err)
	defer func(c *Client) {
		_ = c.Close()
	}(c)

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
	defer func(c *Client) {
		_ = c.Close()
	}(c)

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

func TestEventManager_AggVariableEvaluatedReason(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)
	user := User{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}

	evalReasons := map[string]float64{}
	httpEventsApiMockWithCallback(func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(201, `{}`)
		var body BatchEventsBody
		rawBody, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(rawBody, &body)

		for _, b := range body.Batch {
			for _, e := range b.Events {
				if e.Type_ == api.EventType_AggVariableEvaluated {
					if val, ok := e.MetaData["eval"]; ok {
						if eval, ok := val.(map[string]interface{}); ok {
							for k, v := range eval {
								if count, ok := v.(float64); ok {
									evalReasons[k] += count
									fmt.Printf("%s - Eval count: %f, Total Count: %f\n", k, count, evalReasons[k])
								} else {
									t.Errorf("Expected eval count to be a float, got %T", v)
								}
							}
						} else {
							t.Errorf("Expected eval reason to be a map, got %T", val)
						}
					} else {
						t.Error("Expected eval reason to be present in event metadata")
					}
				}
			}
		}
		if err != nil {
			t.Fatalf("Error unmarshalling request body: %v", err)
		}
		return resp, nil
	})
	c, err := NewClient(sdkKey, &Options{
		MaxEventQueueSize:       100,
		FlushEventQueueSize:     10,
		ConfigPollingIntervalMS: time.Second,
		EventFlushIntervalMS:    time.Second,
	})
	require.NoError(t, err)
	defer func(c *Client) {
		_ = c.Close()
	}(c)
	require.Eventually(t, func() bool {
		return c.isInitialized && c.hasConfig()
	}, 1*time.Second, 100*time.Millisecond)

	for i := 0; i < 2*c.DevCycleOptions.FlushEventQueueSize; i++ {
		_, _ = c.Variable(user, "v-key-76", 69)
	}

	_ = c.FlushEvents()
	require.Eventually(t, func() bool {
		fmt.Println(httpmock.GetCallCountInfo())
		evalReasonsUpdated := len(evalReasons) > 0
		return httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"] > 0 && evalReasonsUpdated
	}, 3*time.Second, 100*time.Millisecond)
}

func TestEventManager_AggVariableDefaultedReason(t *testing.T) {
	sdkKey := generateTestSDKKey()
	httpCustomConfigMock(sdkKey, 200, test_large_config, false)
	user := User{UserId: "dontcare", DeviceModel: "testing", CustomData: map[string]interface{}{"data-key-7": "3yejExtXkma4"}}

	defaultReasons := map[string]float64{}
	httpEventsApiMockWithCallback(func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(201, `{}`)
		var body BatchEventsBody
		rawBody, _ := io.ReadAll(req.Body)
		err := json.Unmarshal(rawBody, &body)

		for _, b := range body.Batch {
			for _, e := range b.Events {
				if e.Type_ == api.EventType_AggVariableDefaulted {
					if val, ok := e.MetaData["eval"]; ok {
						if eval, ok := val.(map[string]interface{}); ok {
							for k, v := range eval {
								if count, ok := v.(float64); ok {
									defaultReasons[k] += count
									fmt.Printf("%s - Default count: %f, Total Count: %f\n", k, count, defaultReasons[k])
								} else {
									t.Errorf("Expected default count to be a float, got %T", v)
								}
							}
						} else {
							t.Errorf("Expected default reason to be a map, got %T", val)
							fmt.Println(val)
						}
					} else {
						t.Error("Expected eval reason to be present in event metadata")
					}
				}
			}
		}
		if err != nil {
			t.Fatalf("Error unmarshalling request body: %v", err)
		}
		return resp, nil
	})
	c, err := NewClient(sdkKey, &Options{
		MaxEventQueueSize:       100,
		FlushEventQueueSize:     10,
		ConfigPollingIntervalMS: time.Second,
		EventFlushIntervalMS:    time.Second,
	})
	require.NoError(t, err)
	defer func(c *Client) {
		_ = c.Close()
	}(c)
	require.Eventually(t, func() bool {
		return c.isInitialized && c.hasConfig()
	}, 1*time.Second, 100*time.Millisecond)

	for i := 0; i < 10; i++ {
		_, _ = c.Variable(user, "variable-doesn't-exist", 69)
	}

	_ = c.FlushEvents()
	require.Eventually(t, func() bool {
		fmt.Println(httpmock.GetCallCountInfo())
		evalReasonsUpdated := len(defaultReasons) > 0
		return httpmock.GetCallCountInfo()["POST https://events.devcycle.com/v1/events/batch"] > 0 && evalReasonsUpdated
	}, 3*time.Second, 100*time.Millisecond)
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
	defer func(c *Client) {
		_ = c.Close()
	}(c)
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
