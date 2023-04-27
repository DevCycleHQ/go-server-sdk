package devcycle

import (
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"reflect"
	"testing"
)

func Test_createUserFromEvaluationContext_NoUserID(t *testing.T) {
	_, err := createUserFromEvaluationContext(openfeature.FlattenedContext{})
	if err == nil {
		t.Fatal("Expected error when userId is not provided")
	}
}

func Test_createUserFromEvaluationContext_SimpleUser(t *testing.T) {
	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234"})
	if err != nil {
		t.Fatal(err)
	}
	if user.UserId != "1234" {
		t.Errorf("Expected userId to be '1234', but got '%s'", user.UserId)
	}

	user, err = createUserFromEvaluationContext(openfeature.FlattenedContext{"targetingKey": "1234"})
	if err != nil {
		t.Fatal(err)
	}
	if user.UserId != "1234" {
		t.Errorf("Expected userId to be '1234' when sourced from targetingKey, but got '%s'", user.UserId)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if user.UserId != ctx["userId"] {
		t.Errorf("Expected userId to be '%s', but got '%s'", ctx["userId"], user.UserId)
	}
	if user.Email != ctx["email"] {
		t.Errorf("Expected email to be '%s', but got '%s'", ctx["email"], user.Email)
	}

	if user.Name != ctx["name"] {
		t.Errorf("Expected name to be '%s', but got '%s'", ctx["name"], user.Name)
	}

	if user.Language != ctx["language"] {
		t.Errorf("Expected language to be '%s', but got '%s'", ctx["language"], user.Language)
	}

	if user.Country != ctx["country"] {
		t.Errorf("Expected country to be '%s', but got '%s'", ctx["country"], user.Country)
	}

	if user.AppVersion != ctx["appVersion"] {
		t.Errorf("Expected appVersion to be '%s', but got '%s'", ctx["appVersion"], user.AppVersion)
	}

	if user.AppBuild != ctx["appBuild"] {
		t.Errorf("Expected appBuild to be '%s', but got '%s'", ctx["appBuild"], user.AppBuild)
	}

	if user.DeviceModel != ctx["deviceModel"] {
		t.Errorf("Expected deviceModel to be '%s', but got '%s'", ctx["deviceModel"], user.DeviceModel)
	}

	if user.CustomData != nil {
		t.Errorf("Expected customData to be nil, but got '%s'", user.CustomData)
	}

	if user.PrivateCustomData != nil {
		t.Errorf("Expected privateCustomData to be nil, but got '%s'", user.PrivateCustomData)
	}

}

func Test_createUserFromEvaluationContext_InvalidDataType(t *testing.T) {
	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234", "email": 1234})
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "" {
		t.Errorf("Expected email to be empty due to bad data, but got '%s'", user.Email)
	}
}

func Test_createUserFromEvaluationContext_CustomData(t *testing.T) {
	testCustomData := map[string]interface{}{"key1": "strVal", "key2": float64(1234), "key3": true}
	testPrivateData := map[string]interface{}{"key1": "otherVal", "key2": float64(9999), "key3": false}
	user, err := createUserFromEvaluationContext(openfeature.FlattenedContext{"userId": "1234", "customData": testCustomData, "privateCustomData": testPrivateData})
	if err != nil {
		t.Fatal(err)
	}
	if user.CustomData == nil {
		t.Errorf("Expected email to be empty due to bad data, but got '%s'", user.Email)
	}
	if !reflect.DeepEqual(user.CustomData, testCustomData) {
		t.Errorf("Expected user custom data to be '%s', but got '%s'", testCustomData, user.CustomData)
	}
	if user.PrivateCustomData == nil {
		t.Errorf("Expected customData to be set properly but it was nil")
	}
	if !reflect.DeepEqual(user.PrivateCustomData, testPrivateData) {
		t.Errorf("Expected user private custom data to be '%s', but got '%s'", testPrivateData, user.PrivateCustomData)
	}
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
		if !reflect.DeepEqual(customData["key"], testCase.expectedVal) {
			t.Errorf("%s Test: Expected '%v', but got '%v'", testCase.testName, testCase.expectedVal, customData["key"])
		}
	}

	// Test 8 - Nil value
	customData := make(map[string]interface{})
	setCustomDataValue(customData, "nilTest", nil)
	if len(customData) != 0 {
		t.Errorf("Nil value should not be set into custom data, but got '%v'", customData)
	}
}
