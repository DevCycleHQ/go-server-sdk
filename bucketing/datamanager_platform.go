package bucketing

var clientCustomData = map[string]map[string]interface{}{}

func GetClientCustomData(sdkKey string) map[string]interface{} {
	if data, ok := clientCustomData[sdkKey]; ok {
		return data
	}
	// Nil maps are safe to read but not write, and this avoids an allocation
	return nil
}

func SetClientCustomData(sdkKey string, data map[string]interface{}) {
	clientCustomData[sdkKey] = data
}
