# Configuration Guide

The Digests API uses environment variables for configuration. This allows easy deployment across different environments without code changes.

## Environment Variables

### Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVER_PORT` | HTTP server port | `8080` | No |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` | No |
| `SERVER_REFRESH_TIMER` | Feed refresh interval | `30m` | No |
| `SERVER_MAX_CONCURRENT_REQUESTS` | Max concurrent feed fetches | `100` | No |

### Cache Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CACHE_TYPE` | Cache backend (`memory` or `redis`) | `memory` | No |
| `CACHE_DEFAULT_TTL` | Default cache TTL | `1h` | No |

### Redis Configuration (when CACHE_TYPE=redis)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_ADDRESS` | Redis server address | `localhost:6379` | Yes* |
| `REDIS_PASSWORD` | Redis password | ` ` (empty) | No |
| `REDIS_DB` | Redis database number | `0` | No |

*Required only when CACHE_TYPE=redis

### Logging Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`) | `info` | No |

### Feed Processing

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `FEED_TIMEOUT` | Timeout for fetching individual feeds | `30s` | No |
| `FEED_MAX_ITEMS` | Maximum items to return per feed | `50` | No |

### API Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `API_RATE_LIMIT` | Requests per rate limit window | `100` | No |
| `API_RATE_LIMIT_WINDOW` | Rate limit time window | `1m` | No |

## Configuration Examples

### Development Configuration

```bash
# .env.development
SERVER_PORT=8080
LOG_LEVEL=debug
CACHE_TYPE=memory
FEED_TIMEOUT=10s
API_RATE_LIMIT=1000
```

### Production Configuration

```bash
# .env.production
SERVER_PORT=8080
LOG_LEVEL=info
CACHE_TYPE=redis
REDIS_ADDRESS=redis.internal:6379
REDIS_PASSWORD=secure-password
CACHE_DEFAULT_TTL=1h
FEED_TIMEOUT=30s
FEED_MAX_ITEMS=100
API_RATE_LIMIT=100
API_RATE_LIMIT_WINDOW=1m
```

### Docker Configuration

```yaml
# docker-compose.yml
services:
  api:
    image: digests-api:latest
    environment:
      - SERVER_PORT=8080
      - CACHE_TYPE=redis
      - REDIS_ADDRESS=redis:6379
      - LOG_LEVEL=info
    depends_on:
      - redis
  
  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
```

## Loading Configuration

The application loads configuration in the following order:

1. Default values (built into the application)
2. Environment variables
3. `.env` file (if present, for local development)

Environment variables always take precedence over `.env` file values.

### Using .env Files

For local development, create a `.env` file in the project root:

```bash
cp .env.example .env
# Edit .env with your values
```

**Note**: Never commit `.env` files to version control. Use `.env.example` as a template.

## Validation

The application validates configuration on startup and will fail fast if:
- Required values are missing
- Values are in incorrect format
- Conflicting values are detected

## Time Duration Format

Time durations use Go's duration format:
- `s` - seconds (e.g., `30s`)
- `m` - minutes (e.g., `5m`)
- `h` - hours (e.g., `1h`)

Examples:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h30m` - 1 hour and 30 minutes

## Feature Flags (Future)

The following feature flags are planned:
- `FEATURE_SEARCH_ENABLED` - Enable/disable search functionality
- `FEATURE_SHARE_ENABLED` - Enable/disable share functionality
- `FEATURE_METRICS_ENABLED` - Enable/disable metrics endpoint

## Monitoring Configuration (Future)

Planned monitoring configuration:
- `METRICS_PORT` - Prometheus metrics port
- `TRACING_ENABLED` - Enable OpenTelemetry tracing
- `TRACING_ENDPOINT` - OpenTelemetry collector endpoint

## Security Configuration

### CORS (Future)
- `CORS_ALLOWED_ORIGINS` - Comma-separated list of allowed origins
- `CORS_ALLOWED_METHODS` - Comma-separated list of allowed methods
- `CORS_ALLOWED_HEADERS` - Comma-separated list of allowed headers

### TLS (Future)
- `TLS_CERT_FILE` - Path to TLS certificate
- `TLS_KEY_FILE` - Path to TLS private key

## Tips

1. **Use specific values in production**: Avoid defaults for security-sensitive settings
2. **Set appropriate timeouts**: Balance between reliability and resource usage
3. **Monitor cache hit rates**: Adjust TTL based on usage patterns
4. **Rate limit carefully**: Consider your user base and infrastructure capacity
5. **Use Redis in production**: For multi-instance deployments
6. **Enable debug logging sparingly**: Only for troubleshooting