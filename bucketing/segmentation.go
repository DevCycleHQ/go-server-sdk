package bucketing

import (
	"fmt"
	"github.com/devcyclehq/go-server-sdk/v2/util"
	"math"
	"regexp"
	"strings"

	"github.com/devcyclehq/go-server-sdk/v2/api"
)

func filterForAudienceMatch(filter *AudienceMatchFilter, configAudiences map[string]NoIdAudience, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	audiences := filter.Audiences
	comparator := filter.GetComparator()

	for _, audience := range audiences {
		a, ok := configAudiences[audience]
		if !ok {
			return false
		}
		if a.Filters.Evaluate(configAudiences, user, clientCustomData) {
			return comparator == "="
		}
	}
	return comparator == "!="
}

func filterFunctionsBySubtype(filter *UserFilter, user api.PopulatedUser, clientCustomData map[string]interface{}) bool {
	switch filter.SubType {
	case SubTypeCountry:
		return checkStringsFilter(user.Country, filter)
	case SubTypeEmail:
		return checkStringsFilter(user.Email, filter)
	case SubTypeUserID:
		return checkStringsFilter(user.UserId, filter)
	case SubTypeAppVersion:
		return checkVersionFilters(user.AppVersion, filter)
	case SubTypePlatformVersion:
		return checkVersionFilters(user.PlatformVersion, filter)
	case SubTypeDeviceModel:
		return checkStringsFilter(user.User.DeviceModel, filter)
	case SubTypePlatform:
		return checkStringsFilter(user.Platform, filter)
	default:
		return false
	}
}

func checkCustomData(filter *CustomDataFilter, data map[string]interface{}, clientCustomData map[string]interface{}) bool {
	operator := filter.GetComparator()
	var dataValue interface{}

	if _, ok := data[filter.DataKey]; !ok {
		if v2, ok2 := clientCustomData[filter.DataKey]; ok2 {
			dataValue = v2
		}
	} else {
		dataValue = data[filter.DataKey]
	}
	isNot64Bit := false
	switch dataValue.(type) {
	case uint8:
		isNot64Bit = true
	case uint16:
		isNot64Bit = true
	case uint32:
		isNot64Bit = true
	case uint:
		isNot64Bit = true
	case int8:
		isNot64Bit = true
	case int16:
		isNot64Bit = true
	case int32:
		isNot64Bit = true
	case int:
		isNot64Bit = true
	case float32:
		isNot64Bit = true
	}
	if isNot64Bit {
		util.Errorf("Custom data key %s is not a 64 bit type. Please use a 64 bit type", filter.DataKey)
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
	if operator == ComparatorEqual {
		return str != "" && stringArrayIn(values, str)
	} else if operator == ComparatorNotEqual {
		return str != "" && !stringArrayIn(values, str)
	} else if operator == ComparatorExist {
		return str != ""
	} else if operator == ComparatorNotExist {
		return str == ""
	} else if operator == ComparatorContain {
		return str != "" && stringArrayContains(values, str)
	} else if operator == ComparatorNotContain {
		return str == "" || !stringArrayContains(values, str)
	} else if operator == ComparatorStartWith {
		return str != "" && stringArrayStartsWith(values, str)
	} else if operator == ComparatorNotStartWith {
		return str == "" || !stringArrayStartsWith(values, str)
	} else if operator == ComparatorEndWith {
		return str != "" && stringArrayEndsWith(values, str)
	} else if operator == ComparatorNotEndWith {
		return str == "" || !stringArrayEndsWith(values, str)
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
		if substring == "" {
			continue
		}
		if strings.Contains(search, substring) {
			return true
		}
	}
	return false
}

func stringArrayStartsWith(prefixes []string, search string) bool {
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(search, prefix) {
			return true
		}
	}
	return false
}

