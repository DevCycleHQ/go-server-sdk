package native_bucketing

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
					&AudienceFilter{
						filter: filter{
							Type: "all",
						},
					},
					&AudienceFilter{
						filter: filter{
							Type: "optIn",
						},
					},
					OperatorFilter{
						Operator: &AudienceOperator{
							Operator: "and",
							Filters: []BaseFilter{
								&AudienceFilter{
									filter: filter{
										Type: "all",
									},
								},
							},
						},
					},
				},
			},
		},
		Id: "2d61e8001089444e9270bc316c294828",
	}, audience)

	filters := audience.NoIdAudience.Filters.Filters

	require.Equal(t, "user", filters[0].GetType())
	require.Equal(t, "customData", filters[0].GetSubType())
	require.Equal(t, "=", filters[0].GetComparator())

	require.Equal(t, "user", filters[1].GetType())
	require.Equal(t, "user_id", filters[1].GetSubType())
	require.Equal(t, "=", filters[1].GetComparator())

	require.Equal(t, "audienceMatch", filters[2].GetType())
	require.Equal(t, "!=", filters[2].GetComparator())

	require.Equal(t, "all", filters[3].GetType())

	require.Equal(t, "optIn", filters[4].GetType())

	operator, isOperator := filters[5].GetOperator()
	require.True(t, isOperator)
	require.NotNil(t, operator)
}
