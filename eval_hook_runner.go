package devcycle

import (
	"fmt"

	"github.com/devcyclehq/go-server-sdk/v2/api"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

// BeforeHookError represents an error that occurred during a before hook
type BeforeHookError struct {
	HookIndex int
	Err       error
}

func (e *BeforeHookError) Error() string {
	return fmt.Sprintf("before hook %d failed: %v", e.HookIndex, e.Err)
}

func (e *BeforeHookError) Unwrap() error {
	return e.Err
}

// AfterHookError represents an error that occurred during an after hook
type AfterHookError struct {
	HookIndex int
	Err       error
}

func (e *AfterHookError) Error() string {
	return fmt.Sprintf("after hook %d failed: %v", e.HookIndex, e.Err)
}

func (e *AfterHookError) Unwrap() error {
	return e.Err
}

// EvalHookRunner manages and executes evaluation hooks
type EvalHookRunner struct {
	hooks []*EvalHook
}

// NewEvalHookRunner creates a new EvalHookRunner with the provided hooks
func NewEvalHookRunner(hooks []*EvalHook) *EvalHookRunner {
	return &EvalHookRunner{
		hooks: hooks,
	}
}

// RunBeforeHooks runs all before hooks in order
func (r *EvalHookRunner) RunBeforeHooks(hooks []*EvalHook, context *HookContext) error {
	if context == nil {
		return nil
	}
	for i, hook := range hooks {
		if hook.Before != nil {
			if err := hook.Before(context); err != nil {
				util.Errorf("Before hook %d failed: %v", i, err)
				return &BeforeHookError{HookIndex: i, Err: err}
			}
		}
	}
	return nil
}

// RunAfterHooks runs all after hooks in reverse order
func (r *EvalHookRunner) RunAfterHooks(hooks []*EvalHook, context *HookContext, variable api.Variable) error {
	if context == nil {
		return nil
	}
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if hook.After != nil {
			if err := hook.After(context, &variable); err != nil {
				util.Errorf("After hook %d failed: %v", i, err)
				return &AfterHookError{HookIndex: i, Err: err}
			}
		}
	}
	return nil
}

// RunOnFinallyHooks runs all onFinally hooks in reverse order
func (r *EvalHookRunner) RunOnFinallyHooks(hooks []*EvalHook, context *HookContext, variable api.Variable) {
	if context == nil {
		return
	}
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if hook.OnFinally != nil {
			if err := hook.OnFinally(context, &variable); err != nil {
				util.Errorf("OnFinally hook %d failed: %v", i, err)
			}
		}
	}
}

// RunErrorHooks runs all error hooks in reverse order
func (r *EvalHookRunner) RunErrorHooks(hooks []*EvalHook, context *HookContext, evalError error) {
	if context == nil {
		return
	}
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if hook.Error != nil {
			if err := hook.Error(context, evalError); err != nil {
				util.Errorf("Error hook %d failed: %v", i, err)
			}
		}
	}
}

func (r *EvalHookRunner) AddHook(hook *EvalHook) {
	r.hooks = append(r.hooks, hook)
}

func (r *EvalHookRunner) ClearHooks() {
	r.hooks = []*EvalHook{}
}
