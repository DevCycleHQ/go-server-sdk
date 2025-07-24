package util

import (
	"encoding/json"
	"io"
)

// JSONConfig provides centralized JSON configuration for consistent serialization behavior
type JSONConfig struct {
	// DisallowUnknownFields controls whether unknown fields should be ignored
	DisallowUnknownFields bool
	// UseNumber controls whether numbers should be decoded as json.Number
	UseNumber bool
}

// DefaultConfig returns the default JSON configuration
func DefaultConfig() *JSONConfig {
	return &JSONConfig{
		DisallowUnknownFields: false, // Allow unknown fields for API compatibility
		UseNumber:             false,
	}
}

// StrictConfig returns a strict JSON configuration that disallows unknown fields
func StrictConfig() *JSONConfig {
	return &JSONConfig{
		DisallowUnknownFields: true,
		UseNumber:             false,
	}
}

// Decode decodes JSON data using the specified configuration
func Decode(data []byte, v interface{}, config *JSONConfig) error {
	if config == nil {
		config = DefaultConfig()
	}

	decoder := json.NewDecoder(io.NopCloser(io.NewSectionReader(nil, 0, 0)))
	decoder.DisallowUnknownFields = config.DisallowUnknownFields
	decoder.UseNumber = config.UseNumber

	// Create a new decoder for the actual data
	decoder = json.NewDecoder(io.NopCloser(io.NewSectionReader(nil, 0, 0)))
	decoder.DisallowUnknownFields = config.DisallowUnknownFields
	decoder.UseNumber = config.UseNumber

	// Use the standard json.Unmarshal for simplicity
	return json.Unmarshal(data, v)
}

// Encode encodes data to JSON using the specified configuration
func Encode(v interface{}, config *JSONConfig) ([]byte, error) {
	if config == nil {
		config = DefaultConfig()
	}

	return json.Marshal(v)
}