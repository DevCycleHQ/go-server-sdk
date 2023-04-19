package native_bucketing

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
