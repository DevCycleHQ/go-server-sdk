package native_bucketing

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
