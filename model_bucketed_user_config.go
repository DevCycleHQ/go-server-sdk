package devcycle

type BucketedUserConfig struct {
	Project             Project             `json:"project"`
	Environment         Environment         `json:"environment"`
	Features            map[string]Feature  `json:"features"`
	FeatureVariationMap map[string]string   `json:"featureVariationMap"`
	Variables           map[string]Variable `json:"variables"`
	KnownVariableKeys   []float64           `json:"knownVariableKeys"`
}
