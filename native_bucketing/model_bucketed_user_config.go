package native_bucketing

type BucketedUserConfig struct {
	Project            PublicProject               `json:"project"`
	Environment        PublicEnvironment           `json:"environment"`
	Features           map[string]SDKFeature       `json:"features"`
	FeatureVariations  map[string]string           `json:"featureVariationMap"`
	Variables          map[string]SDKVariable      `json:"variables"`
	VariableVariations map[string]FeatureVariation `json:"variableVariationMap"`
	KnownVariableKeys  []int64                     `json:"knownVariableKeys"`
}

type SDKFeature struct {
	Id            string `json:"_id"`
	Type          string `json:"type"`
	Key           string `json:"key"`
	VariationId   string `json:"_variation"`
	VariationName string `json:"variationName"`
	VariationKey  string `json:"variationKey"`
	EvalReason    string `json:"evalReason"`
}

func SDKFeatureFromJSONObj(obj map[string]interface{}) SDKFeature {
	var sdkFeature SDKFeature
	sdkFeature.Id = obj["_id"].(string)
	sdkFeature.Type = obj["type"].(string)
	sdkFeature.Key = obj["key"].(string)
	sdkFeature.VariationId = obj["_variation"].(string)
	sdkFeature.VariationName = obj["variationName"].(string)
	sdkFeature.VariationKey = obj["variationKey"].(string)
	sdkFeature.EvalReason = obj["evalReason"].(string)
	return sdkFeature
}

type SDKVariable struct {
	Id         string `json:"_id"`
	Type       string `json:"type"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	EvalReason string `json:"evalReason"`
}
