package native_bucketing

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

// A test that parses an Audience from JSON stored in testdata/audience.json
func TestAudience_Parsing(t *testing.T) {
	jsonAudience, err := ioutil.ReadFile("testdata/audience.json")
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
				},
			},
		},
		Id: "2d61e8001089444e9270bc316c294828",
	}, audience)

}
