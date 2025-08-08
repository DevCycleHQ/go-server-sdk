package devcycle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestConfigMetadata_ExtractionAndStorage(t *testing.T) {
	// Parse the existing test config and modify project/environment
	var configMap map[string]interface{}
	err := json.Unmarshal([]byte(test_config), &configMap)
	require.NoError(t, err)

	// Modify project and environment for this test
	configMap["project"].(map[string]interface{})["_id"] = "project-123"
	configMap["project"].(map[string]interface{})["key"] = "my-project"
	configMap["environment"].(map[string]interface{})["_id"] = "env-456"
	configMap["environment"].(map[string]interface{})["key"] = "development"

	modifiedConfig, _ := json.Marshal(configMap)
	sdkKey := generateTestSDKKey()

	// Use httpCustomConfigMock with custom headers
	headers := map[string]string{
		"ETag":          "test-etag-123",
		"Last-Modified": "Wed, 21 Oct 2015 07:28:00 GMT",
		"Cf-Ray":        "test-ray-123",
	}
	httpCustomConfigMock(sdkKey, 200, string(modifiedConfig), false, headers)

	// Mock events endpoint
	httpEventsApiMock()

	// Create client with local bucketing
	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
	}

	client, err := NewClient(sdkKey, options)
	require.NoError(t, err)
	defer client.Close()

	// Wait for config to load
	time.Sleep(time.Millisecond * 500)

	// Test that metadata is available
	metadata, err := client.GetMetadata()
	require.NoError(t, err, "Expected metadata to be available")

	// Test ETag and LastModified
	// Test Project metadata
	require.NotNil(t, metadata.Project, "Expected project metadata to be available")
	require.Equal(t, "project-123", metadata.Project.Id)
	require.Equal(t, "my-project", metadata.Project.Key)

	// Test Environment metadata
	require.NotNil(t, metadata.Environment, "Expected environment metadata to be available")
	require.Equal(t, "env-456", metadata.Environment.Id)
	require.Equal(t, "development", metadata.Environment.Key)
}

func TestConfigMetadata_CloudSDKReturnsNil(t *testing.T) {
	// Mock bucketing API for cloud SDK
	httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test-variable",
		func(req *http.Request) (*http.Response, error) {
			mockVariable := Variable{
				BaseVariable: BaseVariable{
					Key:   "test-var",
					Value: "test-value",
					Type_: "String",
				},
				DefaultValue: "default",
				IsDefaulted:  false,
			}
			resp, _ := httpmock.NewJsonResponse(200, mockVariable)
			return resp, nil
		})

	// Mock events endpoint
	httpEventsApiMock()

	// Create client with cloud bucketing
	options := &Options{
		EnableCloudBucketing:         true,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
	}

	sdkKey, _ := httpConfigMock(200)

	client, err := NewClient(sdkKey, options)
	require.NoError(t, err)
	defer client.Close()

	// Test that metadata returns error for cloud SDK
	_, err = client.GetMetadata()
	require.Error(t, err, "Expected error for cloud SDK")
	require.Contains(t, err.Error(), "cloud SDK", "Expected cloud SDK error message")
}

func TestConfigMetadata_AvailableInAllHooks(t *testing.T) {
	// Parse the existing test config and modify project/environment
	var configMap map[string]interface{}
	err := json.Unmarshal([]byte(test_config), &configMap)
	require.NoError(t, err)

	// Modify project and environment for this test
	configMap["project"].(map[string]interface{})["_id"] = "hook-project-123"
	configMap["project"].(map[string]interface{})["key"] = "hook-project"
	configMap["environment"].(map[string]interface{})["_id"] = "hook-env-456"
	configMap["environment"].(map[string]interface{})["key"] = "production"

	modifiedConfig, _ := json.Marshal(configMap)
	sdkKey := generateTestSDKKey()

	// Use httpCustomConfigMock with custom headers
	headers := map[string]string{
		"ETag":          "hook-etag-456",
		"Last-Modified": "Thu, 22 Oct 2015 08:30:00 GMT",
		"Cf-Ray":        "hook-ray-456",
	}
	httpCustomConfigMock(sdkKey, 200, string(modifiedConfig), false, headers)

	// Mock events endpoint
	httpEventsApiMock()

	// Track hook calls and metadata
	var beforeMetadata, afterMetadata, finallyMetadata ConfigMetadata
	var hookCallCount int

	// Create hooks that capture metadata
	beforeHook := func(context *HookContext) error {
		hookCallCount++
		beforeMetadata = context.Metadata
		return nil
	}

	afterHook := func(context *HookContext, variable *api.Variable, metadata *EvaluationMetadata) error {
		afterMetadata = context.Metadata
		return nil
	}

	finallyHook := func(context *HookContext, variable *api.Variable, metadata *EvaluationMetadata) error {
		finallyMetadata = context.Metadata
		return nil
	}

	errorHook := func(context *HookContext, evalError error) error {
		return nil
	}

	evalHook := NewEvalHook(beforeHook, afterHook, finallyHook, errorHook)

	// Create client with hooks
	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		EvalHooks:                    []*EvalHook{evalHook},
	}

	client, err := NewClient(sdkKey, options)
	require.NoError(t, err)
	defer client.Close()

	// Wait for config to load
	time.Sleep(time.Millisecond * 500)

	// Test user
	user := User{
		UserId: "test-user",
	}

	// Call Variable to trigger hooks
	_, err = client.Variable(user, "test-variable", "default-value")
	require.NoError(t, err)

	// Verify hooks were called
	require.Equal(t, 1, hookCallCount, "Expected before hook to be called once")

	// Test metadata in before hook
	require.NotNil(t, beforeMetadata, "Expected metadata in before hook")
	validateHookMetadata(t, beforeMetadata, "before hook")

	// Test metadata in after hook
	require.NotNil(t, afterMetadata, "Expected metadata in after hook")
	validateHookMetadata(t, afterMetadata, "after hook")

	// Test metadata in finally hook
	require.NotNil(t, finallyMetadata, "Expected metadata in finally hook")
	validateHookMetadata(t, finallyMetadata, "finally hook")
}

