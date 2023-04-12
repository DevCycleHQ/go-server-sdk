package native_bucketing

import (
	"fmt"
	"runtime"
)

var platformDataMap = map[string]PlatformData{}
var emptyPlatformData = PlatformData{}

var VERSION = "0.0.0"

func GetPlatformData(token string) (data *PlatformData, err error) {
	if data, ok := platformDataMap[token]; ok {
		return &data, nil
	}
	return nil, fmt.Errorf("no platform data found for token %s", token)
}

func SetPlatformData(token string) {
	data := PlatformData{
		Platform:        "Go",
		PlatformVersion: runtime.Version(),
		SdkType:         "server",
		SdkVersion:      VERSION,
	}
	platformDataMap[token] = data
}

var clientCustomData = map[string]map[string]interface{}{}

func GetClientCustomData(token string) (map[string]interface{}, error) {
	if data, ok := clientCustomData[token]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("no client custom data found for token %s", token)
}

func SetClientCustomData(token string, data map[string]interface{}) {
	clientCustomData[token] = data
}
