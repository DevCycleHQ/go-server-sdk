package devcycle

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

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
	testCustomData := map[string]interface{}{"key1": "strVal", "key2": float64(1234), "key3": true}
	testPrivateData := map[string]interface{}{"key1": "otherVal", "key2": float64(9999), "key3": false}
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

	// Test 8 - Nil value
	customData := make(map[string]interface{})
	setCustomDataValue(customData, "nilTest", nil)
	require.Len(t, customData, 0, "Nil value should not be set into custom data")
}

func getProviderForConfig(t *testing.T, config string) openfeature.FeatureProvider {
	t.Helper()

	httpCustomConfigMock(test_environmentKey, 200, config)

	client, err := NewClient(test_environmentKey, &Options{})
	require.NoError(t, err)

	return DevCycleProvider{Client: client}
}

func Test_BooleanEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "test", false, evalCtx)

	require.False(t, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("userId or targetingKey must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_BooleanEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.StringEvaluation(context.Background(), "test-string-variable", "default", evalCtx)

	require.Equal(t, "default", resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("userId or targetingKey must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_StringEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.FloatEvaluation(context.Background(), "test", 1.23, evalCtx)

	require.Equal(t, 1.23, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("userId or targetingKey must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_FloatEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.FloatEvaluation(context.Background(), "test-number-variable", 1.23, evalCtx)

	require.Equal(t, float64(1), resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_FloatEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.IntEvaluation(context.Background(), "test", int64(123), evalCtx)

	require.Equal(t, int64(123), resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("userId or targetingKey must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_IntEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.IntEvaluation(context.Background(), "test-number-variable", 123, evalCtx)

	require.Equal(t, int64(1), resolutionDetail.Value)
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason)
}

func Test_IntEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	defaultValue := map[string]any{
		"default": "value",
	}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "test", defaultValue, evalCtx)

	require.Equal(t, defaultValue, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewInvalidContextResolutionError("userId or targetingKey must be provided"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_ObjectEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
	provider := getProviderForConfig(t, test_config)

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	defaultValue := []string{"default"}
	resolutionDetail := provider.ObjectEvaluation(context.Background(), "test-json-variable", defaultValue, evalCtx)

	require.Equal(t, defaultValue, resolutionDetail.Value)
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason)
	require.Equal(t, openfeature.NewGeneralResolutionError("the default value for variable test-json-variable is not of type Boolean, Number, String, or JSON"), resolutionDetail.ProviderResolutionDetail.ResolutionError)
}

func Test_ObjectEvaluation_TargetMatchInvalidType(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	provider := getProviderForConfig(t, test_config)

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
