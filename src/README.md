# Digests API

A high-performance RSS/Atom feed aggregation and discovery API built with Go, following clean architecture principles.

## Features

- **Feed Parsing**: Parse RSS 2.0, Atom, and Podcast feeds with rich metadata
- **Article Metadata Extraction**: Extract Open Graph tags and thumbnails from article pages
- **Thumbnail Color Extraction**: Extract prominent colors from article thumbnails using K-means clustering
- **Feed Discovery**: Discover RSS feeds from websites
- **Concurrent Processing**: Parse multiple feeds simultaneously with rate limiting
- **Caching**: Multi-level caching with Redis or in-memory storage
- **API Compatibility**: Backward compatible with legacy API v1
- **OpenAPI Documentation**: Auto-generated API documentation with Swagger UI
- **Production Ready**: Structured logging, graceful shutdown, health checks

## Architecture

The project follows clean architecture principles with clear separation of concerns:

```
src/
├── api/           # HTTP API layer (Huma framework)
├── core/          # Business logic (domain models and services)
├── infrastructure/# External implementations (cache, HTTP, logging)
├── pkg/           # Shared packages (configuration)
└── cmd/           # Application entry points
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Redis (optional, for Redis cache)
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/digests-api.git
cd digests-api/src
```

2. Install dependencies:
```bash
make deps
```

3. Copy environment configuration:
```bash
cp .env.example .env
```

4. Run the application:
```bash
make run
```

The API will be available at `http://localhost:8000`

### Development with Hot Reload

For development with automatic hot reloading using [Air](https://github.com/cosmtrek/air):

1. Install Air:
```bash
make install-air
```

2. Run with hot reload:
```bash
make dev
```

Air will automatically rebuild and restart the server when you make changes to any Go files.

### API Documentation

- OpenAPI Spec: `http://localhost:8000/openapi.json`
- Swagger UI: `http://localhost:8000/docs`

## API Endpoints

### Parse Multiple Feeds
```bash
POST /parse
Content-Type: application/json

{
  "urls": [
    "https://example.com/feed1.xml",
    "https://example.com/feed2.xml"
  ]
}
```

### Parse Single Feed
```bash
GET /feed?url=https://example.com/feed.xml&page=1&items_per_page=50
```

### Discover Feeds
```bash
POST /discover
Content-Type: application/json

{
  "url": "https://example.com"
}
```

### Extract Metadata
```bash
POST /metadata
Content-Type: application/json

{
  "urls": ["https://example.com/article1", "https://example.com/article2"]
}
```

### Validate URL
```bash
POST /validate
Content-Type: application/json

{
  "url": "https://example.com/feed.xml"
}
```

## Configuration

Configuration is managed through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8000 |
| `REFRESH_TIMER` | Feed refresh interval (seconds) | 60 |
| `CACHE_TYPE` | Cache backend (memory/redis) | memory |
| `REDIS_ADDRESS` | Redis server address | localhost:6379 |
| `REDIS_PASSWORD` | Redis password | (empty) |
| `REDIS_DB` | Redis database number | 0 |

## Development

### Running Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test suites
make test-core
make test-api
make test-infra
```

### Building
```bash
# Build binary
make build

# Build Docker image
make docker-build
```

### Code Quality
```bash
# Format code
make fmt

# Run linter
make lint
```

## Docker Deployment

### Using Docker Compose
```bash
docker-compose up -d
```

### Manual Docker Run
```bash
# Start Redis
make redis-start

# Build and run API
make docker-build
make docker-run
```

## Project Structure

### Core Layer
- **Domain Models**: Feed, FeedItem, Share, SearchResult, RGBColor
- **Services**: FeedService, SearchService, ShareService, MetadataService, ThumbnailColorService
- **Interfaces**: Cache, HTTPClient, Logger, ShareStorage

### Infrastructure Layer
- **Cache**: Redis and in-memory implementations
- **HTTP Client**: Standard HTTP client with retry logic
- **Logger**: Structured logging implementation

### API Layer
- **Handlers**: HTTP request handlers using Huma framework
- **DTOs**: Request/Response data transfer objects
- **Mappers**: Domain to DTO conversion
- **Middleware**: Rate limiting, logging (extensible)

## Testing

The project maintains 100% test coverage for business logic:

- **Unit Tests**: All core services and domain models
- **Integration Tests**: API handlers with mocked services
- **Infrastructure Tests**: Cache, HTTP client, and logger implementations

## Performance

- Concurrent feed parsing with configurable limits
- Response caching with TTL
- Efficient pagination for large feeds
- Exponential backoff for failed requests

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.