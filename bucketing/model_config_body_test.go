package bucketing

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/devcyclehq/go-server-sdk/v2/api"
)

func TestParseConfigBody(t *testing.T) {
	type testCase struct {
		name           string
		inputJSON      string
		expectedOutput *configBody
		expectError    bool
	}

	testCases := []testCase{
		{
			name:           "Bad JSON",
			inputJSON:      `{`,
			expectedOutput: nil,
			expectError:    true,
		},
		{
			name: "Missing project",
			inputJSON: `{
				"audiences": {},
				"environment": {},
				"features": [],
				"variables": []
			}`,
			expectedOutput: nil,
			expectError:    true,
		},
		{
			name: "Minimal valid config",
			inputJSON: `{
			  "project": {
			    "_id": "61535533396f00bab586cb17",
			    "key": "test-project",
			    "a0_organization": "org_12345612345",
			    "settings": {}
			  },
			  "environment": {
			    "_id": "6153553b8cf4e45e0464268d",
			    "key": "test-environment"
			  },
			  "features": [],
			  "variables": []
			}`,
			expectedOutput: &configBody{
				Project: api.Project{
					Id:               "61535533396f00bab586cb17",
					Key:              "test-project",
					A0OrganizationId: "org_12345612345",
				},
				Audiences: map[string]NoIdAudience{},
				Environment: api.Environment{
					Id:  "6153553b8cf4e45e0464268d",
					Key: "test-environment",
				},
				Features:               []*ConfigFeature{},
				Variables:              []*Variable{},
				variableIdMap:          map[string]*Variable{},
				variableKeyMap:         map[string]*Variable{},
				variableIdToFeatureMap: map[string]*ConfigFeature{},
			},
			expectError: false,
		},
		{
			name: "Minimal valid config",
			inputJSON: `{
			  "project": {
			    "_id": "61535533396f00bab586cb17",
			    "key": "test-project",
			    "a0_organization": "org_12345612345",
			    "settings": {}
			  },
			  "environment": {
			    "_id": "6153553b8cf4e45e0464268d",
			    "key": "test-environment"
			  },
			  "features": [],
			  "variables": []
			}`,
			expectedOutput: &configBody{
				Project: api.Project{
					Id:               "61535533396f00bab586cb17",
					Key:              "test-project",
					A0OrganizationId: "org_12345612345",
				},
				Audiences: map[string]NoIdAudience{},
				Environment: api.Environment{
					Id:  "6153553b8cf4e45e0464268d",
					Key: "test-environment",
				},
				Features:               []*ConfigFeature{},
				Variables:              []*Variable{},
				variableIdMap:          map[string]*Variable{},
				variableKeyMap:         map[string]*Variable{},
				variableIdToFeatureMap: map[string]*ConfigFeature{},
			},
			expectError: false,
		},
		{
			name: "Invalid variable type",
			inputJSON: `{
			  "project": {
			    "_id": "61535533396f00bab586cb17",
			    "key": "test-project",
			    "a0_organization": "org_12345612345",
			    "settings": {}
			  },
			  "environment": {
			    "_id": "6153553b8cf4e45e0464268d",
			    "key": "test-environment"
			  },
			  "features": [],
			  "variables": [{
				"_id": "id",
				"type": "squirrel",
				"key": "key"
			  }]
			}`,
			expectedOutput: nil,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := newConfig([]byte(tc.inputJSON), "", "")

			if tc.expectError {
				require.Error(t, err, "Expected error, got nil")
			} else {
				require.NoError(t, err, "Unexpected error: %v", err)
				require.Equal(t, tc.expectedOutput, result)
			}
		})
	}
}
