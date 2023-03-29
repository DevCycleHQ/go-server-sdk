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
	OperatorClass *AudienceOperator
	FilterClass   *AudienceFilter
}

type AudienceFilter interface {
	Type() string
}

type UserFilterInterface interface {
	AudienceFilter
	SubType() string
	Comparator() string
	Values() []interface{}
	IsValid() bool
	CompiledStringVals() []string
	CompiledBoolVals() []bool
	CompiledNumVals() []float64
}

type UserFilter struct {
	FType              string `json:"type" validate:"regexp=^(all|user|optIn)$"`
	SubType            string `json:"subType" validate:"regexp=^(|user_id|email|ip|country|platform|platformVersion|appVersion|deviceModel|customData)$"`
	Comparator         string `json:"comparator" validate:"regexp=^(=|!=|>|>=|<|<=|exist|!exist|contain|!contain)$"`
	Values             []interface{}
	IsValid            bool
	CompiledStringVals []string
	CompiledBoolVals   []bool
	CompiledNumVals    []float64
}

func (u UserFilter) Type() string {
	return u.FType
}

func NewUserFilter(json []byte) (error, *UserFilter) {
	u := UserFilter{}
	uf, err := u.FromJSON(json)
	if err != nil {
		return err, nil
	}
	uf.CompileValues()

	return nil, &uf
}

func (u *UserFilter) CompileValues() {
	if len(u.Values) == 0 {
		return
	}

	firstValue := u.Values[0]

	if _, bok := firstValue.(bool); bok {
		boolValues := make([]bool, 0)

		for _, value := range u.Values {
			if val, ok := value.(bool); ok {
				boolValues = append(boolValues, val)
			} else {
				fmt.Printf("[DevCycle] Warning: Filter values must be all of the same type. Expected: bool, got: %v\n", value)
			}
		}
		u.CompiledBoolVals = boolValues
	} else if _, sok := firstValue.(string); sok {
		stringValues := make([]string, 0)

		for _, value := range u.Values {
			if val, ok := value.(string); ok {
				stringValues = append(stringValues, val)
			} else {
				fmt.Printf("[DevCycle] Warning: Filter values must be all of the same type. Expected: string, got: %v\n", value)
			}
		}
		u.CompiledStringVals = stringValues
	} else if _, fok := firstValue.(float64); fok {
		numValues := make([]float64, 0)

		for _, value := range u.Values {
			if val, ok := value.(float64); ok {
				numValues = append(numValues, val)
			} else {
				fmt.Printf("[DevCycle] Warning: Filter values must be all of the same type. Expected: number, got: %v\n", value)
			}
		}
		u.CompiledNumVals = numValues
	} else {
		fmt.Printf("Filter values of unknown type. %v\n", firstValue)
	}
}

func (u *UserFilter) GetStringValues() []string {
	if u.CompiledStringVals != nil {
		return u.CompiledStringVals
	} else {
		return []string{}
	}
}

func (u *UserFilter) GetBooleanValues() []bool {
	if u.CompiledBoolVals != nil {
		return u.CompiledBoolVals
	} else {
		return []bool{}
	}
}

func (u *UserFilter) GetNumberValues() []float64 {
	if u.CompiledNumVals != nil {
		return u.CompiledNumVals
	} else {
		return []float64{}
	}
}
func (u *UserFilter) FromJSON(js []byte) (clss UserFilter, err error) {
	err = json.Unmarshal(js, &clss)
	if err != nil {
		return
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

type AudienceOperator struct {
	Operator string                     `json:"operator" validate:"regexp=^(and|or)$"`
	Filters  []AudienceFilterOrOperator `json:"filters"`
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

	audienceFilters := []AudienceFilterOrOperator{}
	for _, filter := range filters {
		audienceFilter, err := NewAudienceFilterOrOperator(filter)
		if err != nil {
			return nil, err
		}
		audienceFilters = append(audienceFilters, *audienceFilter)
	}

	return &AudienceOperator{
		Operator: operator,
		Filters:  audienceFilters,
	}, nil
}

type AudienceMatchFilter struct {
	AudienceFilter
	Audiences  []interface{} `json:"_audiences"`
	Comparator string        `json:"comparator" validate:"regexp=^(=|!=)$"`
	IsValid    bool
}
