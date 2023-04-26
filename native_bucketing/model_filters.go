package native_bucketing

import (
	"encoding/json"
	"fmt"
	"strings"
)

// All filter types must implement this
type BaseFilter interface {
	GetType() string
	GetComparator() string
	GetSubType() string
	GetOperator() (*AudienceOperator, bool)
	Validate() error
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

func (f filter) GetOperator() (*AudienceOperator, bool) {
	return nil, false
}

func (f filter) Validate() error {
	return nil
}

type MixedFilters []BaseFilter

func (m *MixedFilters) UnmarshalJSON(data []byte) error {
	// Parse into a list of RawMessages
	var rawItems []json.RawMessage
	err := json.Unmarshal(data, &rawItems)
	if err != nil {
		return err
	}

	filters := make([]BaseFilter, len(rawItems))

	for index, rawItem := range rawItems {
		// Parse each filter again to get the type
		var partial filter
		err = json.Unmarshal(rawItem, &partial)
		if err != nil {
			return err
		}

		var filter BaseFilter

		if partial.Operator != "" {
			var operator AudienceOperator
			err = json.Unmarshal(rawItem, &operator)
			if err != nil {
				return fmt.Errorf("Error unmarshalling filter: %w", err)
			}
			filters[index] = OperatorFilter{Operator: &operator}
			continue
		}

		switch partial.Type {
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
			filter = &AudienceFilter{}
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

		if err := filter.Validate(); err != nil {
			return fmt.Errorf("Error validating filter: %w", err)
		}

		filters[index] = filter
	}

	*m = filters

	return nil
}

type OperatorFilter struct {
	Operator *AudienceOperator
}

func (f OperatorFilter) GetType() string {
	return "operator"
}

func (f OperatorFilter) GetSubType() string {
	return ""
}

func (f OperatorFilter) GetComparator() string {
	return ""
}

func (f OperatorFilter) GetOperator() (*AudienceOperator, bool) {
	return f.Operator, true
}

func (f OperatorFilter) Validate() error {
	return nil
}

type AudienceFilter struct {
	filter
}

type UserFilter struct {
	filter
	Values []interface{} `json:"values"`

	CompiledStringVals []string
	CompiledBoolVals   []bool
	CompiledNumVals    []float64
}

func (f UserFilter) Type() string {
	return TypeUser
}

func (f UserFilter) Validate() error {
	return nil
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

func (f CustomDataFilter) Type() string {
	return TypeUser
}

func (f CustomDataFilter) SubType() string {
	return SubTypeCustomData
}

func (f CustomDataFilter) Validate() error {
	return nil
}

type AudienceMatchFilter struct {
	filter
	Audiences []string `json:"_audiences"`
}

func (f AudienceMatchFilter) Type() string {
	return TypeAudienceMatch
}

func (f AudienceMatchFilter) Validate() error {
	return nil
}

func checkCustomData(data map[string]interface{}, clientCustomData map[string]interface{}, filter *CustomDataFilter) bool {
	operator := filter.GetComparator()
	var dataValue interface{}

	if _, ok := data[filter.DataKey]; !ok {
		if v2, ok2 := clientCustomData[filter.DataKey]; ok2 {
			dataValue = v2
		}
	} else {
		dataValue = data[filter.DataKey]
	}

	if operator == "exist" {
		return checkValueExists(dataValue)
	} else if operator == "!exist" {
		return !checkValueExists(dataValue)
	} else if v, ok := dataValue.(string); ok && filter.DataKeyType == "String" {
		if dataValue == nil {
			return checkStringsFilter("", filter.UserFilter)
		} else {
			return checkStringsFilter(v, filter.UserFilter)
		}
	} else if _, ok := dataValue.(float64); ok && filter.DataKeyType == "Number" {
		return checkNumbersFilterJSONValue(dataValue, filter.UserFilter)
	} else if v, ok := dataValue.(bool); ok && filter.DataKeyType == "Boolean" {
		return _checkBooleanFilter(v, filter.UserFilter)
	} else if dataValue == nil && operator == "!=" {
		return true
	}
	return false
}

func checkNumbersFilterJSONValue(jsonValue interface{}, filter *UserFilter) bool {
	return _checkNumbersFilter(jsonValue.(float64), filter)
}

func _checkNumbersFilter(number float64, filter *UserFilter) bool {
	operator := filter.GetComparator()
	values := filter.CompiledNumVals
	return _checkNumberFilter(number, values, operator)
}

func checkStringsFilter(str string, filter *UserFilter) bool {
	operator := filter.GetComparator()
	values := filter.CompiledStringVals
	if operator == "=" {
		return str != "" && stringArrayIn(values, str)
	} else if operator == "!=" {
		return str != "" && !stringArrayIn(values, str)
	} else if operator == "exist" {
		return str != ""
	} else if operator == "!exist" {
		return str == ""
	} else if operator == "contain" {
		return str != "" && stringArrayContains(values, str)
	} else if operator == "!contain" {
		return str == "" || !stringArrayContains(values, str)
	} else {
		return false
	}
}

func stringArrayIn(arr []string, search string) bool {
	for _, s := range arr {
		if s == search {
			return true
		}
	}
	return false
}

func stringArrayContains(substrings []string, search string) bool {
	for _, substring := range substrings {
		if strings.Contains(search, substring) {
			return true
		}
	}
	return false
}

func _checkBooleanFilter(b bool, filter *UserFilter) bool {
	contains := func(arr []bool, search bool) bool {
		for _, s := range arr {
			if s == search {
				return true
			}
		}
		return false
	}
	operator := filter.GetComparator()
	values := filter.CompiledBoolVals

	if operator == "contain" || operator == "=" {
		return contains(values, b)
	} else if operator == "!contain" || operator == "!=" {
		return !contains(values, b)
	} else if operator == "exist" {
		return true
	} else if operator == "!exist" {
		return false
	} else {
		return false
	}
}

func checkVersionFilters(appVersion string, filter *UserFilter) bool {
	operator := filter.GetComparator()
	values := filter.CompiledStringVals
	// don't need to do semver if they're looking for an exact match. Adds support for non semver versions.
	if operator == "=" {
		return checkStringsFilter(appVersion, filter)
	} else {
		return checkVersionFilter(appVersion, values, operator)
	}
}
