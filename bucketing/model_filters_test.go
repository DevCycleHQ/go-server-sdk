package bucketing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckCustomData(t *testing.T) {
	tests := []struct {
		name        string
		comparator  string
		values      []interface{}
		dataKeyType string
		dataKey     string
		expected    bool
		data        map[string]interface{}
	}{
		// String Value Filter Tests
		{"should return false if filter and no data", ComparatorEqual, []interface{}{"value"}, "String", "strKey", false, map[string]interface{}{}},
		{"should return false if filter and nil data", ComparatorEqual, []interface{}{"value"}, "String", "strKey", false, nil},
		{"should return true if string value is equal", ComparatorEqual, []interface{}{"value"}, "String", "strKey", true, map[string]interface{}{"strKey": "value"}},
		{"should return true if string is one OR value", ComparatorEqual, []interface{}{"value", "value too"}, "String", "strKey", true, map[string]interface{}{"strKey": "value"}},
		{"should return false if string value is not equal", ComparatorEqual, []interface{}{"value"}, "String", "strKey", false, map[string]interface{}{"strKey": "rutabaga"}},
		{"should return false if string value is not equal (empty string)", ComparatorEqual, []interface{}{"value"}, "String", "strKey", false, map[string]interface{}{"strKey": ""}},
		{"should return false if string value is not present", ComparatorEqual, []interface{}{"value"}, "String", "strKey", false, map[string]interface{}{"otherKey": "something else"}},
		{"should return true if string is not equal to multiple values", ComparatorNotEqual, []interface{}{"value1", "value2", "value3"}, "String", "strKey", true, map[string]interface{}{"strKey": "value"}},
		// Number Value Filter Tests
		{"should return true if number value is equal", ComparatorEqual, []interface{}{float64(0)}, "Number", "numKey", true, map[string]interface{}{"numKey": float64(0)}},
		{"should return true if number is one OR value", ComparatorEqual, []interface{}{float64(0), float64(1)}, "Number", "numKey", true, map[string]interface{}{"numKey": float64(1)}},
		{"should return false if number value is not equal", ComparatorEqual, []interface{}{float64(0)}, "Number", "numKey", false, map[string]interface{}{"numKey": float64(1)}},
		// Boolean Value Filter Tests
		{"should return true if bool value is equal", ComparatorEqual, []interface{}{false}, "Boolean", "boolKey", true, map[string]interface{}{"boolKey": false}},
		{"should return false if bool value is not equal", ComparatorEqual, []interface{}{false}, "Boolean", "boolKey", false, map[string]interface{}{"boolKey": true}},
		// != Value Filter Tests
		{"should return true if no custom data is provided with not equal filter value", ComparatorNotEqual, []interface{}{"value"}, "String", "strKey", true, map[string]interface{}{}},
		// !exist Filter Tests
		{"should return true if no custom data is provided with not exists filter value", ComparatorNotExist, []interface{}{"value"}, "String", "strKey", true, map[string]interface{}{}},
		{"should return false if custom data is provided with not exists filter value", ComparatorNotExist, []interface{}{"value"}, "String", "strKey", false, map[string]interface{}{"strKey": "value"}},

		// Contains filter tests
		{"should return true if custom data contains value", ComparatorContain, []interface{}{"FP"}, "String", "last_order_no", true, map[string]interface{}{"last_order_no": "FP2423423"}},
		// !Contains filter tests
		{"should return false if custom data contains value with !contain", ComparatorNotContain, []interface{}{"FP"}, "String", "last_order_no", false, map[string]interface{}{"last_order_no": "FP2423423"}},

		// Exists filter tests
		{"should return true if custom data contains field with value", ComparatorExist, []interface{}{}, "String", "last_order_no", true, map[string]interface{}{"last_order_no": "FP2423423"}},
		{"should return false if custom data doesn't contain field with value", ComparatorExist, []interface{}{}, "String", "last_order_no", false, map[string]interface{}{"otherField": "value"}},
		{"should return false if custom data empty with exists comparator ", ComparatorExist, []interface{}{}, "String", "last_order_no", false, map[string]interface{}{}},
	}
	for _, test := range tests {
		testFilter := &CustomDataFilter{
			UserFilter: &UserFilter{
				filter: filter{
					Type:       "user",
					SubType:    "customData",
					Comparator: test.comparator,
				},
				Values: test.values,
			},
			DataKey:     test.dataKey,
			DataKeyType: test.dataKeyType,
		}

		require.NoError(t, testFilter.Initialize())
		require.NoError(t, testFilter.UserFilter.Initialize())

		result := checkCustomData(test.data, nil, testFilter)
		if result != test.expected {
			t.Errorf("Test %s failed. Expected %t, got %t", test.name, test.expected, result)
		}

		// test again but use the data as clientCustomData instead to make sure it still works
		result2 := checkCustomData(nil, test.data, testFilter)
		if result2 != test.expected {
			t.Errorf("Test %s (clientCustomData variation) failed. Expected %t, got %t", test.name, test.expected, result)
		}
	}
}

