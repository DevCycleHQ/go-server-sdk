package devcycle

import "github.com/devcyclehq/go-server-sdk/v2/api"

// HookContext stores the context information passed to hooks during variable evaluation
type HookContext struct {
	// User is the user for whom the variable is being evaluated
	User User
	// Key is the variable key being evaluated
	Key string
	// DefaultValue is the default value provided for the variable
	DefaultValue interface{}
	// VariableDetails is the variable that gets evaluated
	VariableDetails api.Variable
}
