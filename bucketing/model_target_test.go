package bucketing

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// A test that parses an Audience from JSON stored in testdata/audience.json
func TestAudience_Parsing(t *testing.T) {
	jsonAudience, err := os.ReadFile("testdata/audience.json")
	require.NoError(t, err)

	var audience Audience
	err = json.Unmarshal(jsonAudience, &audience)
	require.NoError(t, err)

	require.Equal(t, Audience{
		NoIdAudience: NoIdAudience{
			Filters: &AudienceOperator{
				Operator: "and",
				Filters: MixedFilters{
					&CustomDataFilter{
						UserFilter: &UserFilter{
							filter: filter{
								Type:       "user",
								SubType:    "customData",
								Comparator: "=",
							},
							Values: []any{
								"iYI6uwZed0ip",
								"QqDKIhOwJqGz",
								"BkWS2ug4LiRg",
								"h6fCse1VCIo1",
							},
							CompiledStringVals: []string{
								"iYI6uwZed0ip",
								"QqDKIhOwJqGz",
								"BkWS2ug4LiRg",
								"h6fCse1VCIo1",
							},
						},
						DataKey:     "data-key-6",
						DataKeyType: "String",
					},
					&UserFilter{
						filter: filter{
							Type:       "user",
							SubType:    "user_id",
							Comparator: "=",
						},
						Values: []any{
							"user_680f420d-a65f-406c-8aaf-0b39a617e696",
						},
						CompiledStringVals: []string{
							"user_680f420d-a65f-406c-8aaf-0b39a617e696",
						},
					},
					&AudienceMatchFilter{
						filter: filter{
							Type:       "audienceMatch",
							Comparator: "!=",
						},
						Audiences: []string{
							"7db4d6f7e53543e4a413ac539477bac6",
							"145f66b2bfce4e7e9c8bd3a432e28c8d",
						},
					},
					&AllFilter{},
					&OptInFilter{},
					&AudienceOperator{
						Operator: "and",
						Filters: MixedFilters{
							&AllFilter{},
						},
					},
				},
			},
		},
		Id: "2d61e8001089444e9270bc316c294828",
	}, audience)

	filters := audience.NoIdAudience.Filters.Filters

	customDataFilter := filters[0].(*CustomDataFilter)
	require.Equal(t, "user", customDataFilter.GetType())
	require.Equal(t, "customData", customDataFilter.GetSubType())
	require.Equal(t, "=", customDataFilter.GetComparator())

	userFilter := filters[1].(*UserFilter)
	require.Equal(t, "user", userFilter.GetType())
	require.Equal(t, "user_id", userFilter.GetSubType())
	require.Equal(t, "=", userFilter.GetComparator())

	audienceFilter := filters[2].(*AudienceMatchFilter)
	require.Equal(t, "audienceMatch", audienceFilter.GetType())
	require.Equal(t, "!=", audienceFilter.GetComparator())
}
