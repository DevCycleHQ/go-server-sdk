package devcycle

import "github.com/devcyclehq/go-server-sdk/v2/api"

// EvalHook represents a hook that can be executed during variable evaluation
type EvalHook struct {
	// Before is called before variable evaluation
	Before func(context *HookContext) error
	// After is called after variable evaluation (only if Before didn't error)
	After func(context *HookContext, variable *api.Variable, metadata *VariableMetadata) error
	// OnFinally is called after variable evaluation regardless of errors
	OnFinally func(context *HookContext, variable *api.Variable, metadata *VariableMetadata) error
	// Error is called when an error occurs during evaluation
	Error func(context *HookContext, evalError error) error
}

// NewEvalHook creates a new EvalHook with the provided functions
func NewEvalHook(before func(context *HookContext) error, after func(context *HookContext, variable *api.Variable, metadata *VariableMetadata) error, onFinally func(context *HookContext, variable *api.Variable, metadata *VariableMetadata) error, error func(context *HookContext, evalError error) error) *EvalHook {
	return &EvalHook{
		Before:    before,
		After:     after,
		OnFinally: onFinally,
		Error:     error,
	}
}