func TestConfigMetadata_AvailableInErrorHook(t *testing.T) {
	// Parse the existing test config and modify project/environment
	var configMap map[string]interface{}
	err := json.Unmarshal([]byte(test_config), &configMap)
	require.NoError(t, err)

	// Modify project and environment for this test
	configMap["project"].(map[string]interface{})["_id"] = "error-project-123"
	configMap["project"].(map[string]interface{})["key"] = "error-project"
	configMap["environment"].(map[string]interface{})["_id"] = "error-env-456"
	configMap["environment"].(map[string]interface{})["key"] = "staging"

	modifiedConfig, _ := json.Marshal(configMap)
	sdkKey := generateTestSDKKey()

	// Use httpCustomConfigMock with custom headers
	headers := map[string]string{
		"ETag":          "error-etag-789",
		"Last-Modified": "Fri, 23 Oct 2015 09:45:00 GMT",
		"Cf-Ray":        "error-ray-789",
	}
	httpCustomConfigMock(sdkKey, 200, string(modifiedConfig), false, headers)

	// Mock events endpoint
	httpEventsApiMock()

	// Track error hook metadata
	var errorMetadata ConfigMetadata
	var errorHookCalled bool

	// Create hooks that capture metadata - make before hook fail
	beforeHook := func(context *HookContext) error {
		return &BeforeHookError{HookIndex: 0, Err: fmt.Errorf("simulated before hook error")}
	}

	afterHook := func(context *HookContext, variable *api.Variable, metadata *EvaluationMetadata) error {
		return nil
	}

	finallyHook := func(context *HookContext, variable *api.Variable, metadata *EvaluationMetadata) error {
		return nil
	}

	errorHook := func(context *HookContext, evalError error) error {
		errorHookCalled = true
		errorMetadata = context.Metadata
		return nil
	}

	evalHook := NewEvalHook(beforeHook, afterHook, finallyHook, errorHook)

	// Create client with hooks
	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
		EvalHooks:                    []*EvalHook{evalHook},
	}

	client, err := NewClient(sdkKey, options)
	require.NoError(t, err)
	defer client.Close()

	// Wait for config to load
	time.Sleep(time.Millisecond * 500)

	// Test user
	user := User{
		UserId: "test-user",
	}

	// Call Variable to trigger hooks (before hook will fail)
	_, _ = client.Variable(user, "test-variable", "default-value")

	// Verify error hook was called
	require.True(t, errorHookCalled, "Expected error hook to be called")

	// Test metadata in error hook
	require.NotNil(t, errorMetadata, "Expected metadata in error hook")

	require.NotNil(t, errorMetadata.Project, "Expected project metadata in error hook")
	require.Equal(t, "error-project-123", errorMetadata.Project.Id)

	require.NotNil(t, errorMetadata.Environment, "Expected environment metadata in error hook")
	require.Equal(t, "staging", errorMetadata.Environment.Key)
}

func TestConfigMetadata_NullSafetyDuringInitialization(t *testing.T) {
	sdkKey, _ := httpConfigMock(500)

	// Mock events endpoint
	httpEventsApiMock()

	// Create client that will fail to load config but client creation should succeed
	options := &Options{
		EnableCloudBucketing:         false,
		DisableAutomaticEventLogging: true,
		DisableCustomEventLogging:    true,
	}

	client, err := NewClient(sdkKey, options)
	require.NoError(t, err)
	defer client.Close()

	// Wait a bit for the failed config load attempt
	time.Sleep(time.Millisecond * 100)

	// Test that metadata returns error during initialization failure
	_, err = client.GetMetadata()
	require.Error(t, err, "Expected error during initialization failure")
	require.Contains(t, err.Error(), "config not loaded", "Expected config not loaded error")

	// Test that hooks still work with nil metadata
	var hookMetadata ConfigMetadata
	beforeHook := func(context *HookContext) error {
		hookMetadata = context.Metadata
		return nil
	}

	evalHook := NewEvalHook(beforeHook, nil, nil, nil)
	client.evalHookRunner.AddHook(evalHook)

	user := User{UserId: "test-user"}
	_, _ = client.Variable(user, "test-variable", "default-value")

	// Verify hook received empty metadata gracefully
	require.Equal(t, ConfigMetadata{}, hookMetadata, "Expected empty metadata in hook during initialization")
}

// Helper function to validate metadata in hooks
func validateHookMetadata(t *testing.T, metadata ConfigMetadata, hookType string) {
	require.NotNil(t, metadata.Project, "Project metadata in %s", hookType)
	require.Equal(t, "hook-project-123", metadata.Project.Id, "Project ID in %s", hookType)
	require.Equal(t, "hook-project", metadata.Project.Key, "Project key in %s", hookType)

	require.NotNil(t, metadata.Environment, "Environment metadata in %s", hookType)
	require.Equal(t, "hook-env-456", metadata.Environment.Id, "Environment ID in %s", hookType)
	require.Equal(t, "production", metadata.Environment.Key, "Environment key in %s", hookType)
}
