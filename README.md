# URL Fetch Service

A high-performance Go service for fetching and managing HTTP URL content with built-in rate limiting, automatic cleanup, and comprehensive error handling.

## Features

- 🚀 **Concurrent URL Fetching** - Fetch multiple URLs simultaneously
- 🔒 **Rate Limiting** - Per-IP token bucket algorithm to prevent abuse
- 🔄 **Redirect Handling** - Automatic redirect following with configurable limits
- 🧹 **Memory Management** - Automatic cleanup with TTL and max result limits
- ⚙️ **Environment Configuration** - All settings configurable via environment variables
- 🏥 **Health Checks** - Built-in health and statistics endpoints
- 📊 **Comprehensive Statistics** - Track fetch results, rate limiting, and cleanup metrics
- 🧪 **Tested** - Unit and integration tests
- 🐳 **Docker Ready** - Containerized and production-ready

## Quick Start

### Prerequisites

- Go 1.25 or higher
- (Optional) Docker for containerized deployment

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd fetch

# Build the binary
go build -o fetch-service

# Run the service
./fetch-service
```

### Using Docker

```bash
# Build the Docker image
docker build -t fetch-service .

# Run the container
docker run -p 8080:8080 fetch-service
```

## Usage

### Submit URLs for Fetching

```bash
curl -X POST http://localhost:8080/fetch \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com", "https://google.com"]}'
```

**Response:**
```json
{
  "message": "URLs submitted for fetching",
  "total_urls": 2,
  "status": "processing"
}
```

### Retrieve Results

```bash
curl http://localhost:8080/fetch
```

**Response:**
```json
{
  "total_urls": 2,
  "success_count": 2,
  "failed_count": 0,
  "pending_count": 0,
  "last_submission": "2025-12-29T18:00:00Z",
  "results": [
    {
      "url": "https://example.com",
      "status": "success",
      "content": "<!doctype html>...",
      "content_length": 1256,
      "status_code": 200,
      "fetched_at": "2025-12-29T18:00:01Z",
      "created_at": "2025-12-29T18:00:00Z",
      "duration": "234ms",
      "redirect_count": 0,
      "final_url": "https://example.com"
    }
  ]
}
```

### Health Check

```bash
curl http://localhost:8080/health
```

**Response:** `OK`

### Service Statistics

```bash
curl http://localhost:8080/stats
```

**Response:**
```json
{
  "rate_limiter": {
    "active_ips": 3,
    "rate_limit": 100,
    "burst_size": 20,
    "window_seconds": 60
  },
  "fetch_stats": {
    "total_urls": 150,
    "success_count": 145,
    "failed_count": 5,
    "pending_count": 0
  },
  "cleanup": {
    "last_cleanup": "2025-12-29T17:50:00Z",
    "total_cleaned": 50,
    "cleanup_count": 5,
    "results_in_memory": 150,
    "ttl": "1h0m0s",
    "max_results": 10000,
    "cleanup_interval": "10m0s"
  }
}
```

### Admin: Clear All Results

```bash
curl -X POST http://localhost:8080/admin/clear
```

## Configuration

All configuration is done via environment variables. See `env.example` for a complete list.

### Server Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SERVER_ADDRESS` | Server listen address and port | `:8080` | `:3000` |

### Fetch Settings

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `FETCH_TIMEOUT` | Timeout for each URL fetch | `30s` | `60s`, `1m` |
| `MAX_REDIRECTS` | Maximum HTTP redirects to follow | `10` | `5`, `20` |
| `MAX_CONTENT_SIZE` | Maximum response size in bytes | `10485760` (10MB) | `5242880` |

### Rate Limiting

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `RATE_LIMIT_REQUESTS` | Max requests per window | `100` | `50`, `200` |
| `RATE_LIMIT_WINDOW` | Rate limit time window | `1m` | `30s`, `5m` |
| `RATE_LIMIT_BURST` | Burst capacity | `20` | `10`, `50` |

### Result Cleanup/TTL

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `RESULT_TTL` | How long to keep results | `1h` | `30m`, `24h` |
| `CLEANUP_INTERVAL` | How often to run cleanup | `10m` | `5m`, `30m` |
| `MAX_RESULTS_IN_MEMORY` | Max results to keep | `10000` | `5000`, `50000` |

### Setting Environment Variables

**Option 1: Export in shell**
```bash
export SERVER_ADDRESS=":3000"
export FETCH_TIMEOUT="60s"
export RATE_LIMIT_REQUESTS="200"
./fetch-service
```

**Option 2: Use .env file**
```bash
cp env.example .env
# Edit .env with your values
export $(cat .env | xargs)
./fetch-service
```

**Option 3: Docker**
```bash
docker run -p 8080:8080 \
  -e FETCH_TIMEOUT="60s" \
  -e RATE_LIMIT_REQUESTS="200" \
  fetch-service
```

## Project Structure

