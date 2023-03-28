package native_bucketing

import (
	"encoding/json"
	"gopkg.in/validator.v2"
)

type FeatureConfiguration struct {
	Id               string                 `json:"_id"`
	Prerequisites    []FeaturePrerequisites `json:"prerequisites"`
	WinningVariation FeatureVariation       `json:"winningVariation"`
	ForcedUsers      map[string]string      `json:"forcedUsers"`
	Targets          []Target               `json:"targets"`
}

func (f *FeatureConfiguration) FromJSON(js []byte) (err error, rt FeatureConfiguration) {
	var clss FeatureConfiguration
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type FeaturePrerequisites struct {
	Feature    string `json:"_feature"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=)$"`
}

func (f *FeaturePrerequisites) FromJSON(js []byte) (err error, rt FeaturePrerequisites) {
	var clss FeaturePrerequisites
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type FeatureVariation struct {
	Variation string `json:"_variation"`
	Feature   string `json:"_feature"`
}

func (f *FeatureVariation) FromJSON(js []byte) (err error, rt FeatureVariation) {
	var clss FeatureVariation
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}
