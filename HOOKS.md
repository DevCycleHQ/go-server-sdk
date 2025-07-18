# Evaluation Hooks

The DevCycle Go SDK now supports evaluation hooks that allow you to intercept and modify variable evaluation behavior. Hooks provide a way to add custom logic before, after, and during variable evaluation.

## Overview

Evaluation hooks are functions that get called at specific points during variable evaluation:

- **Before hooks**: Called before variable evaluation
- **After hooks**: Called after variable evaluation (only if before hooks succeed)
- **OnFinally hooks**: Called after variable evaluation regardless of errors
- **OnError hooks**: Called when an error occurs during evaluation

## Usage

### Creating Hooks

```go
import "github.com/devcyclehq/go-server-sdk/v2"

// Define your hook functions
beforeHook := func(context *devcycle.HookContext) error {
    fmt.Printf("Before: Evaluating variable '%s' for user '%s'\n",
        context.Key, context.User.UserId)
    return nil
}

afterHook := func(context *devcycle.HookContext) error {
    fmt.Printf("After: Variable '%s' evaluated to %v (defaulted: %t)\n",
        context.Key, context.VariableDetails.Value, context.VariableDetails.IsDefaulted)
    return nil
}

onFinallyHook := func(context *devcycle.HookContext) error {
    fmt.Printf("Finally: Completed evaluation of variable '%s'\n", context.Key)
    return nil
}

onErrorHook := func(context *devcycle.HookContext) error {
    fmt.Printf("Error: Error occurred during evaluation of variable '%s'\n", context.Key)
    return nil
}

// Create an evaluation hook
evalHook := devcycle.NewEvalHook(beforeHook, afterHook, onFinallyHook, onErrorHook)
```

### Configuring Hooks with Client

```go
// Create client options with hooks
options := &devcycle.Options{
    EventFlushIntervalMS:    0,
    ConfigPollingIntervalMS: 10 * time.Second,
    RequestTimeout:          10 * time.Second,
    EvalHooks:              []*devcycle.EvalHook{evalHook},
}

// Initialize client with hooks
client, err := devcycle.NewClient(sdkKey, &options)
if err != nil {
    log.Fatalf("Error initializing client: %v", err)
}

// Use the client normally - hooks will be called automatically
user := devcycle.User{UserId: "test-user"}
variable, err := client.Variable(user, "my-variable", "default")
```

### Hook Context

The `HookContext` struct provides information about the variable evaluation:

```go
type HookContext struct {
    User         User        // The user for whom the variable is being evaluated
    Key          string      // The variable key being evaluated
    DefaultValue interface{} // The default value provided for the variable
    VariableDetails api.Variable // The variable that gets evaluated
}
```

### Hook Execution Order

1. **Before hooks**: Executed in order (hook1, hook2, hook3...)
2. **Variable evaluation**: The actual variable evaluation logic
3. **After hooks**: Executed in reverse order (hook3, hook2, hook1...)
4. **OnFinally hooks**: Executed in reverse order (hook3, hook2, hook1...)
5. **OnError hooks**: Executed in reverse order if an error occurred

### Error Handling

- **BeforeHookError**: Thrown when a before hook fails
- **AfterHookError**: Thrown when an after hook fails
- If a before hook fails, after hooks are not executed
- OnFinally and OnError hooks are always executed

### Multiple Hooks

You can configure multiple hooks:

```go
hook1 := devcycle.NewEvalHook(before1, after1, finally1, error1)
hook2 := devcycle.NewEvalHook(before2, after2, finally2, error2)

options := &devcycle.Options{
    EvalHooks: []*devcycle.EvalHook{hook1, hook2},
}
```

### Example Use Cases

#### Logging and Monitoring

```go
beforeHook := func(context *devcycle.HookContext) error {
    log.Printf("Starting evaluation for variable %s", context.Key)
    return nil
}

afterHook := func(context *devcycle.HookContext) error {
    log.Printf("Completed evaluation for variable %s: %v",
        context.Key, context.VariableDetails.Value)
    return nil
}
```

#### Validation

```go
beforeHook := func(context *devcycle.HookContext) error {
    if context.Key == "" {
        return errors.New("variable key cannot be empty")
    }
    return nil
}
```

#### Transformation

```go
afterHook := func(context *devcycle.HookContext) error {
    // Transform the variable value if needed
    if str, ok := context.VariableDetails.Value.(string); ok {
        context.VariableDetails.Value = strings.ToUpper(str)
    }
    return nil
}
```

#### Error Handling

```go
onErrorHook := func(context *devcycle.HookContext) error {
    // Log errors or send to monitoring service
    log.Printf("Error evaluating variable %s: %v", context.Key, context.VariableDetails.Value)
    return nil
}
```

## Important Notes

- Hooks are optional - the SDK works normally without them
- Hook errors are propagated to the caller
- The functionality remains the same with or without hooks
- Hooks are executed synchronously and can impact performance
- Use hooks judiciously to avoid performance issues in high-throughput scenarios
