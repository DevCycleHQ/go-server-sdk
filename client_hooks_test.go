package devcycle

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientWithHooks(t *testing.T) {
	t.Run("Client with hooks - before hook error", func(t *testing.T) {
		beforeHookError := errors.New("before hook failed")
		afterCalled := false
		onFinallyCalled := false
		errorCalled := false
		beforeHook := func(context *HookContext) error {
			return beforeHookError
		}

		afterHook := func(context *HookContext, variable *api.Variable) error {
			afterCalled = true
			t.Error("After hook should not be called when before hook fails")
			return nil
		}

		onFinallyHook := func(context *HookContext, variable *api.Variable) error {
			onFinallyCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "default", variable.Value)
			assert.True(t, variable.IsDefaulted)
			// This should be called even when before hook fails
			return nil
		}

		errorHook := func(context *HookContext, evalError error) error {
			errorCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			// This should be called when before hook fails
			return nil
		}

		evalHook := NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

		sdkKey := generateTestSDKKey()
		httpCustomConfigMock(sdkKey, 200, test_config, false)

		options := &Options{
			EvalHooks: []*EvalHook{evalHook},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should return the default variable when before hook fails
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)

		// Should not return the before hook error to the user
		assert.NoError(t, err)

		// check after hook was not called
		assert.False(t, afterCalled)

		// check onFinally hook was called
		assert.True(t, onFinallyCalled)

		// check error hook was called
		assert.True(t, errorCalled)
	})

	t.Run("Client with hooks - after hook error", func(t *testing.T) {
		afterHookError := errors.New("after hook failed")
		beforeCalled := false
		onFinallyCalled := false
		errorCalled := false

		beforeHook := func(context *HookContext) error {
			beforeCalled = true
			return nil
		}

		afterHook := func(context *HookContext, variable *api.Variable) error {
			return afterHookError
		}

		onFinallyHook := func(context *HookContext, variable *api.Variable) error {
			// This should be called even when after hook fails
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "default", variable.Value)
			assert.True(t, variable.IsDefaulted)
			onFinallyCalled = true
			return nil
		}

		errorHook := func(context *HookContext, evalError error) error {
			// This should be called when after hook fails
			errorCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			return nil
		}

		evalHook := NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

		sdkKey := generateTestSDKKey()
		httpCustomConfigMock(sdkKey, 200, test_config, false)

		options := &Options{
			EvalHooks: []*EvalHook{evalHook},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should return the variable result
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)

		// Should not return the after hook error to the user
		assert.NoError(t, err)

		// check before hook was called
		assert.True(t, beforeCalled)

		// check onFinally hook was called
		assert.True(t, onFinallyCalled)
		assert.True(t, errorCalled)
	})

	t.Run("Client with hooks - successful evaluation", func(t *testing.T) {
		beforeCalled := false
		afterCalled := false
		onFinallyCalled := false
		errorCalled := false

		beforeHook := func(context *HookContext) error {
			beforeCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			return nil
		}

		afterHook := func(context *HookContext, variable *api.Variable) error {
			afterCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			// VariableDetails should be updated with the result
			assert.Equal(t, "test-key", context.VariableDetails.Key)
			assert.Equal(t, "default", context.VariableDetails.Value)
			assert.True(t, context.VariableDetails.IsDefaulted)

			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "default", variable.Value)
			assert.True(t, variable.IsDefaulted)
			return nil
		}

		onFinallyHook := func(context *HookContext, variable *api.Variable) error {
			onFinallyCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "default", variable.Value)
			assert.True(t, variable.IsDefaulted)
			return nil
		}

		errorHook := func(context *HookContext, evalError error) error {
			errorCalled = true
			t.Error("Error hook should not be called for successful evaluation")
			return nil
		}

		evalHook := NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

		sdkKey := generateTestSDKKey()
		httpCustomConfigMock(sdkKey, 200, test_config, false)

		options := &Options{
			EvalHooks: []*EvalHook{evalHook},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should return the variable result
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)
		assert.NoError(t, err)

		// All hooks should be called
		assert.True(t, beforeCalled)
		assert.True(t, afterCalled)
		assert.True(t, onFinallyCalled)
		assert.False(t, errorCalled)
	})

	t.Run("Client without hooks", func(t *testing.T) {
		sdkKey := generateTestSDKKey()
		httpCustomConfigMock(sdkKey, 200, test_config, false)

		options := &Options{}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should work normally without hooks
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)
		assert.NoError(t, err)
	})

	t.Run("Client with multiple hooks", func(t *testing.T) {
		executionOrder := []int{}

		hook1 := NewEvalHook(
			func(context *HookContext) error {
				executionOrder = append(executionOrder, 1)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 4)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 6)
				return nil
			},
			nil,
		)

		hook2 := NewEvalHook(
			func(context *HookContext) error {
				executionOrder = append(executionOrder, 2)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 3)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 5)
				return nil
			},
			nil,
		)

		sdkKey := generateTestSDKKey()
		httpCustomConfigMock(sdkKey, 200, test_config, false)

		options := &Options{
			EvalHooks: []*EvalHook{hook1, hook2},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should work normally
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)
		assert.NoError(t, err)

		// Verify execution order: before hooks in order, after/onFinally hooks in reverse order
		expectedOrder := []int{1, 2, 3, 4, 5, 6}
		assert.Equal(t, expectedOrder, executionOrder)
	})
}

