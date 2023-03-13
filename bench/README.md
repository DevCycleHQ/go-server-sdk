# Benchy

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