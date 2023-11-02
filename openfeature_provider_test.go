package devcycle

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

func Test_DevCycleProvider_Metadata(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)
	require.Equal(t, "DevCycleProvider Local", provider.Metadata().Name)
	provider = getProviderForConfig(t, test_config, true)
	require.Equal(t, "DevCycleProvider Cloud", provider.Metadata().Name)
}

func Test_createUserFromEvaluationContext_NoUserID(t *testing.T) {
	_, err := createUserFromEvaluationContext(openfeature.FlattenedContext{})
	require.Error(t, err, "Expected error when userId is not provided")
}

func Test_createUserFromEvaluationContext_SimpleUser(t *testing.T) {
	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234"})
	require.NoError(t, err)
	require.Equal(t, "1234", user.UserId)

	user, err = createUserFromEvaluationContext(openfeature.FlattenedContext{"targetingKey": "1234"})
	require.NoError(t, err)
	require.Equal(t, "1234", user.UserId)

	user, err = createUserFromEvaluationContext(openfeature.FlattenedContext{"targetingKey": "1234", "userId": "5678"})
	require.NoError(t, err)
	require.Equal(t, "1234", user.UserId)
}

func Test_createUserFromEvaluationContext_AllUserProperties(t *testing.T) {
	ctx := openfeature.FlattenedContext{
		"userId":      "1234",
		"email":       "someone@example.com",
		"name":        "John Doe",
		"language":    "en",
		"country":     "US",
		"appVersion":  "1.0.0",
		"appBuild":    "1",
		"deviceModel": "iPhone X21",
	}
	user, err := createUserFromEvaluationContext(ctx)
	require.NoError(t, err)
	require.Equal(t, ctx["userId"], user.UserId)
	require.Equal(t, ctx["email"], user.Email)
	require.Equal(t, ctx["name"], user.Name)
	require.Equal(t, ctx["language"], user.Language)
	require.Equal(t, ctx["country"], user.Country)
	require.Equal(t, ctx["appVersion"], user.AppVersion)
	require.Equal(t, ctx["appBuild"], user.AppBuild)
	require.Equal(t, ctx["deviceModel"], user.DeviceModel)
	require.Nil(t, user.CustomData)
	require.Nil(t, user.PrivateCustomData)
}

func Test_createUserFromEvaluationContext_InvalidDataType(t *testing.T) {
	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234", "email": 1234})
	require.NoError(t, err)
	require.Empty(t, user.Email)
}

func Test_createUserFromEvaluationContext_CustomData(t *testing.T) {
	testCustomData := map[string]interface{}{"key1": "strVal", "key2": float64(1234), "key3": true, "key4": nil}
	testPrivateData := map[string]interface{}{"key1": "otherVal", "key2": float64(9999), "key3": false, "key4": nil}
	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234", "customData": testCustomData, "privateCustomData": testPrivateData})
	require.NoError(t, err)
	require.Equal(t, testCustomData, user.CustomData)
	require.Equal(t, testPrivateData, user.PrivateCustomData)
}

func Test_createUserFromEvaluationContext_NestedProperties(t *testing.T) {
	testCustomData := map[string]interface{}{"key1": "strVal", "nested": map[string]string{"child": "value"}}
	testPrivateData := map[string]interface{}{"key1": "otherVal", "nested": map[string]string{"child": "value"}}

	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234", "customData": testCustomData, "privateCustomData": testPrivateData})
	require.NoError(t, err)

	delete(testCustomData, "nested")
	delete(testPrivateData, "nested")
	require.Equal(t, testCustomData, user.CustomData)
	require.Equal(t, testPrivateData, user.PrivateCustomData)
}

func Test_setCustomDataValue(t *testing.T) {
	type DataTestCase struct {
		testName    string
		val         interface{}
		expectedVal interface{}
	}

	testCases := []DataTestCase{
		{"nil", nil, nil},
		{"string", "optimus prime", "optimus prime"},
		{"number", 3.14, 3.14},
		{"int64", int64(42), float64(42)},
		{"int32", int32(42), float64(42)},
		{"int", int(42), float64(42)},
		{"bool", true, true},
	}

	for _, testCase := range testCases {
		customData := make(map[string]interface{})
		setCustomDataValue(customData, "key", testCase.val)
		require.Equal(t, testCase.expectedVal, customData["key"])
	}
}

func getProviderForConfig(t *testing.T, config string, cloudBucketing bool) openfeature.FeatureProvider {
	t.Helper()

	httpCustomConfigMock(test_environmentKey, 200, config)

	client, err := NewClient(test_environmentKey, &Options{
		EnableCloudBucketing: cloudBucketing,
	})
	require.NoError(t, err)

	return client.OpenFeatureProvider()
}

