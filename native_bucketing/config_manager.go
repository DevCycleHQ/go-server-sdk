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

func SetConfig(rawJSON []byte, sdkKey, etag string, eventQueue ...*EventQueue) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	config, err := newConfig(rawJSON, etag)
	if err != nil {
		return err
	}
	internalConfigs[sdkKey] = config
	if eventQueue != nil && len(eventQueue) > 0 {
		eventQueue[0].MergeAggEventQueueKeys(config)
	}
	return nil
}

func clearConfigs() {
	configMutex.Lock()
	defer configMutex.Unlock()
	internalConfigs = make(map[string]*configBody)
}
