package native_bucketing

import (
	"fmt"
)

var platformDataMap = map[string]PlatformData{}

func GetPlatformData(token string) (data *PlatformData, err error) {
	if data, ok := platformDataMap[token]; ok {
		return &data, nil
	}
	return nil, fmt.Errorf("no platform data found for token %s", token)
}

func SetPlatformData(sdkKey string, data PlatformData) {
	platformDataMap[sdkKey] = data
}

var clientCustomData = map[string]map[string]interface{}{}

func GetClientCustomData(sdkKey string) map[string]interface{} {
	if data, ok := clientCustomData[sdkKey]; ok {
		return data
	}
	return map[string]interface{}{}
}

func SetClientCustomData(sdkKey string, data map[string]interface{}) {
	clientCustomData[sdkKey] = data
}
