package native_bucketing

import (
	"encoding/json"
	"gopkg.in/validator.v2"
	"time"
)

type FeatureConfiguration struct {
	Id               string                  `json:"_id"`
	Prerequisites    []FeaturePrerequisites  `json:"prerequisites"`
	WinningVariation FeatureWinningVariation `json:"winningVariation"`
	ForcedUsers      map[string]string       `json:"forcedUsers"`
	Targets          []Target                `json:"targets"`
}

func (f *FeatureConfiguration) FromJSON(js string) (err error, rt FeatureConfiguration) {
	var clss FeatureConfiguration
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type FeaturePrerequisites struct {
	Feature    string `json:"_feature"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=)$"`
}

func (f *FeaturePrerequisites) FromJSON(js string) (err error, rt FeaturePrerequisites) {
	var clss FeaturePrerequisites
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type FeatureWinningVariation struct {
	Variation string    `json:"_variation"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (f *FeatureWinningVariation) FromJSON(js string) (err error, rt FeatureWinningVariation) {
	var clss FeatureWinningVariation
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}
