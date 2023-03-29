package native_bucketing

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gopkg.in/validator.v2"
)

type Target struct {
	Id           string               `json:"_id"`
	Audience     *Audience            `json:"_audience"`
	Rollout      *Rollout             `json:"rollout"`
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

func (t *Target) DecideTargetVariation(boundedHash float64) (string, error) {
	var distributionIndex float64 = 0
	const previousDistributionIndex = 0
	for _, d := range t.Distribution {
		distributionIndex += d.Percentage
		if boundedHash >= previousDistributionIndex && boundedHash < distributionIndex {
			return d.Variation, nil
		}
	}
	return "", fmt.Errorf("failed to decide target variation: %s", t.Id)
}

type NoIdAudience struct {
	Filters *AudienceOperator `json:"filters"`
}

func NewNoIdAudience(audience map[string]interface{}) (NoIdAudience, error) {
	filtersObj, ok := audience["filters"]
	if !ok {
		return NoIdAudience{}, fmt.Errorf("object not found for key: filters")
	}

	filters, ok := filtersObj.(map[string]interface{})
	if !ok {
		return NoIdAudience{}, fmt.Errorf("expected object for key: filters, found: %T", filtersObj)
	}

	audienceOperator, err := NewAudienceOperator(filters)
	if err != nil {
		return NoIdAudience{}, err
	}

	return NoIdAudience{
		Filters: audienceOperator,
	}, nil
}

type Audience struct {
	NoIdAudience
	Id string `json:"_id"`
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

func (r *RolloutStage) FromJSON(js []byte) (clss RolloutStage, err error) {
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return
	}
	err = validator.Validate(clss)
	return
}

type TargetDistribution struct {
	Variation  string  `json:"_variation"`
	Percentage float64 `json:"percentage"`
}

func (t *TargetDistribution) FromJSON(js []byte) (clss TargetDistribution, err error) {
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return clss, err
	}
	err = validator.Validate(clss)
	return
}

func NewAudienceOperator(filter map[string]interface{}) (*AudienceOperator, error) {
	operatorObj, ok := filter["operator"]
	if !ok {
		return nil, fmt.Errorf("object not found for key: filters")
	}

	operator, ok := operatorObj.(string)
	if !ok {
		return nil, fmt.Errorf("expected string for key: operator, found: %T", operatorObj)
	}

	if operator != "and" && operator != "or" {
		// TODO: use centralized logging
		log.Printf(`[DevCycle] Warning: String value: %s, for key: %s does not match a valid string.`, operator, "operator")
	}

	filtersObj, ok := filter["filters"]
	if !ok {
		return nil, fmt.Errorf("object not found for key: filters")
	}

	filters, ok := filtersObj.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected string for key: filters, found: %T", filtersObj)
	}

	audienceFilters := []FilterOrOperator{}
	for _, filter := range filters {
		fmt.Println("filter: ", filter)
		audienceFilter := FilterOrOperator{}
		var err error
		if err != nil {
			return nil, err
		}
		audienceFilters = append(audienceFilters, audienceFilter)
	}
	audOp := AudienceOperator{
		operator: operator,
		filters:  audienceFilters,
	}

	return &audOp, nil
}
