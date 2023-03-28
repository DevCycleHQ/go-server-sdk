package native_bucketing

import (
	"encoding/json"
	"gopkg.in/validator.v2"
	"time"
)

type Target struct {
	Id           string               `json:"_id"`
	Audience     Audience             `json:"_audience"`
	Rollout      Rollout              `json:"rollout"`
	Distribution []TargetDistribution `json:"distribution"`
}

func (t *Target) FromJSON(js string) (err error, rt Target) {
	var clss Target
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type Audience struct {
	Id      string           `json:"_id"`
	Filters TopLevelOperator `json:"filters"`
}

func (a *Audience) FromJSON(js string) (err error, rt Audience) {
	var clss Audience
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type TopLevelOperator struct {
	Filters  []AudienceFilterOrOperator `json:"filters"`
	Operator string                     `json:"operator" validate:"regexp=^(and|or)$"`
}

func (t *TopLevelOperator) FromJSON(js string) (err error, rt TopLevelOperator) {
	var clss TopLevelOperator
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type AudienceFilterOrOperator struct {
	Type        string                     `json:"type" validate:"regexp=^(all|user|optIn)$"`
	SubType     string                     `json:"subType" validate:"regexp=^(|user_id|email|ip|country|platform|platformVersion|appVersion|deviceModel|customData)$"`
	Comparator  string                     `json:"comparator" validate:"regexp=^(=|!=|>|>=|<|<=|exist|!exist|contain|!contain)$"`
	DataKey     string                     `json:"dataKey"`
	DataKeyType string                     `json:"dataKeyType" validate:"regexp=^(String|Boolean|Number)$"`
	Values      []interface{}              `json:"values"`
	Operator    string                     `json:"operator" validate:"regexp=^(and|or)$"`
	Filters     []AudienceFilterOrOperator `json:"filters"`
}

func (a *AudienceFilterOrOperator) FromJSON(js string) (err error, rt AudienceFilterOrOperator) {
	var clss AudienceFilterOrOperator
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type UserFilter struct {
	AudienceFilterOrOperator
	Type       string `json:"type" validate:"regexp=^(all|user|optIn)$"`
	SubType    string `json:"subType" validate:"regexp=^(|user_id|email|ip|country|platform|platformVersion|appVersion|deviceModel|customData)$"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=|>|>=|<|<=|exist|!exist|contain|!contain)$"`
	Values     []interface{}
}

func (u *UserFilter) FromJSON(js string) (err error, rt UserFilter) {
	var clss UserFilter
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type CustomDataFilter struct {
	UserFilter
	DataKey     string `json:"dataKey"`
	DataKeyType string `json:"dataKeyType" validate:"regexp=^(String|Boolean|Number)$"`
}

func (c *CustomDataFilter) FromJSON(js string) (err error, rt CustomDataFilter) {
	var clss CustomDataFilter
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type Rollout struct {
	Type            string         `json:"type" validate:"regexp=^(schedule|gradual|stepped)$"`
	StartPercentage float64        `json:"startPercentage"`
	StartDate       time.Time      `json:"startDate"`
	Stages          []RolloutStage `json:"stages"`
}

func (r *Rollout) FromJSON(js string) (err error, rt Rollout) {
	var clss Rollout
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type RolloutStage struct {
	Type       string    `json:"type"`
	Date       time.Time `json:"date"`
	Percentage float64   `json:"percentage" validate:"regexp=^(linear|discrete)$"`
}

func (r *RolloutStage) FromJSON(js string) (err error, rt RolloutStage) {
	var clss RolloutStage
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}

type TargetDistribution struct {
	Variation  string  `json:"_variation"`
	Percentage float64 `json:"percentage"`
}

func (t *TargetDistribution) FromJSON(js string) (err error, rt TargetDistribution) {
	var clss TargetDistribution
	json.Unmarshal([]byte(js), &clss)
	if errs := validator.Validate(clss); errs != nil {
		return errs, clss
	}
	return nil, clss
}
