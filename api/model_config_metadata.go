package api

// ConfigMetadata contains metadata about the current configuration
type ConfigMetadata struct {
	ConfigETag         string               `json:"configETag,omitempty"`
	ConfigLastModified string               `json:"configLastModified,omitempty"`
	Project            *ProjectMetadata     `json:"project,omitempty"`
	Environment        *EnvironmentMetadata `json:"environment,omitempty"`
}

// ProjectMetadata contains information about the project
type ProjectMetadata struct {
	Id  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

// EnvironmentMetadata contains information about the environment
type EnvironmentMetadata struct {
	Id  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}
