package main

import (
	"fmt"
	"log"
	"os"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
	"github.com/devcyclehq/go-server-sdk/v2/api"
)

func main() {
	sdkKey := os.Getenv("DEVCYCLE_SERVER_SDK_KEY")

	// Create hooks
	beforeHook := func(context *devcycle.HookContext) error {
		fmt.Printf("Before hook: Evaluating variable '%s' for user '%s'\n", context.Key, context.User.UserId)
		return nil
	}

	afterHook := func(context *devcycle.HookContext, variable *api.Variable) error {
		fmt.Printf("After hook: Variable '%s' evaluated to %v (defaulted: %t)\n", 
			context.Key, context.VariableDetails.Value, context.VariableDetails.IsDefaulted)
		return nil
	}

	onFinallyHook := func(context *devcycle.HookContext, variable *api.Variable) error {
		fmt.Printf("OnFinally hook: Completed evaluation of variable '%s'\n", context.Key)
		return nil
	}

	errorHook := func(context *devcycle.HookContext, evalError error) error {
		fmt.Printf("Error hook: Error occurred during evaluation of variable '%s'\n", context.Key)
		return nil
	}

	// Create an evaluation hook
	evalHook := devcycle.NewEvalHook(beforeHook, afterHook, onFinallyHook, errorHook)

	// Create client options with hooks
	dvcOptions := devcycle.Options{
		EventFlushIntervalMS:    0,
		ConfigPollingIntervalMS: 10 * time.Second,
		RequestTimeout:          10 * time.Second,
		EvalHooks:              []*devcycle.EvalHook{evalHook},
	}

	client, err := devcycle.NewClient(sdkKey, &dvcOptions)
	if err != nil {
		log.Fatalf("Error initializing client: %v", err)
	}

	// Wait for client to initialize
	time.Sleep(10 * time.Second)

	user := devcycle.User{UserId: "test", CustomData: map[string]interface{}{"a0_organization": "org_tPyJN5dvNNirKar7"}}
	variableKey := "enable-dark-mode"

	fmt.Println("=== Testing Variable Evaluation with Hooks ===")
	
	// Test variable evaluation
	variable, err := client.Variable(user, variableKey, false)
	if err != nil {
		log.Printf("Error getting variable %v: %v", variableKey, err)
	} else {
		fmt.Printf("Final result: variable %v: value=%v (%v) defaulted=%t\n", 
			variable.Key, variable.Value, variable.Type_, variable.IsDefaulted)
	}

	// Test with a non-existent variable to see error handling
	fmt.Println("\n=== Testing with Non-existent Variable ===")
	missingVariable, err := client.Variable(user, variableKey+"-does-not-exist", "DEFAULT")
	if err != nil {
		log.Printf("Error getting missing variable: %v", err)
	} else {
		fmt.Printf("Missing variable result: variable %v: value=%v (%v) defaulted=%t\n", 
			missingVariable.Key, missingVariable.Value, missingVariable.Type_, missingVariable.IsDefaulted)
	}

	client.Close()
} 