func stringArrayEndsWith(suffixes []string, search string) bool {
	for _, suffix := range suffixes {
		if suffix == "" {
			continue
		}
		if strings.HasSuffix(search, suffix) {
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
	return checkVersionFilter(appVersion, values, operator)
}

func convertToSemanticVersion(version string) string {
	splitVersion := strings.Split(version, ".")
	if len(splitVersion) < 2 {
		splitVersion = append(splitVersion, "0")
	}
	if len(splitVersion) < 3 {
		splitVersion = append(splitVersion, "0")
	}

	for i, value := range splitVersion {
		if value == "" {
			splitVersion[i] = "0"
		}
	}
	return strings.Join(splitVersion, ".")
}

func checkVersionValue(filterVersion string, version string, operator string) bool {
	if version != "" && len(filterVersion) > 0 {
		options := OptionsType{
			Lexicographical: false,
			ZeroExtend:      true,
		}
		result := versionCompare(version, filterVersion, options)

		if math.IsNaN(result) {
			return false
		} else if result == 0 && strings.Contains(operator, "=") {
			return true
		} else if result == 1 && strings.Contains(operator, ">") {
			return true
		} else if result == -1 && strings.Contains(operator, "<") {
			return true
		}
	}

	return false
}

func checkVersionFilter(version string, filterVersions []string, operator string) bool {
	if version == "" {
		return false
	}

	var parsedVersion = version
	var parsedOperator = operator

	var not = false
	if parsedOperator == "!=" {
		parsedOperator = "="
		not = true
	}

	var parsedFilterVersions = filterVersions
	if parsedOperator != "=" {
		// remove any non-number and . characters, and remove everything after a hyphen
		// eg. 1.2.3a-b6 becomes 1.2.3
		regex1, err := regexp.Compile(`[^(\d|.|\-)]`)
		if err != nil {
			fmt.Println("Error compiling regex: ", err)
		}
		regex2, err := regexp.Compile(`-.*`)
		if err != nil {
			fmt.Println("Error compiling regex: ", err)
		}

		parsedVersion = regex2.ReplaceAllString(regex1.ReplaceAllString(parsedVersion, ""), "")

		var mappedFilterVersions []string
		for _, filterVersion := range parsedFilterVersions {
			mappedFilterVersions = append(mappedFilterVersions, regex2.ReplaceAllString(regex1.ReplaceAllString(filterVersion, ""), ""))
		}
		parsedFilterVersions = mappedFilterVersions
	}

	parsedVersion = convertToSemanticVersion(parsedVersion)

	passed := false
	// Replace Array.some(), because you can"t access captured data in a closure
	for _, filterVersion := range parsedFilterVersions {
		if checkVersionValue(filterVersion, parsedVersion, operator) {
			passed = true
			break
		}
	}

	if !not {
		return passed
	}
	return !passed
}

func _checkNumberFilter(num float64, filterNums []float64, operator string) bool {
	if operator != "" {
		if operator == "exist" {
			return !math.IsNaN(num)
		} else if operator == "!exist" {
			return math.IsNaN(num)
		}
	}

	if math.IsNaN(num) {
		return false
	}

	if operator == "!=" {
		passesFilter := true
		for _, filterNum := range filterNums {
			if math.IsNaN(filterNum) || num == filterNum {
				passesFilter = false
			}
		}
		return passesFilter
	}

	// replace filterNums.some() logic
	someValue := false
	for _, filterNum := range filterNums {
		if math.IsNaN(filterNum) {
			continue
		}

		if operator == "=" {
			someValue = num == filterNum
		} else if operator == ">" {
			someValue = num > filterNum
		} else if operator == ">=" {
			someValue = num >= filterNum
		} else if operator == "<" {
			someValue = num < filterNum
		} else if operator == "<=" {
			someValue = num <= filterNum
		} else {
			continue
		}

		if someValue {
			return true
		}
	}
	return someValue
}

/**
 * Returns true if the given value is not a type we define as "nonexistent" (NaN, empty string etc.)
 * Used only for values we don"t have a specific datatype for (eg. customData values)
 * If value has a datatype, use one of the type checkers above (eg. checkStringFilter)
 * NOTE: The use of Number.isNaN is required over the global isNaN as the check it performs is more specific
 */

func checkValueExists(value interface{}) bool {
	isString := false
	isFloat := false
	isInteger := false
	isBool := false

	switch value.(type) {
	case string:
		isString = true
	case int:
		isInteger = true
	case bool:
		isBool = true
	case float64:
		isFloat = true
	default:
	}

	return value != nil && (isString || isFloat || isInteger || isBool) &&
		(!isString || value.(string) != "") &&
		(!isFloat || !math.IsNaN(value.(float64)))
}