func Test_BooleanEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "unknownFlag", false, evalCtx)

	require.False(t, resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_BooleanEvaluation_BadUserData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "test", false, evalCtx)

	require.False(t, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("targetingKey or userId must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_BooleanEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "test", false, evalCtx)

	require.True(t, resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_BooleanEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "test-string-variable", false, evalCtx)

	require.False(t, resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_StringEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.StringEvaluation(context.Background(), "unknownFlag", "default", evalCtx)

	require.Equal(t, "default", resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_StringEvaluation_BadUserData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.StringEvaluation(context.Background(), "test-string-variable", "default", evalCtx)

	require.Equal(t, "default", resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("targetingKey or userId must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_StringEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.StringEvaluation(context.Background(), "test-string-variable", "default", evalCtx)

	require.Equal(t, "on", resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_StringEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.StringEvaluation(context.Background(), "test-number-variable", "default", evalCtx)

	require.Equal(t, "default", resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_FloatEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.FloatEvaluation(context.Background(), "unknownFlag", 1.23, evalCtx)

	require.Equal(t, 1.23, resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_FloatEvaluation_BadUserData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.FloatEvaluation(context.Background(), "test", 1.23, evalCtx)

	require.Equal(t, 1.23, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("targetingKey or userId must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_FloatEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.FloatEvaluation(context.Background(), "test-number-variable", 1.23, evalCtx)

	require.Equal(t, float64(123), resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)

	resolutionDetail = provider.FloatEvaluation(context.Background(), "test-float-variable", 1.23, evalCtx)

	require.Equal(t, float64(4.56), resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_FloatEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.FloatEvaluation(context.Background(), "test-string-variable", float64(1.23), evalCtx)

	require.Equal(t, float64(1.23), resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_IntEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.IntEvaluation(context.Background(), "unknownFlag", int64(123), evalCtx)

	require.Equal(t, int64(123), resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_IntEvaluation_BadUserData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.IntEvaluation(context.Background(), "test", int64(123), evalCtx)

	require.Equal(t, int64(123), resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("targetingKey or userId must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_IntEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.IntEvaluation(context.Background(), "test-number-variable", 1, evalCtx)

	require.Equal(t, int64(123), resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)

	resolutionDetail = provider.IntEvaluation(context.Background(), "test-float-variable", 1, evalCtx)

	// 4.56 is rounded down to 4
	require.Equal(t, int64(4), resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_IntEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.IntEvaluation(context.Background(), "test-string-variable", int64(123), evalCtx)

	require.Equal(t, int64(123), resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_ObjectEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	defaultValue := map[string]any{
		"default": "value",
	}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "unknownFlag", defaultValue, evalCtx)

	require.Equal(t, defaultValue, resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_ObjectEvaluation_BadUserData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	defaultValue := map[string]any{
		"default": "value",
	}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "test", defaultValue, evalCtx)

	require.Equal(t, defaultValue, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("targetingKey or userId must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_ObjectEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	defaultValue := map[string]any{
		"default": "value",
	}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "test-json-variable", defaultValue, evalCtx)

	require.Equal(t, map[string]interface{}{"message": "a"}, resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_ObjectEvaluation_TargetMatchBadDefault(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	defaultValue := []string{"default"}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "test-json-variable", defaultValue, evalCtx)

	require.Equal(t, defaultValue, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("the default value for variable is not of type Boolean, Number, String, or JSON: test-json-variable"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_ObjectEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config, false)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	defaultValue := map[string]any{
		"default": "value",
	}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "test-string-variable", defaultValue, evalCtx)

	require.Equal(t, defaultValue, resolutionDetail.Value)
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

type StubClient struct {
	variable Variable
	err      error
}

func (c StubClient) Variable(userdata User, key string, defaultValue interface{}) (Variable, error) {
	return c.variable, c.err
}

func (c StubClient) IsLocalBucketing() bool { return true }

func TestEvaluationValueHandling(t *testing.T) {
	evalCtx := openfeature.FlattenedContext{"userId": "1234"}
	testCases := []struct {
		name        string
		method      string
		variable    Variable
		errorResult error
		expected    any
	}{
		{
			name:     "BooleanEvaluation default",
			method:   "BooleanEvaluation",
			variable: Variable{IsDefaulted: true},
			expected: openfeature.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason: openfeature.DefaultReason,
				},
			},
		},
		{
			name:     "BooleanEvaluation nil without default",
			method:   "BooleanEvaluation",
			variable: Variable{},
			expected: openfeature.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("Variable result is nil, but not defaulted"),
				},
			},
		},
		{
			name:     "BooleanEvaluation unexpected type",
			method:   "BooleanEvaluation",
			variable: Variable{BaseVariable: BaseVariable{Value: "not a bool"}},
			expected: openfeature.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewTypeMismatchResolutionError("Unexpected type in boolean variable result: string"),
				},
			},
		},
		{
			name:        "BooleanEvaluation error",
			method:      "BooleanEvaluation",
			errorResult: fmt.Errorf("an unexpected error!"),
			expected: openfeature.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("an unexpected error!"),
				},
			},
		},
		{
			name:     "StringEvaluation default",
			method:   "StringEvaluation",
			variable: Variable{IsDefaulted: true},
			expected: openfeature.StringResolutionDetail{
				Value: "default",
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason: openfeature.DefaultReason,
				},
			},
		},
		{
			name:     "StringEvaluation nil without default",
			method:   "StringEvaluation",
			variable: Variable{},
			expected: openfeature.StringResolutionDetail{
				Value: "default",
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("Variable result is nil, but not defaulted"),
				},
			},
		},
		{
			name:     "StringEvaluation unexpected type",
			method:   "StringEvaluation",
			variable: Variable{BaseVariable: BaseVariable{Value: 1234}},
			expected: openfeature.StringResolutionDetail{
				Value: "default",
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewTypeMismatchResolutionError("Unexpected type in string variable result: int"),
				},
			},
		},
		{
			name:        "StringEvaluation error",
			method:      "StringEvaluation",
			errorResult: fmt.Errorf("an unexpected error!"),
			expected: openfeature.StringResolutionDetail{
				Value: "default",
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("an unexpected error!"),
				},
			},
		},
		{
			name:     "FloatEvaluation default",
			method:   "FloatEvaluation",
			variable: Variable{IsDefaulted: true},
			expected: openfeature.FloatResolutionDetail{
				Value: 1.23,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason: openfeature.DefaultReason,
				},
			},
		},
		{
			name:     "FloatEvaluation nil without default",
			method:   "FloatEvaluation",
			variable: Variable{},
			expected: openfeature.FloatResolutionDetail{
				Value: 1.23,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("Variable result is nil, but not defaulted"),
				},
			},
		},
		{
			name:     "FloatEvaluation unexpected type",
			method:   "FloatEvaluation",
			variable: Variable{BaseVariable: BaseVariable{Value: "not a float64"}},
			expected: openfeature.FloatResolutionDetail{
				Value: 1.23,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewTypeMismatchResolutionError("Unexpected type in float variable result: string"),
				},
			},
		},
		{
			name:        "FloatEvaluation error",
			method:      "FloatEvaluation",
			errorResult: fmt.Errorf("an unexpected error!"),
			expected: openfeature.FloatResolutionDetail{
				Value: 1.23,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("an unexpected error!"),
				},
			},
		},
		{
			name:     "IntEvaluation default",
			method:   "IntEvaluation",
			variable: Variable{IsDefaulted: true},
			expected: openfeature.IntResolutionDetail{
				Value: 123,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason: openfeature.DefaultReason,
				},
			},
		},
		{
			name:     "IntEvaluation nil without default",
			method:   "IntEvaluation",
			variable: Variable{},
			expected: openfeature.IntResolutionDetail{
				Value: 123,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("Variable result is nil, but not defaulted"),
				},
			},
		},
		{
			name:     "IntEvaluation unexpected type",
			method:   "IntEvaluation",
			variable: Variable{BaseVariable: BaseVariable{Value: "not a int64"}},
			expected: openfeature.IntResolutionDetail{
				Value: 123,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewTypeMismatchResolutionError("Unexpected type in integer variable result: string"),
				},
			},
		},
		{
			name:        "IntEvaluation error",
			method:      "IntEvaluation",
			errorResult: fmt.Errorf("an unexpected error!"),
			expected: openfeature.IntResolutionDetail{
				Value: 123,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("an unexpected error!"),
				},
			},
		},
		{
			name:     "ObjectEvaluation default",
			method:   "ObjectEvaluation",
			variable: Variable{IsDefaulted: true},
			expected: openfeature.InterfaceResolutionDetail{
				Value: map[string]bool{"default": true},
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason: openfeature.DefaultReason,
				},
			},
		},
		{
			name:     "ObjectEvaluation nil without default",
			method:   "ObjectEvaluation",
			variable: Variable{},
			expected: openfeature.InterfaceResolutionDetail{
				Value: map[string]bool{"default": true},
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("Variable result is nil, but not defaulted"),
				},
			},
		},
		{
			name:        "ObjectEvaluation error",
			method:      "ObjectEvaluation",
			errorResult: fmt.Errorf("an unexpected error!"),
			expected: openfeature.InterfaceResolutionDetail{
				Value: map[string]bool{"default": true},
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					Reason:          openfeature.ErrorReason,
					ResolutionError: openfeature.NewGeneralResolutionError("an unexpected error!"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a DevCycleProvider with the mock client
			provider := DevCycleProvider{
				Client: StubClient{
					variable: tc.variable,
					err:      tc.errorResult,
				},
			}

			var result any
			switch tc.method {
			case "BooleanEvaluation":
				result = provider.BooleanEvaluation(context.Background(), "example", false, evalCtx)
			case "StringEvaluation":
				result = provider.StringEvaluation(context.Background(), "example", "default", evalCtx)
			case "FloatEvaluation":
				result = provider.FloatEvaluation(context.Background(), "example", float64(1.23), evalCtx)
			case "IntEvaluation":
				result = provider.IntEvaluation(context.Background(), "example", int64(123), evalCtx)
			case "ObjectEvaluation":
				result = provider.ObjectEvaluation(context.Background(), "example", map[string]bool{"default": true}, evalCtx)
			}

			require.Equalf(t, tc.expected, result, tc.name)
		})
	}
}
