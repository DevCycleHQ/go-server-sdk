# HTTP benchmark test

This tool runs variable evaluations in a local HTTP server, for measuring the latency via any load testing client.

```
$ ./bench --help
Usage of bench:
  -config-interval duration
        interval between checks for config updates (default 1m0s)
  -enable-events
        enable event tracking
  -event-interval duration
        interval between flushing events (default 1m0s)
  -disable-logging
        disables all logging for the server (default false)
  -listen string
        [host]:port to listen on (default ":8080")
  -max-memory-buckets int
        set max memory allocation buckets
  -max-wasm-workers int
        set number of WASM workers (zero defaults to GOMAXPROCS)
  -num-variable int
        Unique variables to use in multipleVariables endpoint (default 85)
  -config-failure-chance float
        Chance of failure when polling config service (default 0.0)  
  -event-failure-chance float
        Chance of failure when flushing events (default 0.0)
  -flush-event-queue-size int
        Maximum number of events to queue before flushing (default 5000)
  -max-event-queue-size int
        Maximum number of events to queue before dropping (default 50000)
  -enable-full-profiling
        Enable full profiling, impacting performance (default false)
  -datadog
        Use datadog for tracing and profiling (default true)
  -datadog-env string
        Datadog environment to report data to (default nil)
```

The server supports the following endpoints:

`/empty` - returns an empty response  

`/variable` - evaluates a single variable

`/multipleVariables` - evaluates multiple variables, as configured by `-num-variable`


## Running the server
```
go run ./bench
```

## Generating requests

We recommend using [Hey](https://github.com/rakyll/hey) for generating load testing requests.

```bash
brew install hey

# Get results for /variable call
hey -n 5000 -c 100 http://127.0.0.1:8080/variable

# Get baseline results for the HTTP server overhead
hey -n 5000 -c 100 http://127.0.0.1:8080/empty

# Get results for /multipleVariables call
hey -n 5000 -c 100 http://127.0.0.1:8080/multipleVariables


```