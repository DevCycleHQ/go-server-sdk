package native_bucketing

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

func _evaluateOperator(operator BaseOperator, audiences map[string]NoIdAudience, user DVCPopulatedUser, clientCustomData map[string]interface{}) bool {
	if len(operator.Filters()) == 0 {
		return false
	}
	if operator.Operator() == "or" {
		for _, f := range operator.Filters() {
			if f.OperatorClass != nil {
				return _evaluateOperator(*f.OperatorClass, audiences, user, clientCustomData)
			} else if f.FilterClass != nil && doesUserPassFilter(*f.FilterClass, audiences, user, clientCustomData) {
				return true
			}
		}
		return false
	} else if operator.Operator() == "and" {
		for _, f := range operator.Filters() {
			if f.OperatorClass != nil {
				return _evaluateOperator(*f.OperatorClass, audiences, user, clientCustomData)
			} else if f.FilterClass != nil && !doesUserPassFilter(*f.FilterClass, audiences, user, clientCustomData) {
				return false
			}
		}
		return true
	}
	return false
}

func doesUserPassFilter(filter BaseFilter, audiences map[string]NoIdAudience, user DVCPopulatedUser, clientCustomData map[string]interface{}) bool {
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
		if err := userFilter.Validate(); err != nil {
			return filterFunctionsBySubtype(userFilter.SubType(), user, userFilter, clientCustomData)
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

func filterFunctionsBySubtype(subType string, user DVCPopulatedUser, filter BaseFilter, clientCustomData map[string]interface{}) bool {
	if subType == "country" {
		return checkStringsFilter(user.Country, filter.(UserFilter))
	} else if subType == "email" {
		return checkStringsFilter(user.Email, filter.(UserFilter))
	} else if subType == "user_id" {
		return checkStringsFilter(user.UserId, filter.(UserFilter))
	} else if subType == "appVersion" {
		return checkVersionFilters(user.AppVersion, filter.(UserFilter))
	} else if subType == "platformVersion" {
		return checkVersionFilters(user.PlatformVersion, filter.(UserFilter))
	} else if subType == "deviceModel" {
		return checkStringsFilter(user.DeviceModel, filter.(UserFilter))
	} else if subType == "platform" {
		return checkStringsFilter(user.Platform, filter.(UserFilter))
	} else if subType == "customData" {
		if err := filter.(CustomDataFilter).Validate(); err != nil {
			return false
		}
		return checkCustomData(user.CombinedCustomData(), clientCustomData, filter.(CustomDataFilter))
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
