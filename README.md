# URL Fetcher Service

A high-performance URL fetching service that executes multiple HTTP requests concurrently while respecting individual and global timeout constraints.

## Features

- **Concurrent Execution**: All URLs are fetched simultaneously using goroutines
- **Dual Timeout Support**: Respects both global execution timeout and per-URL individual timeouts
- **Custom Headers**: Supports custom HTTP headers per request
- **Order Preservation**: Results are returned in the same order as input URLs
- **Graceful Degradation**: Partial results returned even when some requests timeout
- **Clean Error Messages**: Distinguishes between global vs individual timeouts

### Prerequisites

- Go 1.21 or later

### Running the Service

```bash
# Start the server (default port 8080)
go run .

# Or specify a custom port
go run . -port 9000
```

### Testing with cURL

**Basic Request:**
```bash
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "execution_timeout": 2000,
    "urls": [
      {"url": "https://httpbin.org/get", "timeout": 1000},
      {"url": "https://httpbin.org/delay/3", "timeout": 500}
    ]
  }'
```

**With Custom Headers:**
```bash
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "execution_timeout": 2000,
    "urls": [
      {
        "url": "https://httpbin.org/headers",
        "timeout": 1000,
        "headers": {
          "X-Device-IP": "10.20.3.15",
          "Authorization": "Bearer token123"
        }
      }
    ]
  }'
```

**Health Check:**
```bash
curl http://localhost:8080/health
```

### Running the Client

The included CLI client provides an easy way to test the service:

```bash
go run ./client

# List available test scenarios
go run ./client -list-scenarios

# Run a specific test scenario
go run ./client -scenario success           # All requests succeed
go run ./client -scenario individual-timeout # Individual timeout demo
go run ./client -scenario global-timeout     # Global timeout demo
go run ./client -scenario mixed              # Mixed results demo

# Fetch specific URLs
go run ./client -urls "https://google.com,https://github.com"

# Custom timeout
go run ./client -urls "https://httpbin.org/delay/1" -timeout 2000

# From JSON file
go run ./client -json request.json

# Compact output
go run ./client -payload-only
```

**Client Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `-server` | `http://localhost:8080` | Server URL |
| `-timeout` | `800` | Global execution timeout (ms) |
| `-urls` | - | Comma-separated URLs |
| `-json` | - | Path to JSON request file |
| `-scenario` | - | Run a predefined test scenario |
| `-list-scenarios` | `false` | List available test scenarios |
| `-pretty` | `true` | Pretty print output |
| `-payload-only` | `false` | Show truncated results only |

## API Reference

### POST /execute

Execute URL fetch requests.

**Request Body:**
```json
{
  "execution_timeout": 800,
  "urls": [
    {
      "url": "https://example.com",
      "timeout": 300,
      "headers": {
        "X-Custom-Header": "value"
      }
    },
    {
      "url": "https://api.example.com/data",
      "timeout": 500
    }
  ]
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `execution_timeout` | int | No | 800 | Global timeout in milliseconds |
| `urls` | array | Yes | - | Array of URL objects |
| `urls[].url` | string | Yes | - | URL to fetch (GET request) |
| `urls[].timeout` | int | No | - | Individual timeout in milliseconds |
| `urls[].headers` | object | No | - | Custom HTTP headers |

**Response:**
```json
{
  "results": [
    {
      "code": 200,
      "payload": "<!DOCTYPE html>..."
    },
    {
      "code": 0,
      "error": "Request aborted by individual timeout"
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `results` | array | Results in same order as input URLs |
| `results[].code` | int | HTTP status code (0 if failed) |
| `results[].payload` | string | Response body (if successful) |
| `results[].error` | string | Error message (if failed) |

### GET /health

Health check endpoint.

**Response:**
```json
{"status": "ok"}
```

## Technical Decisions

### Why Go?

- **Excellent Concurrency**: Goroutines and channels provide lightweight, efficient parallelism
- **Strong HTTP Support**: Built-in `net/http` package is production-ready
- **Context Package**: Native support for timeout propagation and cancellation
- **Single Binary**: Easy deployment without runtime dependencies



1. **Global Timeout** (`execution_timeout`): Applied to the entire operation
2. **Individual Timeout** (`timeout` per URL): Applied to specific requests
3. **Whichever is shorter wins**: If individual timeout exceeds remaining global time, global takes precedence

### Concurrency Model

```
┌─────────────────────────────────────────┐
│          Main Handler                    │
│  - Create context with global timeout    │
│  - Launch N goroutines                   │
├─────────────────────────────────────────┤
│  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐    │
│  │ G1  │  │ G2  │  │ G3  │  │ GN  │    │
│  │URL1 │  │URL2 │  │URL3 │  │URLN │    │
│  └──┬──┘  └──┬──┘  └──┬──┘  └──┬──┘    │
│     │        │        │        │        │
│     ▼        ▼        ▼        ▼        │
│  results[0] [1]      [2]      [N-1]     │
├─────────────────────────────────────────┤
│  WaitGroup.Wait() - collect all results │
└─────────────────────────────────────────┘
```

- **Parallel Execution**: All URLs fetched simultaneously
- **Order Preserved**: Results stored by index in pre-allocated slice
- **No Race Conditions**: Each goroutine writes to its own index

### Error Handling

| Error Type | Response |
|------------|----------|
| Global timeout | `code: 0, error: "Request aborted by global timeout"` |
| Individual timeout | `code: 0, error: "Request aborted by individual timeout"` |
| Invalid URL | `code: 0, error: "Failed to create request: ..."` |
| Network error | `code: 0, error: "Request failed: ..."` |
| HTTP error (4xx/5xx) | `code: <status>, payload: <body>` |

## Project Structure

```
.
├── main.go           # Entry point, HTTP server setup
├── handler.go        # Request handler logic
├── handler_test.go   # Handler unit tests
├── fetcher.go        # URL fetching with timeout management
├── fetcher_test.go   # Fetcher unit tests
├── models.go         # Request/Response data structures
├── models_test.go    # Model unit tests
├── client/
│   └── main.go       # CLI client for testing
├── go.mod            # Go module definition
└── README.md         # This file
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific test file
go test -v -run TestFetchSingleURL
```

### Test Coverage

The test suite covers:

- **Models**: JSON serialization/deserialization
- **Fetcher**: 
  - GET requests with custom headers
  - Individual and global timeouts
  - Error handling (invalid URLs, network errors)
  - Concurrent execution and order preservation
- **Handler**:
  - Request validation
  - Timeout configuration
  - Response formatting
  - Health endpoint

## Assumptions & Notes

1. **GET Only**: This service only supports GET requests, as it is designed for URL fetching/retrieval operations.
2. **Response Size**: No limit on response body size (could be added for production).
3. **Redirects**: Follows up to 10 redirects automatically.
4. **TLS**: Uses system certificate pool for HTTPS verification.

