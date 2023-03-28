package native_bucketing

import (
	"encoding/json"
	"fmt"
	"gopkg.in/validator.v2"
	"time"
)

type Target struct {
	Id           string               `json:"_id"`
	Audience     Audience             `json:"_audience"`
	Rollout      Rollout              `json:"rollout"`
	Distribution []TargetDistribution `json:"distribution"`
}

func (t *Target) FromJSON(js []byte) (err error, rt Target) {
	var clss Target
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

func (t *Target) DecideTargetVariation(boundedHash float64) (error, string) {
	var distributionIndex float64 = 0
	const previousDistributionIndex = 0
	for _, d := range t.Distribution {
		distributionIndex += d.Percentage
		if boundedHash >= previousDistributionIndex && boundedHash < distributionIndex {
			return nil, d.Variation
		}
	}
	return fmt.Errorf("failed to decide target variation: %s", t.Id), ""
}

type Audience struct {
	Id      string           `json:"_id"`
	Filters TopLevelOperator `json:"filters"`
}

func (a *Audience) FromJSON(js []byte) (err error, rt Audience) {
	var clss Audience
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type TopLevelOperator struct {
	Filters  []AudienceFilterOrOperator `json:"filters"`
	Operator string                     `json:"operator" validate:"regexp=^(and|or)$"`
}

func (t *TopLevelOperator) FromJSON(js []byte) (err error, rt TopLevelOperator) {
	var clss TopLevelOperator
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
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

func (a *AudienceFilterOrOperator) FromJSON(js []byte) (err error, rt AudienceFilterOrOperator) {
	var clss AudienceFilterOrOperator
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type UserFilter struct {
	AudienceFilterOrOperator
	Type       string `json:"type" validate:"regexp=^(all|user|optIn)$"`
	SubType    string `json:"subType" validate:"regexp=^(|user_id|email|ip|country|platform|platformVersion|appVersion|deviceModel|customData)$"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=|>|>=|<|<=|exist|!exist|contain|!contain)$"`
	Values     []interface{}
}

func (u *UserFilter) FromJSON(js []byte) (err error, rt UserFilter) {
	var clss UserFilter
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type CustomDataFilter struct {
	UserFilter
	DataKey     string `json:"dataKey"`
	DataKeyType string `json:"dataKeyType" validate:"regexp=^(String|Boolean|Number)$"`
}

func (c *CustomDataFilter) FromJSON(js []byte) (err error, rt CustomDataFilter) {
	var clss CustomDataFilter
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type Rollout struct {
	Type            string         `json:"type" validate:"regexp=^(schedule|gradual|stepped)$"`
	StartPercentage float64        `json:"startPercentage"`
	StartDate       time.Time      `json:"startDate"`
	Stages          []RolloutStage `json:"stages"`
}

func (r *Rollout) FromJSON(js []byte) (err error, rt Rollout) {
	var clss Rollout
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
}

type RolloutStage struct {
	Type       string    `json:"type"`
	Date       time.Time `json:"date"`
	Percentage float64   `json:"percentage" validate:"regexp=^(linear|discrete)$"`
}

func (r *RolloutStage) FromJSON(js []byte) (err error, rt RolloutStage) {
	var clss RolloutStage
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return err, clss
	}
	err = validator.Validate(clss)
	return
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

type AudienceOperator struct {
	Operator string                     `json:"operator" validate:"regexp=^(and|or)$"`
	Filters  []AudienceFilterOrOperator `json:"filters"`
}
