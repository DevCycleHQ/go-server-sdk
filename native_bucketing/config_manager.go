package native_bucketing

import (
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"
)

var internalConfigs = make(map[string]*configBody)
var configMutex = &sync.RWMutex{}

// use a single instance of Validate, it caches struct info
var validate *validator.Validate

func init() {
	validate = validator.New()
}

func getConfig(sdkKey string) (*configBody, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if val, ok := internalConfigs[sdkKey]; ok && val != nil {
		return val, nil
	}
	return nil, fmt.Errorf("config not initialized")
}

func SetConfig(rawJSON []byte, sdkKey, etag string) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	config, err := newConfig(rawJSON, etag)
	if err != nil {
		return err
	}
	if err := validate.Struct(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	internalConfigs[sdkKey] = &config
	return nil
}

func clearConfigs() {
	configMutex.Lock()
	defer configMutex.Unlock()
	internalConfigs = make(map[string]*configBody)
}
