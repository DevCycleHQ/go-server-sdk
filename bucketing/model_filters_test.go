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
