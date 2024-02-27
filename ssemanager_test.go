package devcycle

import "testing"

const (
	test_sseRawMessage = "{\"etag\":\"\\\"1d0be8bbc8e607590b11131237d608c0\\\"\",\"lastModified\":1708618754000}"
)

func TestSSEManager_ParseMessage(t *testing.T) {
	m := &SSEManager{}
	message, err := m.parseMessage([]byte(test_sseRawMessage))
	if err != nil {
		t.Fatal(err)
	}
	if message.Etag != "\"1d0be8bbc8e607590b11131237d608c0\"" {
		t.Fatal("message.Etag != \"1d0be8bbc8e607590b11131237d608c0\"")
	}
	if message.LastModified != 1708618754000 {
		t.Fatal("message.LastModified != 1708618754000")
	}
}
