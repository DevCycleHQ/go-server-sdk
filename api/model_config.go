package api

type MinimalConfig struct {
	SSE *SSEHost `json:"sse,omitempty"`
}

type SSEHost struct {
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path,omitempty"`
}
