package api

type MinimalConfig struct {
	SSE                 *SSEHost             `json:"sse,omitempty"`
	Project             *Project             `json:"project,omitempty"`
	Environment         *Environment         `json:"environment,omitempty"`
	ConfigETag          string               `json:"configETag,omitempty"`
	ConfigLastModified  string               `json:"configLastModified,omitempty"`
	ProjectMetadata     *ProjectMetadata     `json:"projectMetadata,omitempty"`
	EnvironmentMetadata *EnvironmentMetadata `json:"environmentMetadata,omitempty"`
}

type SSEHost struct {
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path,omitempty"`
}

type ProjectMetadata struct {
	Id  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

type EnvironmentMetadata struct {
	Id  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}
