# DevCycle Go Server SDK.

This SDK supports both cloud bucketing (requests outbound to https://bucketing-api.devcycle.com) as well as local bucketing (requests to a local bucketing engine self-contained in this SDK).

## Installation

```bash
go get "github.com/devcyclehq/go-server-sdk"
```

```golang
package main
import "github.com/devcyclehq/go-server-sdk"
```

## Getting Started

```golang
    environmentKey := os.Getenv("DVC_SERVER_KEY")
	user := devcycle.DVCUser{UserId: "test"}
	auth := context.WithValue(context.Background(), devcycle.ContextAPIKey, devcycle.APIKey{
		Key: environmentKey,
	})

	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}

	lb, err := devcycle.InitializeLocalBucketing(environmentKey, &dvcOptions)
	if err != nil {
		log.Fatal(err)
	}
	client, _ := devcycle.NewDVCClient(environmentKey, &dvcOptions, lb)
```

## Usage

To find usage documentation, visit our [docs](https://docs.devcycle.com/docs/sdk/server-side-sdks/go#usage).