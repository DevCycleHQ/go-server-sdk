package devcycle

import (
	"testing"
)

const (
	test_sseFullData = "{\"id\":\"yATzPE/mOzgY:0\",\"timestamp\":1712853334259,\"channel\":\"dvc_server_4fedfbd7a1aef0848768c8fad8f4536ca57e0ba0_v1\",\"data\":\"{\\\"etag\\\":\\\"\\\\\\\"714bc6a9acb038971923289ee6ce665b\\\\\\\"\\\",\\\"lastModified\\\":1712853333000}\",\"name\":\"change\"}"
)

func TestSSEManager_ParseMessage(t *testing.T) {
	m := &SSEManager{}
	message, err := m.parseMessage([]byte(test_sseFullData))
	if err != nil {
		t.Fatal(err)
	}
	if message.Etag != "\"714bc6a9acb038971923289ee6ce665b\"" {
		t.Fatal("message.Etag != \"714bc6a9acb038971923289ee6ce665b\"")
	}
	if message.LastModified != 1712853333000 {
		t.Fatal("message.LastModified != 1712853333000")
	}
	if message.Type_ != "" {
		t.Fatal("message.Type_ != \"\"")
	}
}