func TestClientWithHooksCloud(t *testing.T) {
	t.Run("Cloud client with hooks - before hook error", func(t *testing.T) {
		beforeHookError := errors.New("before hook failed")
		afterCalled := false
		onFinallyCalled := false
		errorCalled := false
		beforeHook := func(context *HookContext) error {
			return beforeHookError
		}

		afterHook := func(context *HookContext, variable *api.Variable) error {
			afterCalled = true
			t.Error("After hook should not be called when before hook fails")
			return nil
		}

		onFinallyHook := func(context *HookContext, variable *api.Variable) error {
			onFinallyCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "default", variable.Value)
			assert.True(t, variable.IsDefaulted)
			// This should be called even when before hook fails
			return nil
		}

		errorHook := func(context *HookContext, evalError error) error {
			errorCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)	
			// This should be called when before hook fails
			return nil
		}

		evalHook := NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

		sdkKey := generateTestSDKKey()
		// Mock the bucketing API to return a 404 for test-key (variable not found)
		httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test-key",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(404, `{"message": "Variable not found"}`)
				return resp, nil
			},
		)

		options := &Options{
			EnableCloudBucketing: true,
			EvalHooks:            []*EvalHook{evalHook},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should return the default variable when before hook fails
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)

		// Should not return the before hook error to the user
		assert.NoError(t, err)

		// check after hook was not called
		assert.False(t, afterCalled)

		// check onFinally hook was called
		assert.True(t, onFinallyCalled)

		// check error hook was called
		assert.True(t, errorCalled)
	})

	t.Run("Cloud client with hooks - after hook error", func(t *testing.T) {
		afterHookError := errors.New("after hook failed")
		beforeCalled := false
		onFinallyCalled := false
		errorCalled := false

		beforeHook := func(context *HookContext) error {
			beforeCalled = true
			return nil
		}

		afterHook := func(context *HookContext, variable *api.Variable) error {
			return afterHookError
		}

		onFinallyHook := func(context *HookContext, variable *api.Variable) error {
			// This should be called even when after hook fails
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "default", variable.Value)
			assert.True(t, variable.IsDefaulted)
			onFinallyCalled = true
			return nil
		}

		errorHook := func(context *HookContext, evalError error) error {
			// This should be called when after hook fails
			errorCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			return nil
		}

		evalHook := NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

		sdkKey := generateTestSDKKey()
		// Mock the bucketing API to return a 404 for test-key (variable not found)
		httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test-key",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(404, `{"message": "Variable not found"}`)
				return resp, nil
			},
		)

		options := &Options{
			EnableCloudBucketing: true,
			EvalHooks:            []*EvalHook{evalHook},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should return the variable result
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)

		// Should not return the after hook error to the user
		assert.NoError(t, err)

		// check before hook was called
		assert.True(t, beforeCalled)

		// check onFinally hook was called
		assert.True(t, onFinallyCalled)
		assert.True(t, errorCalled)
	})

	t.Run("Cloud client with hooks - successful evaluation", func(t *testing.T) {
		beforeCalled := false
		afterCalled := false
		onFinallyCalled := false
		errorCalled := false

		beforeHook := func(context *HookContext) error {
			beforeCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			return nil
		}

		afterHook := func(context *HookContext, variable *api.Variable) error {
			afterCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			// VariableDetails should be updated with the result
			assert.Equal(t, "test-key", context.VariableDetails.Key)
			assert.Equal(t, "newValue", context.VariableDetails.Value)
			assert.False(t, context.VariableDetails.IsDefaulted)

			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "newValue", variable.Value)
			assert.False(t, variable.IsDefaulted)
			return nil
		}

		onFinallyHook := func(context *HookContext, variable *api.Variable) error {
			onFinallyCalled = true
			assert.Equal(t, "test-key", context.Key)
			assert.Equal(t, "test-user", context.User.UserId)
			assert.Equal(t, "default", context.DefaultValue)
			assert.Equal(t, "test-key", variable.Key)
			assert.Equal(t, "newValue", variable.Value)
			assert.False(t, variable.IsDefaulted)
			return nil
		}

		errorHook := func(context *HookContext, evalError error) error {
			errorCalled = true
			t.Error("Error hook should not be called for successful evaluation")
			return nil
		}

		evalHook := NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

		sdkKey := generateTestSDKKey()
		// Mock the bucketing API to return a 404 for test-key (variable not found)
		httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test-key",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, `{"value": "newValue", "_id": "614ef6ea475129459160721a", "key": "test-key", "type": "String"}`)
				resp.Header.Set("Etag", "TESTING")
				resp.Header.Set("Last-Modified", time.Now().Add(-time.Second*2).Format(time.RFC1123Z))
				return resp, nil
			},
		)

		options := &Options{
			EnableCloudBucketing: true,
			EvalHooks:            []*EvalHook{evalHook},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should return the variable result
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "newValue", variable.Value)
		assert.False(t, variable.IsDefaulted)
		assert.NoError(t, err)

		// All hooks should be called
		assert.True(t, beforeCalled)
		assert.True(t, afterCalled)
		assert.True(t, onFinallyCalled)
		assert.False(t, errorCalled)
	})

	t.Run("Cloud client without hooks", func(t *testing.T) {
		sdkKey := generateTestSDKKey()
		// Mock the bucketing API to return a 404 for test-key (variable not found)
		httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test-key",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(404, `{"message": "Variable not found"}`)
				return resp, nil
			},
		)

		options := &Options{
			EnableCloudBucketing: true,
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should work normally without hooks
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)
		assert.NoError(t, err)
	})

	t.Run("Cloud client with multiple hooks", func(t *testing.T) {
		executionOrder := []int{}

		hook1 := NewEvalHook(
			func(context *HookContext) error {
				executionOrder = append(executionOrder, 1)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 4)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 6)
				return nil
			},
			nil,
		)

		hook2 := NewEvalHook(
			func(context *HookContext) error {
				executionOrder = append(executionOrder, 2)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 3)
				return nil
			},
			func(context *HookContext, variable *api.Variable) error {
				executionOrder = append(executionOrder, 5)
				return nil
			},
			nil,
		)

		sdkKey := generateTestSDKKey()
		// Mock the bucketing API to return a 404 for test-key (variable not found)
		httpmock.RegisterResponder("POST", "https://bucketing-api.devcycle.com/v1/variables/test-key",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(404, `{"message": "Variable not found"}`)
				return resp, nil
			},
		)

		options := &Options{
			EnableCloudBucketing: true,
			EvalHooks:            []*EvalHook{hook1, hook2},
		}

		client, err := NewClient(sdkKey, options)
		require.NoError(t, err)

		// Wait for client to initialize
		time.Sleep(100 * time.Millisecond)

		user := User{UserId: "test-user"}
		variable, err := client.Variable(user, "test-key", "default")

		// Should work normally
		assert.Equal(t, "test-key", variable.Key)
		assert.Equal(t, "default", variable.Value)
		assert.True(t, variable.IsDefaulted)
		assert.NoError(t, err)

		// Verify execution order: before hooks in order, after/onFinally hooks in reverse order
		expectedOrder := []int{1, 2, 3, 4, 5, 6}
		assert.Equal(t, expectedOrder, executionOrder)
	})
}
