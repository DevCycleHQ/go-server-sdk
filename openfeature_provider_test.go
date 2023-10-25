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
	require.NotNil(t, user.PrivateCustomData, "Expected customData to be set properly but it was nil")
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

func Test_BooleanEvaluation_Default(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_config)

	client, err := NewClient(test_environmentKey, &Options{})
	require.NoError(t, err)

	provider := DevCycleProvider{Client: client}

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "unknownFlag", false, evalCtx)

	require.False(t, resolutionDetail.Value, "Expected value to be false")
	require.Equal(t, openfeature.DefaultReason, resolutionDetail.ProviderResolutionDetail.Reason, "Expected reason to be 'DefaultReason'")
}

func Test_BooleanEvaluation_BadUserData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_config)

	client, err := NewClient(test_environmentKey, &Options{})
	require.NoError(t, err)

	provider := DevCycleProvider{Client: client}

	evalCtx := openfeature.FlattenedContext{
		"badUserIDKey": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "test", false, evalCtx)

	require.False(t, resolutionDetail.Value, "Expected value to be false")
	require.Equal(t, openfeature.ErrorReason, resolutionDetail.ProviderResolutionDetail.Reason, "Expected reason to be 'ErrorReason'")
}

func Test_BooleanEvaluation_TargetMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpCustomConfigMock(test_environmentKey, 200, test_config)

	client, err := NewClient(test_environmentKey, &Options{})
	fatalErr(t, err)

	provider := DevCycleProvider{Client: client}

	evalCtx := openfeature.FlattenedContext{
		"userId": "1234",
	}
	resolutionDetail := provider.BooleanEvaluation(context.Background(), "test", false, evalCtx)

	require.True(t, resolutionDetail.Value, "Expected value to be true")
	require.Equal(t, openfeature.TargetingMatchReason, resolutionDetail.ProviderResolutionDetail.Reason, "Expected reason to be 'TargetingMatchReason'")
}
