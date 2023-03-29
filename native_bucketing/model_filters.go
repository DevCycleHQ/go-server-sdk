package native_bucketing

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"strings"
)

type BaseFilter interface {
	Type() string
	Validate() error
	Comparator() string
	SubType() string
	Values() []interface{}
}

type filter struct {
	ftype      string `json:"type" validate:"regexp=^(all|user|optIn)$"`
	subType    string `json:"subType" validate:"regexp=^(|user_id|email|ip|country|platform|platformVersion|appVersion|deviceModel|customData)$"`
	comparator string `json:"comparator" validate:"regexp=^(=|!=|>|>=|<|<=|exist|!exist|contain|!contain)$"`
	values     []interface{}
}

func (f filter) Type() string {
	return f.ftype
}

func (f filter) Comparator() string {
	return f.comparator
}

func (f filter) SubType() string {
	return f.subType
}

func (f filter) Values() []interface{} {
	return f.values
}

func checkCustomData(data map[string]interface{}, clientCustomData map[string]interface{}, filter BaseFilter) bool {
	operator := filter.Comparator()
	var dataValue interface{}
	customDataFilter := filter.(CustomDataFilter)

	if _, ok := data[customDataFilter.DataKey]; !ok {
		if v2, ok2 := clientCustomData[customDataFilter.DataKey]; ok2 {
			dataValue = v2
		}
	} else {
		dataValue = data[customDataFilter.DataKey]
	}

	if operator == "exist" {
		return checkValueExists(dataValue)
	} else if operator == "!exist" {
		return !checkValueExists(dataValue)
	} else if v, ok := dataValue.(string); ok && customDataFilter.DataKeyType == "String" {
		if dataValue == nil {
			// TODO Need to redo the interface inheritance to make this work
			return checkStringsFilter("", filter)
		} else {
			return checkStringsFilter(v, filter)
		}
	} else if _, ok := dataValue.(float64); ok && customDataFilter.DataKeyType == "Number" {
		return checkNumbersFilterJSONValue(dataValue, filter.(UserFilter))
	} else if v, ok := dataValue.(bool); ok && customDataFilter.DataKeyType == "Boolean" {
		return _checkBooleanFilter(v, filter.(UserFilter))
	} else if dataValue == nil && operator == "!=" {
		return true
	}
	return false
}

func checkNumbersFilterJSONValue(jsonValue interface{}, filter BaseFilter) bool {
	return _checkNumbersFilter(jsonValue.(float64), filter)
}

func _checkNumbersFilter(number float64, filter BaseFilter) bool {
	operator := filter.Comparator()
	values := getFilterValuesAsF64(filter)
	return _checkNumberFilter(number, values, operator)
}

func checkStringsFilter(str string, filter BaseFilter) bool {
	contains := func(arr []string, substr string) bool {
		for _, s := range arr {
			if strings.Contains(s, substr) {
				return true
			}
		}
		return false
	}
	operator := filter.Comparator()
	values := getFilterValuesAsString(filter)
	if operator == "=" {
		return str != "" && contains(values, str)
	} else if operator == "!=" {
		return str != "" && !contains(values, str)
	} else if operator == "exist" {
		return str != ""
	} else if operator == "!exist" {
		return str == ""
	} else if operator == "contain" {
		return str != "" && !!contains(values, str)
	} else if operator == "!contain" {
		return str == "" || !contains(values, str)
	} else {
		return true
	}
}

func _checkBooleanFilter(b bool, filter BaseFilter) bool {
	contains := func(arr []bool, search bool) bool {
		for _, s := range arr {
			if s == search {
				return true
			}
		}
		return false
	}
	operator := filter.Comparator()
	values := getFilterValuesAsBoolean(filter)

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

func checkVersionFilters(appVersion string, filter BaseFilter) bool {
	operator := filter.Comparator()
	values := getFilterValuesAsString(filter)
	// dont need to do semver if they"re looking for an exact match. Adds support for non semver versions.
	if operator == "=" {
		return checkStringsFilter(appVersion, filter)
	} else {
		return checkVersionFilter(appVersion, values, operator)
	}
}

func getFilterValues(filter BaseFilter) []interface{} {
	values := filter.Values()
	var ret []interface{}
	for _, value := range values {
		if value != nil {
			ret = append(ret, value)
		}
	}
	return ret
}

func getFilterValuesAsString(filter BaseFilter) []string {
	var ret []string
	jsonValues := getFilterValues(filter)
	for _, jsonValue := range jsonValues {
		switch v := jsonValue.(type) {
		case string:
			ret = append(ret, v)
		default:
			continue
		}
	}
	return ret
}

func getFilterValuesAsF64(filter BaseFilter) []float64 {
	var ret []float64
	jsonValues := getFilterValues(filter)
	for _, jsonValue := range jsonValues {
		switch v := jsonValue.(type) {
		case int:
		case float64:
			ret = append(ret, v)
		default:
			continue
		}
	}
	return ret
}

func getFilterValuesAsBoolean(filter BaseFilter) []bool {
	var ret []bool
	jsonValues := getFilterValues(filter)
	for _, jsonValue := range jsonValues {
		switch v := jsonValue.(type) {
		case bool:
			ret = append(ret, v)
		default:
			continue
		}
	}
	return ret
}

type FilterOrOperator struct {
	OperatorClass *BaseOperator
	FilterClass   *BaseFilter
}

type UserFilter struct {
	filter
	CompiledStringVals []string
	CompiledBoolVals   []bool
	CompiledNumVals    []float64
}

func (u UserFilter) Validate() error {
	err := validator.New().Struct(u)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserFilter) CompileValues() {
	if len(u.Values()) == 0 {
		return
	}
	firstValue := u.Values()[0]

	switch firstValue.(type) {
	case bool:
		var boolValues []bool
		for _, value := range u.Values() {
			if val, ok := value.(bool); ok {
				boolValues = append(boolValues, val)
			} else {
				fmt.Printf("[DevCycle] Warning: Filter values must be all of the same type. Expected: bool, got: %v\n", value)
			}
		}
		u.CompiledBoolVals = boolValues
		break
	case string:
		var stringValues []string
		for _, value := range u.Values() {
			if val, ok := value.(string); ok {
				stringValues = append(stringValues, val)
			} else {
				fmt.Printf("[DevCycle] Warning: Filter values must be all of the same type. Expected: string, got: %v\n", value)
			}
		}
		u.CompiledStringVals = stringValues
		break
	case float64:
		var numValues []float64
		for _, value := range u.Values() {
			if val, ok := value.(float64); ok {
				numValues = append(numValues, val)
			} else {
				fmt.Printf("[DevCycle] Warning: Filter values must be all of the same type. Expected: number, got: %v\n", value)
			}
		}
		u.CompiledNumVals = numValues
		break
	default:
		fmt.Printf("[DevCycle] Warning: Filter values must be of type bool, string, or float64. Got: %v\n", firstValue)
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

type CustomDataFilter struct {
	UserFilter
	DataKey     string `json:"dataKey"`
	DataKeyType string `json:"dataKeyType" validate:"regexp=^(String|Boolean|Number)$"`
}

func (c CustomDataFilter) Validate() error {
	err := validator.New().Struct(c)
	if err != nil {
		return err
	}
	return nil
}

type AudienceMatchFilter struct {
	filter
	Audiences  []interface{} `json:"_audiences"`
	Comparator string        `json:"comparator" validate:"regexp=^(=|!=)$"`
	IsValid    bool
}

func (a AudienceMatchFilter) Validate() error {
	err := validator.New().Struct(a)
	if err != nil {
		return err
	}
	return nil
}
