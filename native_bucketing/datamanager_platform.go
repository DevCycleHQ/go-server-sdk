package native_bucketing

import "errors"

var platformDataMap = map[string]PlatformData{}
var emptyPlatformData = PlatformData{}

func GetPlatformData(token string) (err error, data PlatformData) {
	if platformDataMap[token] != emptyPlatformData {
		return nil, platformDataMap[token]
	}
	return errors.New("No platform data found for token " + token), data
}

func SetPlatformData(token string, data PlatformData) {
	platformDataMap[token] = data
}
