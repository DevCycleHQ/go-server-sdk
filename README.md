# DevCycle Go Server SDK.

This SDK supports both cloud bucketing (requests outbound to https://bucketing-api.devcycle.com) as well as local bucketing (requests to a local bucketing engine self-contained in this SDK).

## Installation

```bash
go get "github.com/devcyclehq/go-server-sdk/v2"
```

```golang
package main
import "github.com/devcyclehq/go-server-sdk/v2"
```

## Getting Started

```golang
    sdkKey := os.Getenv("DVC_SERVER_KEY")
	user := devcycle.DVCUser{UserId: "test"}

	dvcOptions := devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}

	client, _ := devcycle.NewDVCClient(sdkKey, &dvcOptions)
```

## Usage

To find usage documentation, visit our [docs](https://docs.devcycle.com/docs/sdk/server-side-sdks/go#usage).

## Testing

This SDK is supported by our [test harness](https://github.com/DevCycleHQ/test-harness), a test suite shared between all DevCycle SDKs for consistency.
