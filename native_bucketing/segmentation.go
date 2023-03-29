package native_bucketing

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

func _evaluateOperator(operator *AudienceOperator, audiences map[string]NoIdAudience, user DVCPopulatedUser, clientCustomData map[string]interface{}) bool {
	if len(operator.Filters) == 0 {
		return false
	}
	if operator.Operator == "or" {
		for _, f := range operator.Filters {
			if f.OperatorClass != nil {
				return _evaluateOperator(f.OperatorClass, audiences, user, clientCustomData)
			} else if f.FilterClass != nil && doesUserPassFilter(*f.FilterClass, audiences, user, clientCustomData) {
				return true
			}
		}
		return false
	} else if operator.Operator == "and" {
		for _, f := range operator.Filters {
			if f.OperatorClass != nil {
				return _evaluateOperator(f.OperatorClass, audiences, user, clientCustomData)
			} else if f.FilterClass != nil && !doesUserPassFilter(*f.FilterClass, audiences, user, clientCustomData) {
				return false
			}
		}
		return true
	}
	return false
}

func doesUserPassFilter(filter AudienceFilter, audiences map[string]NoIdAudience, user DVCPopulatedUser, clientCustomData map[string]interface{}) bool {
	isValid := true

	if filter.Type() == "all" {
		return true
	} else if filter.Type() == "optIn" {
		return false
	} else if filter.Type() == "audienceMatch" {
		if amF := filter.(AudienceMatchFilter); amF.IsValid {
			return filterForAudienceMatch(amF, audiences, user, clientCustomData)
		}
		isValid = false
	}

	if isValid {
		userFilter := filter.(UserFilter)
		if userFilter.IsValid {
			return filterFunctionsBySubtype(userFilter.SubType, user, userFilter, clientCustomData)
		}
	}

	return false

}

func filterForAudienceMatch(filter AudienceMatchFilter, configAudiences map[string]NoIdAudience, user DVCPopulatedUser, clientCustomData map[string]interface{}) bool {

	audiences := getFilterAudiencesAsStrings(filter)
	comparator := filter.Comparator

	for _, audience := range audiences {
		a, ok := configAudiences[audience]
		if !ok {
			return false
		}
		if _evaluateOperator(a.Filters, configAudiences, user, clientCustomData) {
			return comparator == "="
		}
	}
	return comparator == "!="
}

func getFilterAudiences(filter AudienceMatchFilter) []interface{} {
	audiences := filter.Audiences
	var acc []interface{}
	for _, audience := range audiences {
		if audience != nil {
			acc = append(acc, audience)
		}
	}
	return acc
}

func getFilterAudiencesAsStrings(filter AudienceMatchFilter) []string {
	jsonAud := getFilterAudiences(filter)
	var ret []string
	for _, aud := range jsonAud {
		if v, ok := aud.(string); ok {
			ret = append(ret, v)
		}
	}
	return ret
}

func filterFunctionsBySubtype(subType string, user DVCPopulatedUser, filter AudienceFilter, clientCustomData map[string]interface{}) bool {
	if subType == "country" {
		return _checkStringsFilter(user.Country, filter.(UserFilter))
	} else if subType == "email" {
		return _checkStringsFilter(user.Email, filter.(UserFilter))
	} else if subType == "user_id" {
		return _checkStringsFilter(user.UserId, filter.(UserFilter))
	} else if subType == "appVersion" {
		return _checkVersionFilters(user.AppVersion, filter.(UserFilter))
	} else if subType == "platformVersion" {
		return _checkVersionFilters(user.PlatformVersion, filter.(UserFilter))
	} else if subType == "deviceModel" {
		return _checkStringsFilter(user.DeviceModel, filter.(UserFilter))
	} else if subType == "platform" {
		return _checkStringsFilter(user.Platform, filter.(UserFilter))
	} else if subType == "customData" {
		if cdf := filter.(CustomDataFilter); !cdf.IsValid {
			return false
		}
		return _checkCustomData(user.CombinedCustomData(), clientCustomData, filter.(CustomDataFilter))
	} else {
		return false
	}
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
		regex1, err := regexp.Compile("[^(\\d|.|\\-)]/g")
		if err != nil {
			fmt.Println("Error compiling regex: ", err)
		}
		regex2, err := regexp.Compile("-.*/g")
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

	// replace filterNums.some() logic
	someValue := false
	for _, filterNum := range filterNums {
		if math.IsNaN(filterNum) {
			continue
		}

		if operator == "=" {
			someValue = num == filterNum
		} else if operator == "!=" {
			someValue = num != filterNum
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

func checkNumbersFilterJSONValue(jsonValue interface{}, filter UserFilter) bool {
	return _checkNumbersFilter(jsonValue.(float64), filter)
}

func _checkNumbersFilter(number float64, filter UserFilter) bool {
	operator := filter.Comparator
	values := getFilterValuesAsF64(filter)
	return _checkNumberFilter(number, values, operator)
}

func _checkStringsFilter(str string, filter UserFilter) bool {
	contains := func(arr []string, substr string) bool {
		for _, s := range arr {
			if strings.Contains(s, substr) {
				return true
			}
		}
		return false
	}
	operator := filter.Comparator
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

func _checkBooleanFilter(b bool, filter UserFilter) bool {
	contains := func(arr []bool, search bool) bool {
		for _, s := range arr {
			if s == search {
				return true
			}
		}
		return false
	}
	operator := filter.Comparator
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

func _checkVersionFilters(appVersion string, filter UserFilter) bool {
	operator := filter.Comparator
	values := getFilterValuesAsString(filter)
	// dont need to do semver if they"re looking for an exact match. Adds support for non semver versions.
	if operator == "=" {
		return _checkStringsFilter(appVersion, filter)
	} else {
		return checkVersionFilter(appVersion, values, operator)
	}
}

func _checkCustomData(data map[string]interface{}, clientCustomData map[string]interface{}, filter UserFilterInterface) bool {
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
			return _checkStringsFilter("", filter.(UserFilter))
		} else {

			return _checkStringsFilter(v, filter)
		}
	} else if v, ok := dataValue.(float64); ok && customDataFilter.DataKeyType == "Number" {
		return checkNumbersFilterJSONValue(dataValue, filter.(UserFilter)
	} else if v, ok := dataValue.(bool); ok && customDataFilter.DataKeyType == "Boolean" {
		return _checkBooleanFilter(v, filter.(UserFilter))
	} else if dataValue == nil && operator == "!=" {
		return true
	}
	return false
}

func getFilterValues(filter UserFilter) []interface{} {
	values := filter.Values
	var ret []interface{}
	for _, value := range values {
		if value != nil {
			ret = append(ret, value)
		}
	}
	return ret
}

func getFilterValuesAsString(filter UserFilter) []string {
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

func getFilterValuesAsF64(filter UserFilter) []float64 {
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

func getFilterValuesAsBoolean(filter UserFilter) []bool {
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
		break
	case int:
		isInteger = true
		break
	case bool:
		isBool = true
		break
	case float64:
		isFloat = true
		break
	default:
		break
	}

	return value != nil && !!(isString || isFloat || isInteger || isBool) &&
		(!isString || value.(string) != "") &&
		(!isFloat || !math.IsNaN(value.(float64))) &&
		(!isInteger || !math.IsNaN(float64(value.(int))))
}
