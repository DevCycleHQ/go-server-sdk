package native_bucketing

import "errors"

var configDataMap = map[string]ConfigBody{}
var emptyConfigData = ConfigBody{}

func GetConfigData(token string) (err error, data ConfigBody) {
	if !emptyConfigData.Equals(configDataMap[token]) {
		return nil, configDataMap[token]
	}
	return errors.New("No config data found for token " + token), data
}

func SetConfigData(token string, data ConfigBody) {
	configDataMap[token] = data
}
