// TODO: Add docstrings to all types in this file
package bucketing

import (
	"encoding/json"
	"fmt"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

type FilterOrOperator interface {
	Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool
}

// For compiling values inside a filter after parsing, or other optimizations
type InitializedFilter interface {
	Initialize() error
}

// Represents a partially parsed filter object from the JSON, before parsing a specific filter type
type filter struct {
	Type       string `json:"type" validate:"regexp=^(all|user|optIn)$"`
	SubType    string `json:"subType" validate:"regexp=^(|user_id|email|ip|country|platform|platformVersion|appVersion|deviceModel|customData)$"`
	Comparator string `json:"comparator" validate:"regexp=^(=|!=|>|>=|<|<=|exist|!exist|contain|!contain)$"`
	Operator   string `json:"operator" validate:"regexp=^(and|or)$"`
}

func (f filter) GetType() string {
	return f.Type
}

func (f filter) GetSubType() string {
	return f.SubType
}

func (f filter) GetComparator() string {
	return f.Comparator
}

type MixedFilters []FilterOrOperator

func (m *MixedFilters) UnmarshalJSON(data []byte) error {
	// Parse into a list of RawMessages
	var rawItems []json.RawMessage
	err := json.Unmarshal(data, &rawItems)
	if err != nil {
		return err
	}

	filters := make([]FilterOrOperator, len(rawItems))

	for index, rawItem := range rawItems {
		// Parse each filter again to get the type
		var partial filter
		err = json.Unmarshal(rawItem, &partial)
		if err != nil {
			return err
		}

		var filter FilterOrOperator

		if partial.Operator != "" {
			var operator *AudienceOperator
			err = json.Unmarshal(rawItem, &operator)
			if err != nil {
				return fmt.Errorf("Error unmarshalling filter: %w", err)
			}
			filters[index] = operator
			continue
		}

		switch partial.Type {
		case TypeAll:
			filter = &AllFilter{}
		case TypeOptIn:
			filter = &OptInFilter{}
		case TypeUser:
			switch partial.SubType {
			case SubTypeCustomData:
				filter = &CustomDataFilter{}
			default:
				filter = &UserFilter{}
			}
		case TypeAudienceMatch:
			filter = &AudienceMatchFilter{}
		default:
			util.Warnf(`Warning: Invalid filter type %s. To leverage this new filter definition, please update to the latest version of the DevCycle SDK.`, partial.Type)
			continue
		}

		err = json.Unmarshal(rawItem, &filter)
		if err != nil {
			return fmt.Errorf("Error unmarshalling filter: %w", err)
		}

		if filter, ok := filter.(InitializedFilter); ok {
			if err := filter.Initialize(); err != nil {
				return fmt.Errorf("Error initializing filter: %w", err)
			}
		}

		filters[index] = filter
	}

	*m = filters

	return nil
}

type PassFilter struct{}

func (filter PassFilter) Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	return true
}

type NoPassFilter struct{}

func (filter NoPassFilter) Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	return false
}

type AllFilter struct {
	PassFilter
}

type OptInFilter struct {
	NoPassFilter
}

type UserFilter struct {
	filter
	Values []interface{} `json:"values"`

	CompiledStringVals []string
	CompiledBoolVals   []bool
	CompiledNumVals    []float64
}

func (filter *UserFilter) Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	return filterFunctionsBySubtype(filter, user, clientCustomData)
}

func (f UserFilter) Type() string {
	return TypeUser
}

func (f *UserFilter) Initialize() error {
	return f.compileValues()
}

func (u *UserFilter) compileValues() error {
	if len(u.Values) == 0 {
		return nil
	}
	firstValue := u.Values[0]

	switch firstValue.(type) {
	case bool:
		var boolValues []bool
		for _, value := range u.Values {
			if val, ok := value.(bool); ok {
				boolValues = append(boolValues, val)
			} else {
				return fmt.Errorf("Filter values must be all of the same type. Expected: bool, got: %T %#v\n", value, value)
			}
		}
		u.CompiledBoolVals = boolValues
	case string:
		var stringValues []string
		for _, value := range u.Values {
			if val, ok := value.(string); ok {
				stringValues = append(stringValues, val)
			} else {
				return fmt.Errorf("Filter values must be all of the same type. Expected: string, got: %T %#v\n", value, value)
			}
		}
		u.CompiledStringVals = stringValues
	case float64:
		var numValues []float64
		for _, value := range u.Values {
			if val, ok := value.(float64); ok {
				numValues = append(numValues, val)
			} else {
				return fmt.Errorf("Filter values must be all of the same type. Expected: number, got: %T %#v\n", value, value)
			}
		}
		u.CompiledNumVals = numValues
	default:
		return fmt.Errorf("Filter values must be of type bool, string, or float64. Got: %T %#v\n", firstValue, firstValue)
	}

	return nil
}

type CustomDataFilter struct {
	*UserFilter
	DataKey     string `json:"dataKey"`
	DataKeyType string `json:"dataKeyType" validate:"regexp=^(String|Boolean|Number)$"`
}

func (filter *CustomDataFilter) Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	return checkCustomData(filter, user.CombinedCustomData(), clientCustomData)
}

func (f CustomDataFilter) Type() string {
	return TypeUser
}

func (f CustomDataFilter) SubType() string {
	return SubTypeCustomData
}

type AudienceMatchFilter struct {
	filter
	Audiences []string `json:"_audiences"`
}

func (filter *AudienceMatchFilter) Evaluate(audiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	return filterForAudienceMatch(filter, audiences, user, clientCustomData)
}

func (f AudienceMatchFilter) Type() string {
	return TypeAudienceMatch
}
