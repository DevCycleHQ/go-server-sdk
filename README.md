# DevCycle Go Server SDK.

Welcome to the the DevCycle Go SDK, initially generated via the [DevCycle Bucketing API](https://docs.devcycle.com/bucketing-api/#tag/devcycle).

## Installation

```bash
go get "github.com/devcyclehq/go-server-sdk"
```
Put the package under your project folder and add the following in import:
```golang
package main
import "github.com/devcyclehq/go-server-sdk"
```

## Getting Started

```golang
package main 
import (
    "github.com/devcyclehq/go-server-sdk"
    "context"
)
auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
    Key: "your_server_key_here",
})
dvcOptions := devcycle.DVCOptions{EnableEdgeDB: false}

client := devcycle.NewDVCClient()
client.SetOptions(dvcOptions)
```

## Usage

To find usage documentation, visit our [docs](https://docs.devcycle.com/docs/sdk/server-side-sdks/go#usage).