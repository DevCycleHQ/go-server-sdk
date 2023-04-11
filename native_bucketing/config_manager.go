package native_bucketing

import (
	"fmt"
	"sync"
)

var internalConfig *configBody
var configMutex = &sync.RWMutex{}

func getConfig() (*configBody, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if internalConfig == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	return internalConfig, nil
}

func SetConfig(rawJSON []byte, etag string) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	if internalConfig != nil && internalConfig.etag == etag {
		return nil
	}
	config, err := newConfig(rawJSON, etag)
	if err != nil {
		return err
	}
	internalConfig = &config
	return err
}
