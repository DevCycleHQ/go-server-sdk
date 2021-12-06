<<<<<<< HEAD
# Go API client for DevCycle
=======
# DevCycle Go SDK.
>>>>>>> 9ec1a23 (Unify docs format)

Welcome to the the DevCycle Go SDK, initially generated via the [DevCycle Bucketing API](https://docs.devcycle.com/bucketing-api/#tag/devcycle).

## Installation

```bash
go get "github.com/devcyclehq/go-server-sdk"
```
Put the package under your project folder and add the following in import:
```golang
import "github.com/devcyclehq/go-server-sdk"
```

## Getting Started

```golang
import (
    "github.com/devcyclehq/go-server-sdk"
    "context"
)
auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
    Key: "your_server_key_here",
})

client := devcycle.NewDVCClient()
```

## Usage

### User Object
The user object is required for all methods. The only required field in the user object is UserId

See the UserData class in `model_user_data.go` for all accepted fields.

```golang
user := devcycle.UserData{UserId: "test"}
```

### Getting All Features
This method will fetch all features for a given user and return them in a map of `key: feature_object`

```golang
features, err := client.DevcycleApi.AllFeatures(auth, user)
```

### Grabbing Variables
To get values from your Variables, the `value` field inside the variable object can be accessed.

This method will fetch all variables for a given user and return them in a map of `key: variable_object`

```golang
variables, err := client.DevcycleApi.AllVariables(auth, user)
```

### Grabbing Variable By Key

This method will fetch a specific variable by key for a given user. It will return the variable 
object from the server unless an error occurs or the server has no response. In that case it will return
a variable object with the value set to whatever was passed in as the `defaultValue` parameter, 
and the `IsDefaulted` field boolean on the variable will be true.

To get values from your Variables, the `value` field inside the variable object can be accessed.

```golang
variable, err := client.DevcycleApi.Variable(auth, user, "variable-key", "default_value")
```

### Track Event
To POST custom event for a user, pass in the user and event object.

```golang
event := devcycle.Event{
    Type_: "customEvent",
    Target: "somevariable.key"}

response, err := client.DevcycleApi.Track(auth, user, event)
```
