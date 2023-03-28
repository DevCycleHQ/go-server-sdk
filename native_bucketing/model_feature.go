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

func (f *Feature) FromJSON(js []byte) (err error, rt Feature) {
	var clss Feature
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type Variation struct {
	Id        string              `json:"_id"`
	Name      string              `json:"name"`
	Key       string              `json:"key"`
	Variables []VariationVariable `json:"variables"`
}

func (v *Variation) FromJSON(js []byte) (err error, rt Variation) {
	var clss Variation
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type VariationVariable struct {
	Var   string `json:"_var"`
	Value string `json:"value"`
}

func (v *VariationVariable) FromJSON(js []byte) (err error, rt VariationVariable) {
	var clss VariationVariable
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}