```
fetch/
├── main.go                      # Application entry point
├── main_test.go                 # Integration tests
├── cmd/
│   └── model/                   # Data models
│       └── models.go
├── internal/
│   ├── config/                  # Configuration management
│   │   └── config.go
│   ├── handler/                 # HTTP handlers
│   │   └── handlers.go
│   ├── ratelimit/              # Rate limiting logic
│   │   └── ratelimit.go
│   └── service/                # Core business logic
│       ├── service.go
│       └── service_test.go
├── env.example                  # Example environment config
├── CONFIG.md                    # Detailed configuration guide
├── Dockerfile                   # Docker configuration
├── docker-compose.yml           # Docker Compose setup
└── go.mod                       # Go module dependencies
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/fetch` | Submit URLs for fetching |
| `GET` | `/fetch` | Retrieve fetch results |
| `GET` | `/health` | Health check endpoint |
| `GET` | `/stats` | Service statistics |
| `POST` | `/admin/clear` | Clear all results (admin) |

## Testing

### Run All Tests

```bash
go test -v ./...
```

### Run Specific Package Tests

```bash
# Service tests
go test -v ./internal/service

# Integration tests
go test -v .
```

### Run with Coverage

```bash
go test -v -cover ./...
```

### Run with Race Detection

```bash
go test -v -race ./...
```

## Rate Limiting

The service implements per-IP rate limiting using a token bucket algorithm:

- **Requests Per Window**: Maximum number of requests allowed per IP in a time window
- **Burst Size**: Number of requests that can be made immediately
- **Time Window**: Duration for rate limit calculation

When rate limited, clients receive a `429 Too Many Requests` response with headers:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Window`: Time window for rate limiting

## Memory Management

The service includes automatic memory management to prevent unbounded growth:

### TTL-Based Cleanup
- Results older than `RESULT_TTL` are automatically removed
- Cleanup runs every `CLEANUP_INTERVAL`
- Based on result creation time (`CreatedAt`)

### Max Results Limit
- If results exceed `MAX_RESULTS_IN_MEMORY`, oldest are removed
- Keeps only the most recent results
- Prevents memory exhaustion

### Manual Cleanup
- Use `/admin/clear` endpoint to immediately clear all results
- Useful for testing or emergency memory recovery

## Error Handling

The service handles various error scenarios:

- **Invalid URLs**: Returns `failed` status with error message
- **Network Timeouts**: Respects `FETCH_TIMEOUT` setting
- **Too Many Redirects**: Stops after `MAX_REDIRECTS`
- **Large Responses**: Truncates at `MAX_CONTENT_SIZE`
- **DNS Failures**: Captures and reports connection errors
- **Invalid JSON**: Returns `400 Bad Request` for malformed requests

## Performance Considerations

- **Concurrent Fetching**: Each URL is fetched in a separate goroutine
- **Rate Limiting**: Per-IP to prevent abuse while allowing legitimate traffic
- **Memory Limits**: Automatic cleanup prevents memory leaks
- **Connection Pooling**: Go's HTTP client reuses connections
- **Timeouts**: All requests have configurable timeouts

## Production Deployment

### Recommended Settings

```bash
# Production configuration
export SERVER_ADDRESS=":8080"
export FETCH_TIMEOUT="30s"
export MAX_REDIRECTS="10"
export MAX_CONTENT_SIZE="10485760"
export RATE_LIMIT_REQUESTS="100"
export RATE_LIMIT_WINDOW="1m"
export RATE_LIMIT_BURST="20"
export RESULT_TTL="1h"
export CLEANUP_INTERVAL="10m"
export MAX_RESULTS_IN_MEMORY="10000"
```

### Docker Compose

```yaml
version: '3.8'
services:
  fetch-service:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVER_ADDRESS=:8080
      - FETCH_TIMEOUT=30s
      - RATE_LIMIT_REQUESTS=100
      - RESULT_TTL=1h
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Monitoring

Monitor the service using the `/stats` endpoint:

```bash
# Check rate limiting status
curl http://localhost:8080/stats | jq '.rate_limiter'

# Check fetch statistics
curl http://localhost:8080/stats | jq '.fetch_stats'

# Check memory/cleanup status
curl http://localhost:8080/stats | jq '.cleanup'
```

## Development

### Prerequisites

```bash
# Install Go 1.25+
# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Code Style

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Fix imports
goimports -w .
```

### Adding New Features

1. Add models in `cmd/model/`
2. Add business logic in `internal/service/`
3. Add HTTP handlers in `internal/handler/`
4. Update configuration in `internal/config/` if needed
5. Add tests for all new code
6. Update README and CONFIG.md

## Troubleshooting

### High Memory Usage

```bash
# Reduce result retention
export MAX_RESULTS_IN_MEMORY="5000"
export RESULT_TTL="30m"
export CLEANUP_INTERVAL="5m"
```

### Rate Limiting Too Strict

```bash
# Increase limits
export RATE_LIMIT_REQUESTS="200"
export RATE_LIMIT_BURST="50"
```

### Slow Fetches

```bash
# Increase timeout or reduce redirects
export FETCH_TIMEOUT="60s"
export MAX_REDIRECTS="5"
```