# Digests API

A high-performance RSS/Atom feed parser and enrichment service, available both as an HTTP API and a Go library.

## Features

- 🚀 **Fast concurrent feed parsing** - Parse multiple feeds in parallel
- 🎨 **Automatic color extraction** - Extract dominant colors from images
- 📰 **Metadata enrichment** - Extract og:image, descriptions, and more from articles
- 🎙️ **Podcast support** - Full support for podcast feeds with iTunes extensions
- 💾 **Flexible caching** - In-memory, Redis, or SQLite cache options
- 📚 **Library mode** - Use as a standalone Go library without HTTP dependencies
- 🔍 **Feed discovery** - Search for RSS feeds by keyword
- 🔗 **URL sharing** - Create shareable collections of URLs

## Architecture

The project follows Clean Architecture principles with clear separation of concerns:

```
src/
├── core/           # Business logic (library-ready)
│   ├── domain/     # Domain models
│   ├── feed/       # Feed parsing service
│   ├── services/   # Enrichment services
│   └── workers/    # Background processing
├── api/            # HTTP API layer
│   ├── handlers/   # HTTP handlers
│   └── dto/        # Request/Response DTOs
├── infrastructure/ # External implementations
│   ├── cache/      # Cache implementations
│   ├── http/       # HTTP client
│   └── logger/     # Logging
└── digests-lib/    # Go library interface
    ├── client.go   # Main client API
    ├── types.go    # Public types
    └── examples/   # Usage examples
```

## Requirements

- Go 1.21+
- Redis (optional, for Redis cache)
- SQLite (optional, for SQLite cache)

## Installation

### As an HTTP API

```bash
# Clone the repository
git clone https://github.com/BumpyClock/digests-api.git
cd digests-api

# Install dependencies
cd src && go mod download

# Run the API
go run cmd/api/main.go

# Or build and run
go build -o digests-api cmd/api/main.go
./digests-api
```

### As a Go Library

```bash
go get github.com/BumpyClock/digests-api/digests-lib
```

## Configuration

### API Configuration

Environment variables:
- `PORT` - HTTP port (default: 8000)
- `CACHE_TYPE` - Cache type: `memory`, `redis`, or `sqlite` (default: memory)
- `REDIS_URL` - Redis connection URL (default: localhost:6379)
- `SQLITE_PATH` - SQLite database path (default: ./cache.db)
- `COLOR_CACHE_DAYS` - Color cache TTL in days (default: 7)
- `LOG_LEVEL` - Logging level (default: info)

### Library Configuration

```go
client, err := digests.NewClient(
    // Cache options
    digests.WithCacheOption(digests.CacheOption{
        Type:     digests.CacheTypeSQLite,
        FilePath: "./feeds.db",
    }),
    
    // HTTP client options
    digests.WithHTTPClientConfig(digests.HTTPClientConfig{
        Timeout:   30 * time.Second,
        UserAgent: "MyApp/1.0",
    }),
    
    // Enable background processing
    digests.WithBackgroundProcessing(true),
)
```

## API Usage

### Parse Multiple Feeds

```bash
curl -X POST http://localhost:8000/parse \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://news.ycombinator.com/rss",
      "https://feeds.arstechnica.com/arstechnica/index"
    ],
    "enrichment": {
      "extract_metadata": true,
      "extract_colors": true
    }
  }'
```

### Parse Single Feed

```bash
curl "http://localhost:8000/feed?url=https://xkcd.com/rss.xml"
```

### Disable Enrichment for Performance

```bash
curl -X POST http://localhost:8000/parse \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://example.com/feed.xml"],
    "enrichment": {
      "extract_metadata": false,
      "extract_colors": false
    }
  }'
```

## Library Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/BumpyClock/digests-api/digests-lib"
)

func main() {
    // Create client with defaults
    client, err := digests.NewClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Parse a feed
    feed, err := client.ParseFeed(
        context.Background(),
        "https://xkcd.com/rss.xml",
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Feed: %s\n", feed.Title)
    for _, item := range feed.Items {
        fmt.Printf("- %s\n", item.Title)
    }
}
```

### Advanced Example

```go
// Parse without enrichment for speed
feeds, err := client.ParseFeeds(
    ctx,
    urls,
    digests.WithoutEnrichment(),
)

// Parse with pagination
feeds, err := client.ParseFeeds(
    ctx,
    urls,
    digests.WithPagination(1, 20), // Page 1, 20 items
)

// Search for feeds
results, err := client.Search(ctx, "technology news")
```

## API Endpoints

### Feed Parsing

- `POST /parse` - Parse multiple feeds with enrichment options
- `GET /feed` - Parse a single feed

### Discovery

- `GET /discover` - Discover feeds from a website URL

### Metadata

- `POST /metadata/extract` - Extract metadata from URLs

### Validation

- `POST /validate` - Validate feed URLs

## Response Format

```json
{
  "feeds": [{
    "id": "feed-id",
    "title": "Feed Title",
    "description": "Feed description",
    "url": "https://example.com/feed.xml",
    "feed_type": "article|podcast",
    "items": [{
      "id": "item-id",
      "title": "Article Title",
      "description": "Article description",
      "link": "https://example.com/article",
      "published": "2024-01-01T00:00:00Z",
      "thumbnail": "https://example.com/image.jpg",
      "thumbnail_color": {
        "r": 255,
        "g": 128,
        "b": 0
      }
    }]
  }]
}
```

## Performance Considerations

1. **Enrichment Impact**: Metadata and color extraction add latency. Disable when not needed.
2. **Caching**: Use Redis or SQLite for production deployments.
3. **Concurrency**: The service processes multiple feeds concurrently by default.
4. **Background Processing**: Color extraction happens in the background to reduce response time.

## Development

### Running Tests

```bash
cd src
go test ./...
```

### Building

```bash
cd src
go build -o ../build/digests-api cmd/api/main.go
```

### Docker

```bash
docker build -t digests-api .
docker run -p 8000:8000 digests-api
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Huma](https://github.com/danielgtaylor/huma) for the HTTP API
- Uses [gofeed](https://github.com/mmcdole/gofeed) for RSS/Atom parsing
- Color extraction powered by [prominentcolor](https://github.com/EdlinOrg/prominentcolor)