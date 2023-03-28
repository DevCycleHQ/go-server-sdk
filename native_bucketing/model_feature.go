package native_bucketing

import (
	"encoding/json"
	"gopkg.in/validator.v2"
)

type Feature struct {
	Id            string                 `json:"_id"`
	Type          string                 `json:"type" validate:"regexp=^(release|experiment|permission|ops)$"`
	Key           string                 `json:"key"`
	Variations    []Variation            `json:"variations"`
	Configuration FeatureConfiguration   `json:"configuration"`
	Settings      map[string]interface{} `json:"settings"`
}

func (f *Feature) FromJSON(js string) (err error, rt Feature) {
	var clss Feature
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type Variation struct {
	Id        string              `json:"_id"`
	Name      string              `json:"name"`
	Key       string              `json:"key"`
	Variables []VariationVariable `json:"variables"`
}

func (v *Variation) FromJSON(js string) (err error, rt Variation) {
	var clss Variation
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type VariationVariable struct {
	Var   string `json:"_var"`
	Value string `json:"value"`
}

func (v *VariationVariable) FromJSON(js string) (err error, rt VariationVariable) {
	var clss VariationVariable
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}
