package devcycle

import (
	"errors"
	"testing"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvalHookRunner(t *testing.T) {
	t.Run("RunBeforeHooks - success", func(t *testing.T) {
		hook1 := NewEvalHook(
			func(context *HookContext) error { return nil },
			nil, nil, nil,
		)
		hook2 := NewEvalHook(
			func(context *HookContext) error { return nil },
			nil, nil, nil,
		)

		runner := NewEvalHookRunner([]*EvalHook{hook1, hook2})
		context := &HookContext{Key: "test-key"}

		err := runner.RunBeforeHooks([]*EvalHook{hook1, hook2}, context)
		assert.NoError(t, err)
	})

	t.Run("RunBeforeHooks - error", func(t *testing.T) {
		expectedError := errors.New("before hook error")
		hook := NewEvalHook(
			func(context *HookContext) error { return expectedError },
			nil, nil, nil,
		)

		runner := NewEvalHookRunner([]*EvalHook{hook})
		context := &HookContext{Key: "test-key"}

		err := runner.RunBeforeHooks([]*EvalHook{hook}, context)
		require.Error(t, err)

		beforeHookError, ok := err.(*BeforeHookError)
		assert.True(t, ok)
		assert.Equal(t, 0, beforeHookError.HookIndex)
		assert.Equal(t, expectedError, beforeHookError.Err)
	})

	t.Run("RunAfterHooks - success", func(t *testing.T) {
		hook1 := NewEvalHook(
			nil,
			func(context *HookContext, variable *api.Variable) error { return nil },
			nil, nil,
		)
		hook2 := NewEvalHook(
			nil,
			func(context *HookContext, variable *api.Variable) error { return nil },
			nil, nil,
		)

		runner := NewEvalHookRunner([]*EvalHook{hook1, hook2})
		context := &HookContext{Key: "test-key"}

		err := runner.RunAfterHooks([]*EvalHook{hook1, hook2}, context, api.Variable{})
		assert.NoError(t, err)
	})

	t.Run("RunAfterHooks - error", func(t *testing.T) {
		expectedError := errors.New("after hook error")
		hook := NewEvalHook(
			nil,
			func(context *HookContext, variable *api.Variable) error { return expectedError },
			nil, nil,
		)

		variable := api.Variable{
			BaseVariable: api.BaseVariable{
				Key:   "test-key",
				Type_: "String",
				Value: "test-value",
			},
			DefaultValue: "default",
			IsDefaulted:  false,
		}

		runner := NewEvalHookRunner([]*EvalHook{hook})
		context := &HookContext{Key: "test-key"}

		err := runner.RunAfterHooks([]*EvalHook{hook}, context, variable)
		require.Error(t, err)

		afterHookError, ok := err.(*AfterHookError)
		assert.True(t, ok)
		assert.Equal(t, 0, afterHookError.HookIndex)
		assert.Equal(t, expectedError, afterHookError.Err)
	})

	t.Run("RunOnFinallyHooks", func(t *testing.T) {
		called := false
		hook := NewEvalHook(
			nil, nil,
			func(context *HookContext, variable *api.Variable) error {
				called = true
				return nil
			}, nil,
		)

		runner := NewEvalHookRunner([]*EvalHook{hook})
		context := &HookContext{Key: "test-key"}

		runner.RunOnFinallyHooks([]*EvalHook{hook}, context, api.Variable{})
		assert.True(t, called)
	})

	t.Run("RunOnErrorHooks", func(t *testing.T) {
		called := false
		hook := NewEvalHook(
			nil, nil, nil,
			func(context *HookContext, evalError error) error {
				called = true
				return nil
			},
		)

		runner := NewEvalHookRunner([]*EvalHook{hook})
		context := &HookContext{Key: "test-key"}

		runner.RunErrorHooks([]*EvalHook{hook}, context, errors.New("test error"))
		assert.True(t, called)
	})

	t.Run("Hook execution order", func(t *testing.T) {
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

		runner := NewEvalHookRunner([]*EvalHook{hook1, hook2})
		context := &HookContext{Key: "test-key"}

		// Run before hooks (should be in order: 1, 2)
		err := runner.RunBeforeHooks([]*EvalHook{hook1, hook2}, context)
		assert.NoError(t, err)

		// Run after hooks (should be in reverse order: 3, 4)
		err = runner.RunAfterHooks([]*EvalHook{hook1, hook2}, context, api.Variable{})
		assert.NoError(t, err)

		// Run onFinally hooks (should be in reverse order: 5, 6)
		runner.RunOnFinallyHooks([]*EvalHook{hook1, hook2}, context, api.Variable{})

		// Verify execution order: before hooks in order, after/onFinally hooks in reverse order
		expectedOrder := []int{1, 2, 3, 4, 5, 6}
		assert.Equal(t, expectedOrder, executionOrder)
	})
}

func TestHookContext(t *testing.T) {
	user := User{UserId: "test-user"}
	context := &HookContext{
		User:         user,
		Key:          "test-key",
		DefaultValue: "default",
		VariableDetails: api.Variable{
			BaseVariable: api.BaseVariable{
				Key:   "test-key",
				Type_: "String",
				Value: "test-value",
			},
			DefaultValue: "default",
			IsDefaulted:  false,
		},
	}

	assert.Equal(t, user, context.User)
	assert.Equal(t, "test-key", context.Key)
	assert.Equal(t, "default", context.DefaultValue)
	assert.Equal(t, "test-value", context.VariableDetails.Value)
	assert.False(t, context.VariableDetails.IsDefaulted)
}

func TestNewEvalHook(t *testing.T) {
	beforeCalled := false
	afterCalled := false
	onFinallyCalled := false
	onErrorCalled := false

	before := func(context *HookContext) error {
		beforeCalled = true
		return nil
	}
	after := func(context *HookContext, variable *api.Variable) error {
		afterCalled = true
		return nil
	}
	onFinally := func(context *HookContext, variable *api.Variable) error {
		onFinallyCalled = true
		return nil
	}
	onError := func(context *HookContext, evalError error) error {
		onErrorCalled = true
		return nil
	}

	hook := NewEvalHook(before, after, onFinally, onError)

	assert.NotNil(t, hook.Before)
	assert.NotNil(t, hook.After)
	assert.NotNil(t, hook.OnFinally)
	assert.NotNil(t, hook.Error)

	// Test that the functions work
	context := &HookContext{Key: "test"}

	// Test that the before hook works
	err := hook.Before(context)
	assert.NoError(t, err)
	assert.True(t, beforeCalled)

	// Test that the after hook works
	err = hook.After(context, &api.Variable{})
	assert.NoError(t, err)
	assert.True(t, afterCalled)

	err = hook.OnFinally(context, &api.Variable{})
	assert.NoError(t, err)
	assert.True(t, onFinallyCalled)

	err = hook.Error(context, errors.New("test error"))
	assert.NoError(t, err)
	assert.True(t, onErrorCalled)
}
