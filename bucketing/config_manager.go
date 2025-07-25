package bucketing

import (
	"fmt"
	"sync"
)

var internalConfigs = make(map[string]*configBody)
var internalRawConfigs = make(map[string][]byte)
var configMutex = &sync.RWMutex{}

func getConfig(sdkKey string) (*configBody, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if val, ok := internalConfigs[sdkKey]; ok && val != nil {
		return val, nil
	}
	return nil, fmt.Errorf("config not initialized")
}

func GetEtag(sdkKey string) string {
	config, err := getConfig(sdkKey)
	if err != nil {
		return ""
	}
	return config.etag
}

func GetRayId(sdkKey string) string {
	config, err := getConfig(sdkKey)
	if err != nil {
		return ""
	}
	return config.rayId
}

func GetLastModified(sdkKey string) string {
	config, err := getConfig(sdkKey)
	if err != nil {
		return ""
	}
	return config.lastModified
}

func GetRawConfig(sdkKey string) []byte {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if val, ok := internalRawConfigs[sdkKey]; ok && val != nil {
		return val
	}
	return nil
}

func SetConfig(rawJSON []byte, sdkKey, etag, rayId, lastModified string, eventQueue ...*EventQueue) error {
	config, err := newConfig(rawJSON, etag, rayId, lastModified)
	if err != nil {
		return err
	}

	configMutex.Lock()
	internalConfigs[sdkKey] = config
	internalRawConfigs[sdkKey] = rawJSON
	configMutex.Unlock()

	// Call MergeAggEventQueueKeys outside of the configMutex to avoid deadlock
	if len(eventQueue) > 0 {
		eventQueue[0].MergeAggEventQueueKeys(config)
	}
	return nil
}

func HasConfig(sdkKey string) bool {
	configMutex.RLock()
	defer configMutex.RUnlock()
	_, ok := internalConfigs[sdkKey]
	return ok
}

func clearConfigs() {
	configMutex.Lock()
	defer configMutex.Unlock()
	internalConfigs = make(map[string]*configBody)
}
