package bucketing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckStringsFilter(t *testing.T) {
	tests := []struct {
		name       string
		comparator string
		values     []string
		subject    string
		expected   bool
	}{
		{"=_empty", ComparatorEqual, []string{""}, "", false},
		{"=_match", ComparatorEqual, []string{"foo"}, "foo", true},
		{"=_nomatch", ComparatorEqual, []string{"foo"}, "fo", false},
		{"!=_empty", ComparatorNotEqual, []string{""}, "", false},
		{"!=_match", ComparatorNotEqual, []string{"foo"}, "foo", false},
		{"!=_nomatch", ComparatorNotEqual, []string{"foo"}, "bar", true},
		{"exist_empty", ComparatorExist, []string{}, "", false},
		{"exist_notempty", ComparatorExist, []string{}, "exists", true},
		{"exist_empty", ComparatorNotExist, []string{}, "", true},
		{"exist_notempty", ComparatorNotExist, []string{}, "exists", false},
		{"contain_empty", ComparatorContain, []string{""}, "", false},
		{"contain_match", ComparatorContain, []string{"oob"}, "foobar", true},
		{"contain_nomatch", ComparatorContain, []string{"foo"}, "bar", false},
		{"contain_empty", ComparatorNotContain, []string{""}, "", true},
		{"contain_match", ComparatorNotContain, []string{"oob"}, "foobar", false},
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
