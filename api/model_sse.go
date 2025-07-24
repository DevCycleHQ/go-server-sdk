package api

type MinimalConfig struct {
	SSE         *SSEHost     `json:"sse,omitempty"`
	Project     *Project     `json:"project,omitempty"`
	Environment *Environment `json:"environment,omitempty"`
}

type SSEHost struct {
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path,omitempty"`
}