func TestCheckStringsFilter(t *testing.T) {
	tests := []struct {
		name       string
		comparator string
		values     []string
		subject    string
		expected   bool
	}{
		{"=_empty test, no values", ComparatorEqual, []string{}, "", false},
		{"=_empty test with values", ComparatorEqual, []string{"1", "2"}, "", false},

		{"=_match", ComparatorEqual, []string{"foo"}, "foo", true},
		{"=_match in list", ComparatorEqual, []string{"iPhone OS", "Android", "Blackberry"}, "Android", true},
		{"=_nomatch", ComparatorEqual, []string{"foo"}, "fo", false},

		{"!=_empty", ComparatorNotEqual, []string{""}, "", false},
		{"!=_match", ComparatorNotEqual, []string{"foo"}, "foo", false},
		{"!=_match in list", ComparatorNotEqual, []string{"iPhone OS", "Android", "Blackberry"}, "Android", false},
		{"!=_nomatch", ComparatorNotEqual, []string{"foo"}, "bar", true},

		{"exist_empty test", ComparatorExist, []string{}, "", false},
		{"exist_notempty test, no values", ComparatorExist, []string{}, "string", true},
		{"exist_notempty test, with values", ComparatorExist, []string{"hello", "world"}, "string", true},

		{"!exist_empty", ComparatorNotExist, []string{}, "", true},
		{"!exist_notempty, no values", ComparatorNotExist, []string{}, "exists", false},
		{"!exist_notempty test, with values", ComparatorNotExist, []string{"hello", "world"}, "exists", false},

		{"contain_empty test", ComparatorContain, []string{""}, "", false},
		{"contain_match, browser filter", ComparatorContain, []string{"Chrome"}, "Chrome", true},
		{"contain_match, device filter", ComparatorContain, []string{"Desktop"}, "Desktop", true},
		{"contain_partialMatch", ComparatorContain, []string{"hello"}, "helloWorld", true},
		{"contain_nomatch", ComparatorContain, []string{"foo"}, "bar", false},

		{"contain_empty test", ComparatorNotContain, []string{""}, "", true},
		{"contain_match", ComparatorNotContain, []string{"Desktop"}, "Desktop", false},
		{"contain_partialMatch", ComparatorNotContain, []string{"oob"}, "foobar", false},
		{"contain_nomatch", ComparatorNotContain, []string{"foo"}, "bar", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter := &UserFilter{
				filter: filter{
					Comparator: test.comparator,
				},
				CompiledStringVals: test.values,
			}
			actual := checkStringsFilter(test.subject, filter)
			require.Equal(t, test.expected, actual)
		})
	}
}

func Test_CheckBooleanFilter(t *testing.T) {
	tests := []struct {
		name       string
		comparator string
		values     []bool
		testParam  bool
		expected   bool
	}{
		{"contain_true", ComparatorContain, []bool{true}, true, true},
		{"contain false", ComparatorContain, []bool{true}, false, false},
		{"equal true", ComparatorEqual, []bool{true}, true, true},
		{"equal false", ComparatorEqual, []bool{true}, false, false},
		{"!equal true", ComparatorNotEqual, []bool{false}, true, true},
		{"!contains true", ComparatorNotContain, []bool{false}, true, true},
		{"exist", ComparatorExist, []bool{}, true, true},
		{"!exist", ComparatorNotExist, []bool{}, true, false},
		{"Unsupported comparator 1", ComparatorGreater, []bool{}, true, false},
		{"Unsupported comparator 2", ComparatorGreater, []bool{true}, true, false},
	}

	for _, test := range tests {
		filter := &UserFilter{
			filter: filter{
				Type:       "user",
				SubType:    "customData",
				Comparator: test.comparator,
			},
			CompiledBoolVals: test.values,
		}
		result := _checkBooleanFilter(test.testParam, filter)
		if result != test.expected {
			t.Errorf("Test %v failed: Expected %t, but got %t", test.name, test.expected, result)
		}
	}
}

