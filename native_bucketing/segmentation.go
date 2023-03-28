package native_bucketing

import (
	"math"
	"regexp"
	"strings"
)

func _evaluateOperator(operator TopLevelOperator, user DVCPopulatedUser) bool {
	if len(operator.Filters) == 0 {
		return false
	}
	if operator.Operator == "or" {
		for _, f := range operator.Filters {
			if doesUserPassFilter(f, user) {
				return true
			}
		}
		return false
	} else {
		for _, f := range operator.Filters {
			if !doesUserPassFilter(f, user) {
				return false
			}
		}
		return true
	}
}

func doesUserPassFilter(filter AudienceFilterOrOperator, user DVCPopulatedUser) bool {
	if filter.Operator == "all" {
		return true
	}
	if filter.Operator == "optIn" {
		return false
	}
	// Check userfilter
	if filter.Type == "user" {
		// throw err
	}
	userFilter := cast
	as
	userfilter
	if !validSubTypes.includes(userFilter.SubType) {
		// invalid subtype - should be caught by validator
		return false
	}
	return filterFunctionsBySubtype(userFilter.SubType, user, userFilter)
}

func filterFunctionsBySubtype(subType string, user DVCPopulatedUser, filter UserFilter) bool {
	if subType == "country" {
		return _checkStringsFilter(user.country, filter)
	} else if subType == "email" {
		return _checkStringsFilter(user.email, filter)
	} else if subType == "user_id" {
		return _checkStringsFilter(user.user_id, filter)
	} else if subType == "appVersion" {
		return _checkVersionFilters(user.appVersion, filter)
	} else if subType == "platformVersion" {
		return _checkVersionFilters(user.platformVersion, filter)
	} else if subType == "deviceModel" {
		return _checkStringsFilter(user.deviceModel, filter)
	} else if subType == "platform" {
		return _checkStringsFilter(user.platform, filter)
	} else if subType == "customData" {
		if !(filter instanceof
		CustomDataFilter)) {
		throw new Error("Invalid filter data")
		}
		return _checkCustomData(user.getCombinedCustomData(), filter
		as
		CustomDataFilter)
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

func checkVersionValue(
	filterVersion string,
	version string,
	operator string,
) bool {
	if version != "" && len(filterVersion) > 0 {
		options := OptionsType{
			Lexicographical: false,
			ZeroExtend:      true,
		}
		result := versionCompare(version, filterVersion, options)

		if isNaN(result) {
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

func checkVersionFilter(
	version string,
	filterVersions []string,
	operator string) bool {
	if !version {
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
	if (parsedOperator != = "=") {
		// remove any non-number and . characters, and remove everything after a hyphen
		// eg. 1.2.3a-b6 becomes 1.2.3
		regex1, err := regexp.Compile("[^(\\d|.|\\-)]/g")
		regex2, err := regexp.Compile("-.*/g")

		parsedVersion = replace(replace(parsedVersion, regex1, ""), regex2, "")

		var mappedFilterVersions []string
		for _, filterVersion := range parsedFilterVersions {
			mappedFilterVersions = append(mappedFilterVersions, replace(
				regex1.
					replace(filterVersion, regex1, ""), regex2, ""))
		}
		// Replace Array.map(), because you can"t access captured data in a closure
		for let
		i = 0
		i < filterVersions.length
		i++) {
			mappedFilterVersions.push(replace(replace(filterVersions[i], regex1, ""), regex2, ""))
		}
		parsedFilterVersions = mappedFilterVersions
	}

	parsedVersion = convertToSemanticVersion(parsedVersion)

	let
	passed = false
	// Replace Array.some(), because you can"t access captured data in a closure
	for let
	i = 0
	i < parsedFilterVersions.length
	i++) {
		if checkVersionValue(parsedFilterVersions[i], parsedVersion, operator) {
			passed = true
			break
		}
	}

	return !not ? passed:
	!passed
}

export function _checkNumberFilter(num: f64, filterNums: f64[], operator: string | null): bool {
if (operator && isString(operator)) {
if (operator == = "exist") {
return !isNaN(num)
} else if (operator == = "!exist") {
return isNaN(num)
}
}

if (isNaN(num)) {
return false
}

// replace filterNums.some() logic
let someValue = false
for (let i = 0; i < filterNums.length; i++) {
const filterNum = filterNums[i]
if (isNaN(filterNum)) {
continue
}

if (operator == = "=") {
someValue = num == = filterNum
} else if (operator == = "!=") {
someValue = num != = filterNum
} else if (operator == = ">") {
someValue = num > filterNum
} else if (operator == = ">=") {
someValue = num >= filterNum
} else if (operator == = "<") {
someValue = num < filterNum
} else if (operator == = "<=") {
someValue = num <= filterNum
} else {
continue
}

if (someValue) {
return true
}
}
return someValue
}

export function checkNumbersFilterJSONValue(jsonValue: JSON.Value, filter: UserFilter): bool {
return _checkNumbersFilter(getF64FromJSONValue(jsonValue), filter)
}

function _checkNumbersFilter(number: f64, filter: UserFilter): bool {
const operator = filter.comparator
const values = getFilterValuesAsF64(filter)
return _checkNumberFilter(number, values, operator)
}


func _checkStringsFilter(str string, filter UserFilter) bool {
	operator := filter.Comparator
	values := getFilterValuesAsString(filter)
}
export function _checkStringsFilter(string: string | null, filter: UserFilter): bool {
const operator = filter.comparator
const values = getFilterValuesAsStrings(filter)

if (operator == = "=") {
return string != = null && values.includes(string)
} else if (operator == = "!=") {
return string != = null && !values.includes(string)
} else if (operator == = "exist") {
return string != = null && string != = ""
} else if (operator == = "!exist") {
return string == = null || string == = ""
} else if (operator == = "contain") {
return string != = null && !!findString(values, string)
} else if (operator == = "!contain") {
return string == = null || !findString(values, string)
} else {
return isString(string)
}
}

export function _checkBooleanFilter(bool: bool, filter: UserFilter): bool {
const operator = filter.comparator
const values = getFilterValuesAsBoolean(filter)

if (operator == = "contain" || operator == = "=") {
return isBoolean(bool) && values.includes(bool)
} else if (operator == = "!contain" || operator === "!=") {
return isBoolean(bool) && !values.includes(bool)
} else if (operator == = "exist") {
return isBoolean(bool)
} else if (operator == = "!exist") {
return !isBoolean(bool)
} else {
return false
}
}

export function _checkVersionFilters(appVersion: string | null, filter: UserFilter): bool {
const operator = filter.comparator
const values = getFilterValuesAsStrings(filter)
// dont need to do semver if they"re looking for an exact match. Adds support for non semver versions.
if (operator == = "=") {
return _checkStringsFilter(appVersion, filter)
} else {
return checkVersionFilter(appVersion, values, operator)
}
}

export function _checkCustomData(data: JSON.Obj | null, filter: CustomDataFilter): bool {
const operator = filter.comparator

const dataValue = data ? data.get(filter.dataKey) : null

if (operator == = "exist") {
return checkValueExists(dataValue)
} else if (operator == = "!exist") {
return !checkValueExists(dataValue)
} else if (filter.dataKeyType == = "String" && dataValue && (dataValue.isString || dataValue.isNull)) {
if (dataValue.isNull) {
return _checkStringsFilter(null, filter)
} else {
const jsonStr = dataValue as JSON.Str
return _checkStringsFilter(jsonStr.valueOf(), filter)
}
} else if (filter.dataKeyType == = "Number"
&& dataValue && (dataValue.isFloat || dataValue.isInteger)) {
return checkNumbersFilterJSONValue(dataValue, filter)
} else if (filter.dataKeyType == = "Boolean" && dataValue && dataValue.isBool) {
const boolValue = dataValue as JSON.Bool
const result = _checkBooleanFilter(boolValue.valueOf(), filter)
return result
} else if (!dataValue && operator == = "!=") {
return true
} else {
return false
}
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
