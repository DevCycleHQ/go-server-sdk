package native_bucketing

type Feature struct {
	Id            string                 `json:"_id"`
	Type          string                 `json:"type" validate:"regexp=^(release|experiment|permission|ops)$"`
	Key           string                 `json:"key"`
	Variations    []Variation            `json:"variations"`
	Configuration FeatureConfiguration   `json:"configuration"`
	Settings      map[string]interface{} `json:"settings"`
}

type Variation struct {
	Id        string              `json:"_id"`
	Name      string              `json:"name"`
	Key       string              `json:"key"`
	Variables []VariationVariable `json:"variables"`
}

type VariationVariable struct {
	Var   string      `json:"_var"`
	Value interface{} `json:"value"`
}