func TestCheckVersionFilters(t *testing.T) {
	type VersionTestCase struct {
		expected   bool
		version    string
		values     []interface{}
		comparator string
	}

	type TestCaseGroup struct {
		name      string
		testCases []VersionTestCase
	}

	groups := []TestCaseGroup{
		{
			name: "should return true if string versions equal",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1"}, comparator: "="},
				{expected: true, version: "1.1", values: []interface{}{"1.1"}, comparator: "="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.1.1"}, comparator: "="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1"}, comparator: "="},
			},
		},
		{
			name: "should return false if string versions not equal",
			testCases: []VersionTestCase{
				{expected: false, version: "", values: []interface{}{"2"}, comparator: "="},
				{expected: false, version: "1", values: []interface{}{"2"}, comparator: "="},
				{expected: false, version: "1.1", values: []interface{}{"1.2"}, comparator: "="},
				{expected: false, version: "1.1", values: []interface{}{"1.1.1"}, comparator: "="},
				{expected: false, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: "="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.1"}, comparator: "="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.1."}, comparator: "="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.2.3"}, comparator: "="},
			},
		},
		{
			name: "should return false if string versions not equal",
			testCases: []VersionTestCase{
				{expected: false, version: "1", values: []interface{}{"1"}, comparator: "!="},
				{expected: false, version: "1.1", values: []interface{}{"1.1"}, comparator: "!="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.1.1"}, comparator: "!="},
				{expected: false, version: "1.1.", values: []interface{}{"1.1"}, comparator: "!="},
			},
		},
		{
			name: "should return true if string versions not equal",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"2"}, comparator: "!="},
				{expected: true, version: "1.1", values: []interface{}{"1.2"}, comparator: "!="},
				{expected: true, version: "1.1", values: []interface{}{"1.1.1"}, comparator: "!="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: "!="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.1"}, comparator: "!="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.1."}, comparator: "!="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.2.3"}, comparator: "!="},
			},
		},
		{
			name: "should return false if string versions greater than",
			testCases: []VersionTestCase{
				{expected: false, version: "", values: []interface{}{"1"}, comparator: ">"},
				{expected: false, version: "1", values: []interface{}{"1"}, comparator: ">"},
				{expected: false, version: "1.1", values: []interface{}{"1.1"}, comparator: ">"},
				{expected: false, version: "1.1.1", values: []interface{}{"1.1.1"}, comparator: ">"},
				{expected: false, version: "1.1.", values: []interface{}{"1.1"}, comparator: ">"},
				{expected: false, version: "1", values: []interface{}{"2"}, comparator: ">"},
				{expected: false, version: "1.1", values: []interface{}{"1.2"}, comparator: ">"},
				{expected: false, version: "1.1", values: []interface{}{"1.1.1"}, comparator: ">"},
				{expected: false, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: ">"},
				{expected: false, version: "1.1.1", values: []interface{}{"1.2.3"}, comparator: ">"},
			},
		},
		{
			name: "should return true if string versions greater than",
			testCases: []VersionTestCase{
				{expected: true, version: "2", values: []interface{}{"1"}, comparator: ">"},
				{expected: true, version: "1.2", values: []interface{}{"1.1"}, comparator: ">"},
				{expected: true, version: "2.1", values: []interface{}{"1.1"}, comparator: ">"},
				{expected: true, version: "1.2.1", values: []interface{}{"1.2"}, comparator: ">"},
				{expected: true, version: "1.2.", values: []interface{}{"1.1"}, comparator: ">"},
				{expected: true, version: "1.2.1", values: []interface{}{"1.1.1"}, comparator: ">"},
				{expected: true, version: "1.2.2", values: []interface{}{"1.2"}, comparator: ">"},
				{expected: true, version: "1.2.2", values: []interface{}{"1.2.1"}, comparator: ">"},
				{expected: true, version: "4.8.241", values: []interface{}{"4.8"}, comparator: ">"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4"}, comparator: ">"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8"}, comparator: ">"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.2"}, comparator: ">"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.241.0"}, comparator: ">"},
			},
		},
		{
			name: "should return false if string versions greater than or equal",
			testCases: []VersionTestCase{
				{expected: false, version: "", values: []interface{}{"2"}, comparator: ">="},
				{expected: false, version: "1", values: []interface{}{"2"}, comparator: ">="},
				{expected: false, version: "1.1", values: []interface{}{"1.2"}, comparator: ">="},
				{expected: false, version: "1.1", values: []interface{}{"1.1.1"}, comparator: ">="},
				{expected: false, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: ">="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.2.3"}, comparator: ">="},
				{expected: false, version: "4.8.241", values: []interface{}{"4.9"}, comparator: ">="},
				{expected: false, version: "4.8.241.2", values: []interface{}{"5"}, comparator: ">="},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.9"}, comparator: ">="},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.8.242"}, comparator: ">="},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.8.241.5"}, comparator: ">="},
			},
		},
		{
			name: "should return true if string versions greater than or equal",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1"}, comparator: ">="},
				{expected: true, version: "1.1", values: []interface{}{"1.1"}, comparator: ">="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.1.1"}, comparator: ">="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1"}, comparator: ">="},
				{expected: true, version: "2", values: []interface{}{"1"}, comparator: ">="},
				{expected: true, version: "1.2", values: []interface{}{"1.1"}, comparator: ">="},
				{expected: true, version: "2.1", values: []interface{}{"1.1"}, comparator: ">="},
				{expected: true, version: "1.2.1", values: []interface{}{"1.2"}, comparator: ">="},
				{expected: true, version: "1.2.", values: []interface{}{"1.1"}, comparator: ">="},
				{expected: true, version: "1.2.1", values: []interface{}{"1.1.1"}, comparator: ">="},
				{expected: true, version: "1.2.2", values: []interface{}{"1.2"}, comparator: ">="},
				{expected: true, version: "1.2.2", values: []interface{}{"1.2.1"}, comparator: ">="},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4"}, comparator: ">="},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8"}, comparator: ">="},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.2"}, comparator: ">="},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.241.0"}, comparator: ">="},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.241.2"}, comparator: ">="},
			},
		},
		{
			name: "should work if version has other characters",
			testCases: []VersionTestCase{
				{expected: true, version: "1.2.2", values: []interface{}{"v1.2.1-2v3asda"}, comparator: ">="},
				{expected: true, version: "1.2.2", values: []interface{}{"v1.2.1-va1sda"}, comparator: ">"},
				{expected: true, version: "1.2.1", values: []interface{}{"v1.2.1-vasd32a"}, comparator: ">="},
				{expected: false, version: "1.2.1", values: []interface{}{"v1.2.1-vasda"}, comparator: "="},
				{expected: false, version: "v1.2.1-va21sda", values: []interface{}{"v1.2.1-va13sda"}, comparator: "="},
				{expected: false, version: "1.2.0", values: []interface{}{"v1.2.1-vas1da"}, comparator: ">="},
				{expected: true, version: "1.2.1", values: []interface{}{"v1.2.1- va34sda"}, comparator: "<="},
				{expected: true, version: "1.2.0", values: []interface{}{"v1.2.1-vas3da"}, comparator: "<="},
			},
		},
		{
			name: "should return true if string versions less than",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"2"}, comparator: "<"},
				{expected: true, version: "1.1", values: []interface{}{"1.2"}, comparator: "<"},
				{expected: true, version: "1.1", values: []interface{}{"1.1.1"}, comparator: "<"},
				{expected: true, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: "<"},
				{expected: true, version: "1.1.1", values: []interface{}{"1.2.3"}, comparator: "<"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"5"}, comparator: "<"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.9"}, comparator: "<"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.242"}, comparator: "<"},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.241.5"}, comparator: "<"}},
		},
		{
			name: "should return false if string versions less than",
			testCases: []VersionTestCase{
				{expected: false, version: "", values: []interface{}{"1"}, comparator: "<"},
				{expected: false, version: "1", values: []interface{}{"1"}, comparator: "<"},
				{expected: false, version: "1.1", values: []interface{}{"1.1"}, comparator: "<"},
				{expected: false, version: "1.1.1", values: []interface{}{"1.1.1"}, comparator: "<"},
				{expected: false, version: "1.1.", values: []interface{}{"1.1"}, comparator: "<"},
				{expected: false, version: "2", values: []interface{}{"1"}, comparator: "<"},
				{expected: false, version: "1.2", values: []interface{}{"1.1"}, comparator: "<"},
				{expected: false, version: "2.1", values: []interface{}{"1.1"}, comparator: "<"},
				{expected: false, version: "1.2.1", values: []interface{}{"1.2"}, comparator: "<"},
				{expected: false, version: "1.2.", values: []interface{}{"1.1"}, comparator: "<"},
				{expected: false, version: "1.2.1", values: []interface{}{"1.1.1"}, comparator: "<"},
				{expected: false, version: "1.2.2", values: []interface{}{"1.2"}, comparator: "<"},
				{expected: false, version: "1.2.2", values: []interface{}{"1.2."}, comparator: "<"},
				{expected: false, version: "1.2.2", values: []interface{}{"1.2.1"}, comparator: "<"},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4"}, comparator: "<"},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.8"}, comparator: "<"},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.8.241"}, comparator: "<"},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.8.241.0"}, comparator: "<"}},
		},
		{
			name: "should return true if string versions less than or equal",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1"}, comparator: "<="},
				{expected: true, version: "1.1", values: []interface{}{"1.1"}, comparator: "<="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.1.1"}, comparator: "<="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1"}, comparator: "<="},
				{expected: true, version: "1", values: []interface{}{"2"}, comparator: "<="},
				{expected: true, version: "1.1", values: []interface{}{"1.2"}, comparator: "<="},
				{expected: true, version: "1.1", values: []interface{}{"1.1.1"}, comparator: "<="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: "<="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.2.3"}, comparator: "<="},
				{expected: true, version: "4.8.241.2", values: []interface{}{"4.8.241.2"}, comparator: "<="}},
		},
		{
			name: "should return false if string versions less than or equal",
			testCases: []VersionTestCase{
				{expected: false, version: "", values: []interface{}{"1"}, comparator: "<="},
				{expected: false, version: "2", values: []interface{}{"1"}, comparator: "<="},
				{expected: false, version: "1.2", values: []interface{}{"1.1"}, comparator: "<="},
				{expected: false, version: "2.1", values: []interface{}{"1.1"}, comparator: "<="},
				{expected: false, version: "1.2.1", values: []interface{}{"1.2"}, comparator: "<="},
				{expected: false, version: "1.2.", values: []interface{}{"1.1"}, comparator: "<="},
				{expected: false, version: "1.2.1", values: []interface{}{"1.1.1"}, comparator: "<="},
				{expected: false, version: "1.2.2", values: []interface{}{"1.2"}, comparator: "<="},
				{expected: false, version: "1.2.2", values: []interface{}{"1.2."}, comparator: "<="},
				{expected: false, version: "1.2.2", values: []interface{}{"1.2.1"}, comparator: "<="},
				{expected: false, version: "4.8.241.2", values: []interface{}{"4.8.241"}, comparator: "<="}},
		},
		{
			name: "should return true if any numbers equal array",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1", "1.1"}, comparator: "="},
				{expected: true, version: "1.1", values: []interface{}{"1", "1.1"}, comparator: "="},
				{expected: true, version: "1.1", values: []interface{}{"1.1", ""}, comparator: "="}},
		},
		{
			name: "should return false if all numbers not equal array",
			testCases: []VersionTestCase{
				{expected: false, version: "1", values: []interface{}{"2", "1.1"}, comparator: "="},
				{expected: false, version: "1.1", values: []interface{}{"1.2", "1"}, comparator: "="}},
		},
		{
			name: "should return true if any string versions equal array",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1", "1.1"}, comparator: "="},
				{expected: true, version: "1.1", values: []interface{}{"1.1", "1"}, comparator: "="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.1.1", "1.1"}, comparator: "="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1", "1.1"}, comparator: "="}},
		},
		{
			name: "should return false if all string versions not equal array",
			testCases: []VersionTestCase{
				{expected: false, version: "", values: []interface{}{"2", "3"}, comparator: "="},
				{expected: false, version: "1", values: []interface{}{"2", "3"}, comparator: "="},
				{expected: false, version: "1.1", values: []interface{}{"1.2", "1.2"}, comparator: "="},
				{expected: false, version: "1.1", values: []interface{}{"1.1.1", "1.2"}, comparator: "="},
				{expected: false, version: "1.1.", values: []interface{}{"1.1.1", "1.2"}, comparator: "="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.1", "1.1"}, comparator: "="},
				{expected: false, version: "1.1.1", values: []interface{}{"1", "1.1."}, comparator: "="},
				{expected: false, version: "1.1.1", values: []interface{}{"1.2.3", "1."}, comparator: "="}},
		},
		{
			name: " should return false if multiple versions do not equal the version",
			testCases: []VersionTestCase{
				{expected: false, version: "1", values: []interface{}{"2", "1"}, comparator: "!="},
				{expected: false, version: "1.1", values: []interface{}{"1.2", "1.1"}, comparator: "!="}},
		},
		{
			name: "should return true if multiple versions do not equal version",
			testCases: []VersionTestCase{
				{expected: true, version: "1.1", values: []interface{}{"1.1.1", "1.2"}, comparator: "!="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1.1", "1"}, comparator: "!="}},
		},

		{
			name: "should return false if any string versions not greater than array",
			testCases: []VersionTestCase{
				{expected: false, version: "1", values: []interface{}{"1", "1"}, comparator: ">"},
				{expected: false, version: "1.1", values: []interface{}{"1.1", "1.1.", "1.1"}, comparator: ">"},
				{expected: false, version: "1", values: []interface{}{"2"}, comparator: ">"},
				{expected: false, version: "1.1", values: []interface{}{"1.1.0"}, comparator: ">"}},
		},

		{
			name: "should return true any if string versions greater than array",
			testCases: []VersionTestCase{
				{expected: true, version: "2", values: []interface{}{"1", "2.0"}, comparator: ">"},
				{expected: true, version: "1.2.1", values: []interface{}{"1.2", "1.2"}, comparator: ">"},
				{expected: true, version: "1.2.", values: []interface{}{"1.1", "1.9."}, comparator: ">"}},
		},

		{
			name: "should return false if all string versions not greater than or equal array",
			testCases: []VersionTestCase{
				{expected: false, version: "1", values: []interface{}{"2", "1.2"}, comparator: ">="},
				{expected: false, version: "1.1", values: []interface{}{"1.2"}, comparator: ">="},
				{expected: false, version: "1.1", values: []interface{}{"1.1.1", "1.2"}, comparator: ">="}},
		},

		{
			name: "should return true if any string versions greater than or equal array",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1", "1.1"}, comparator: ">="},
				{expected: true, version: "1.1", values: []interface{}{"1.1", "1"}, comparator: ">="},
				{expected: true, version: "1.1.1", values: []interface{}{"1.2", "1.1.1"}, comparator: ">="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1"}, comparator: ">="},
				{expected: true, version: "2", values: []interface{}{"1", "3"}, comparator: ">="}},
		},

		{
			name: "should return true if any string versions less than array",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"2", "1"}, comparator: "<"},
				{expected: true, version: "1.1", values: []interface{}{"1.2", "1.5"}, comparator: "<"},
				{expected: true, version: "1.1.", values: []interface{}{"1.1.1"}, comparator: "<"}},
		},

		{
			name: "should return false if all string versions less than array",
			testCases: []VersionTestCase{
				{expected: false, version: "1", values: []interface{}{"1", "1.0"}, comparator: "<"},
				{expected: false, version: "1.1.", values: []interface{}{"1.1", "1.1.0"}, comparator: "<"}},
		},

		{
			name: "should return true if any string versions less than or equal array",
			testCases: []VersionTestCase{
				{expected: true, version: "1", values: []interface{}{"1", "5"}, comparator: "<="},
				{expected: true, version: "1.1", values: []interface{}{"1.1", "1.1."}, comparator: "<="},
				{expected: true, version: "1.1.", values: []interface{}{"1.1.1", "1.1."}, comparator: "<="}},
		},

		{
			name: "should return false if all string versions not less than or equal array",
			testCases: []VersionTestCase{
				{expected: false, version: "2", values: []interface{}{"1", "1.9"}, comparator: "<="},
				{expected: false, version: "1.2.1", values: []interface{}{"1.2", "1.2"}, comparator: "<="},
				{expected: false, version: "1.2.", values: []interface{}{"1.1", "1.1.9"}, comparator: "<="}},
		},
	}
	for _, tg := range groups {
		for x, tc := range tg.testCases {
			versionFilter := &UserFilter{
				filter: filter{
					Type:       "user",
					SubType:    "appVersion",
					Comparator: tc.comparator,
					Operator:   OperatorAnd,
				},
				Values: tc.values,
			}
			require.NoError(t, versionFilter.Initialize())
			result := checkVersionFilters(tc.version, versionFilter)
			if result != tc.expected {
				t.Errorf("Group: %s #%d: Expected %t, but got %t", tg.name, x, tc.expected, result)
			}
		}
	}

}
