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
				Operator_: "and",
				Filters_: MixedFilters{
					&UserFilter{},
					&UserFilter{},
				},
			},
		},
		Id: "2d61e8001089444e9270bc316c294828",
	}, audience)

}
