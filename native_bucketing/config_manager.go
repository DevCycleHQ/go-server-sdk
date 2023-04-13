package native_bucketing

import (
	"fmt"
	"sync"
)

var internalConfigs = make(map[string]*configBody)
var configMutex = &sync.RWMutex{}

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
	if val, ok := internalConfigs[sdkKey]; ok && val != nil && val.etag == etag {
		return nil
	}
	config, err := newConfig(rawJSON, etag)
	if err != nil {
		return err
	}
	internalConfigs[sdkKey] = &config
	return nil
}
