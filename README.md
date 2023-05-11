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
	user := devcycle.User{UserId: "test"}

	options := devcycle.Options{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		EventFlushIntervalMS:         0,
		ConfigPollingIntervalMS:      10 * time.Second,
		RequestTimeout:               10 * time.Second,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    false,
	}

	client, _ := devcycle.NewClient(sdkKey, &options)
```

## Usage

To find usage documentation, visit our [docs](https://docs.devcycle.com/docs/sdk/server-side-sdks/go#usage).

## Testing

This SDK is supported by our [test harness](https://github.com/DevCycleHQ/test-harness), a test suite shared between all DevCycle SDKs for consistency.

## Configuration

Configuration of the SDK is done through the `Options` struct.

## Logging

By default, logging is disabled to avoid overhead and noise in your logs. To enable it for debugging the SDK, set the `devcycle_debug_logging` build tag when compiling your project:
```
go build -tags devcycle_debug_logging ...
```

### Cloud Bucketing

The following options are available when you are using the SDK in Cloud Bucketing mode.

| Option | Type          | Description                                                                                                                               | Default |
| --- |---------------|-------------------------------------------------------------------------------------------------------------------------------------------|---------|
| EnableCloudBucketing | bool          | Sets the SDK to Cloud Bucketing mode                                                                                                      | false   |
| EnableEdgeDB | bool          | Turns on EdgeDB support for Cloud Bucketing                                                                                               | false   |
| BucketingAPIURI | string        | The base URI for communicating with the DevCycle Cloud Bucketing service. Can be set if you need to proxy traffic through your own server | https://bucketing-api.devcycle.com        |
| Logger | util.Logger   | Allows you to set a custom logger to manage output from the SDK. The default logger will write to stdout and stderr                       | nil     |

### Local Bucketing

The following options are available when you are using the SDK in Local Bucketing mode.

| Option                       | Type           | Description                                                                                                                                                                                                                     | Default    |
|------------------------------|----------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|
| OnInitializedChannel         | chan bool      | A callback channel to get notified when the SDK is fully initialized and ready to use                                                                                                                                           | nil        |
| EventFlushIntervalMS         | time.Duration  | How frequently events are flushed to the backend. <br>*value must be between 500ms and 60s*                                                                                                                                     | 30000      |
| ConfigPollingIntervalMS      | time.Duration  | How frequently the SDK will attempt to reload the feature config. <br>*value must be > 1s*                                                                                                                                      | 10000      |
| RequestTimeout               | time.Duration  | Maximum time to spend retrieving project configurations. <br>*value must be > 5s*                                                                                                                                               | 5000       |
| DisableAutomaticEventLogging | bool           | Turn off tracking of automated variable events                                                                                                                                                                                  | false      |
| DisableCustomEventLogging    | bool           | Turns off tracking of custom events submitted via the client.Track()                                                                                                                                                            | false      |
| MaxEventQueueSize            | int            | Maximum size of the event queue before new events get dropped. Higher values can impact memory usage of the SDK. <br>*value must be > 0 and <= 50000.*                                                                          | 10000      |
| FlushEventQueueSize          | int            | Maximum size of the queue used to prepare events for submission to DevCycle. Higher values can impact memory usage of the SDK. <br>*value must be > 0 and <= 50000.*                                                            | 1000       | |
| ConfigCDNURI                 | string         | The base URI for retrieving your project configuration from DevCycle. Can be set if you need to proxy traffic through your own server                                                                                           | https://config-cdn.devcycle.com           |
| EventsAPIURI                 | string         | The base URI for sending events to DevCycle for analytics tracking. Can be set if you need to proxy traffic through your own server                                                                                             | https://events.devcycle.com           |
| Logger                       | util.Logger    | Allows you to set a custom logger to manage output from the SDK. The default logger will write to stdout and stderr                                                                                                             | nil        |
| MaxMemoryAllocationBuckets   | int            | Controls the maximum number of pre-allocated memory blocks used for WASM execution to optimize performance. Can be set to -1 to disable pre-allocated memory blocks entirely.<br>*Not applicable for Native Bucketing Library.* | 12         |
| MaxWasmWorkers               | int           | The number of WASM worker objects in the object pool to support high-concurrency. <br>*Not applicable for Native Bucketing Library.*                                                                                            | GOMAXPROCS | 
| UseDebugWASM                 | bool           | Configures the SDK to use a debug WASM binary to generate more detailed error reporting. Use caution when enabling this setting in production environments.<br>*Not applicable for Native Bucketing Library.*                   | false      |


## Native Bucketing Library

This SDK also supports a version of the DevCycle bucketing and segmentation logic built natively in Go. This system is designed as an ultra-high performance alternative to the WASM and Cloud bucketing solutions. 

To activate the native bucketing library, include the following build tag for your application:

```bash
-tags native_bucketing
```

This implementation is still under-going active development. Take care when utilizing it in production environments.

## Linting

We run golangci/golangci-lint on every PR to catch common errors. You can run the linter locally via the Makefile with:
```
make lint
```

Lint failures on PRs will show comments on the "Files changed" tab inline with the code, not on the main Conversation tab.
