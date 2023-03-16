# HTTP benchmark test

This tool runs variable evaluations in a local HTTP server, for measuring the latency via any load testing client.

```
$ ./bench --help
Usage of bench:
  -config-interval duration
        interval between checks for config updates (default 1m0s)
  -enable-events
        enable event logging
  -event-interval duration
        interval between flushing events (default 1m0s)
  -listen string
        [host]:port to listen on (default ":8080")
  -max-memory-buckets int
        set max memory allocation buckets
  -max-wasm-workers int
        set number of WASM workers (zero defaults to GOMAXPROCS)
```

## Running the server
```
go run ./bench
```

## Generating requests
```bash
brew install hey

# Get results for /variable call
hey -n 5000 -c 100 http://127.0.0.1:8080/variable

# Get baseline results for the HTTP server overhead
hey -n 5000 -c 100 http://127.0.0.1:8080/empty
```