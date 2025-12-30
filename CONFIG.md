# Configuration Guide

The URL Fetch Service is configured via environment variables with sensible defaults.

## Environment Variables

### Server Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SERVER_ADDRESS` | Server listen address and port | `:8080` | `:3000`, `0.0.0.0:8080` |

### Fetch Settings

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `FETCH_TIMEOUT` | Timeout for each URL fetch | `30s` | `60s`, `1m`, `5m` |
| `MAX_REDIRECTS` | Maximum HTTP redirects to follow | `10` | `5`, `20` |
| `MAX_CONTENT_SIZE` | Maximum response size in bytes | `10485760` (10MB) | `5242880` (5MB) |

### Rate Limiting

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `RATE_LIMIT_REQUESTS` | Max requests per window | `100` | `50`, `200` |
| `RATE_LIMIT_WINDOW` | Rate limit time window | `1m` | `30s`, `5m` |
| `RATE_LIMIT_BURST` | Burst capacity | `20` | `10`, `50` |

### Result Cleanup/TTL

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `RESULT_TTL` | How long to keep results | `1h` | `30m`, `2h`, `24h` |
| `CLEANUP_INTERVAL` | How often to run cleanup | `10m` | `5m`, `30m` |
| `MAX_RESULTS_IN_MEMORY` | Max results to keep | `10000` | `5000`, `50000` |

## Usage

### Method 1: Environment Variables

```bash
export SERVER_ADDRESS=":3000"
export FETCH_TIMEOUT="60s"
export RATE_LIMIT_REQUESTS="200"
go run main.go
```

### Method 2: .env File

Create a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
# Edit .env with your values
```

Then use a tool like `godotenv` or `direnv`:

```bash
# With direnv
direnv allow

# Or source it in bash
export $(cat .env | xargs)
go run main.go
```

### Method 3: Docker

```bash
docker run -p 8080:8080 \
  -e SERVER_ADDRESS=":8080" \
  -e FETCH_TIMEOUT="60s" \
  -e RATE_LIMIT_REQUESTS="200" \
  fetch-service
```

### Method 4: Docker Compose

```yaml
version: '3.8'
services:
  fetch-service:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVER_ADDRESS=:8080
      - FETCH_TIMEOUT=60s
      - RATE_LIMIT_REQUESTS=200
      - RESULT_TTL=2h
```

## Duration Format

Duration values support these units:
- `s` - seconds (e.g., `30s`)
- `m` - minutes (e.g., `5m`)
- `h` - hours (e.g., `1h`)
- Combinations (e.g., `1h30m`, `2h45m30s`)

## Examples

### High Performance (More Resources)

```bash
export FETCH_TIMEOUT="60s"
export MAX_REDIRECTS="20"
export RATE_LIMIT_REQUESTS="500"
export RATE_LIMIT_BURST="100"
export MAX_RESULTS_IN_MEMORY="50000"
```

### Resource Constrained (Less Memory)

```bash
export FETCH_TIMEOUT="15s"
export MAX_REDIRECTS="5"
export RATE_LIMIT_REQUESTS="50"
export RATE_LIMIT_BURST="10"
export MAX_RESULTS_IN_MEMORY="1000"
export RESULT_TTL="30m"
```

### Development (Relaxed Limits)

```bash
export SERVER_ADDRESS=":3000"
export FETCH_TIMEOUT="120s"
export RATE_LIMIT_REQUESTS="1000"
export RATE_LIMIT_BURST="200"
export RESULT_TTL="24h"
```

### Production (Balanced)

```bash
export SERVER_ADDRESS=":8080"
export FETCH_TIMEOUT="30s"
export MAX_REDIRECTS="10"
export RATE_LIMIT_REQUESTS="100"
export RATE_LIMIT_BURST="20"
export RESULT_TTL="1h"
export CLEANUP_INTERVAL="10m"
export MAX_RESULTS_IN_MEMORY="10000"
```

## Validation

The service validates all configuration values:
- Invalid integers: Falls back to default, logs warning
- Invalid durations: Falls back to default, logs warning
- Missing values: Uses defaults

## Viewing Current Configuration

On startup, the service logs all configuration values:

```
Configuration:
  Server Address: :8080
  Fetch Timeout: 30s
  Max Redirects: 10
  Max Content Size: 10485760 bytes (10.00 MB)
  Rate Limit: 100 requests per 1m0s (burst: 20)
  Result TTL: 1h0m0s
  Cleanup Interval: 10m0s
  Max Results in Memory: 10000
```

You can also check via the `/stats` endpoint:

```bash
curl http://localhost:8080/stats | jq '.cleanup'
```

## Best Practices

1. **Use .env for local development**
   - Keep `.env` out of git (already in `.gitignore`)
   - Commit `.env.example` with safe defaults

2. **Use environment variables in production**
   - Set via Docker, Kubernetes, or cloud provider
   - Never hardcode secrets

3. **Monitor memory usage**
   - Adjust `MAX_RESULTS_IN_MEMORY` based on available RAM
   - Watch cleanup stats via `/stats` endpoint

4. **Tune rate limits**
   - Start conservative, increase as needed
   - Monitor rate limit stats

5. **Set appropriate TTLs**
   - Balance memory vs. data retention needs
   - Shorter TTL = less memory, more cleanup runs

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
# Increase timeout
export FETCH_TIMEOUT="60s"

# Or reduce redirects
export MAX_REDIRECTS="5"
```

### Too Many Old Results

```bash
# More aggressive cleanup
export RESULT_TTL="30m"
export CLEANUP_INTERVAL="5m"
```

## Configuration in Tests

Tests use default configuration. Override if needed:

```go
func TestWithCustomConfig(t *testing.T) {
    os.Setenv("FETCH_TIMEOUT", "5s")
    defer os.Unsetenv("FETCH_TIMEOUT")
    
    cfg := config.Load()
    // ... test with custom config
}
```


