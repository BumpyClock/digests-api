# Digests API - Clean Architecture

## Overview

This project follows Clean Architecture principles with clear separation of concerns and dependency inversion.

## Directory Structure

```
src/
├── api/                    # HTTP API layer
│   ├── dto/               # Data Transfer Objects
│   │   ├── mappers/       # Domain <-> DTO mappers
│   │   ├── requests/      # Request DTOs
│   │   └── responses/     # Response DTOs
│   ├── handlers/          # HTTP request handlers
│   ├── middleware/        # HTTP middleware (logging, rate limiting)
│   └── server.go          # API server setup
│
├── core/                  # Business logic layer
│   ├── domain/           # Domain models
│   ├── errors/           # Domain-specific errors
│   ├── feed/             # Feed parsing service
│   ├── interfaces/       # Port interfaces
│   ├── search/           # Search service
│   ├── services/         # Additional services (metadata, thumbnail color)
│   └── share/            # Share service
│
├── infrastructure/        # External adapters
│   ├── cache/           # Cache implementations
│   │   ├── memory/      # In-memory cache
│   │   └── redis/       # Redis cache
│   ├── http/            # HTTP client implementations
│   └── logger/          # Logger implementations
│
├── cmd/                  # Application entry points
│   └── api/             # API server main
│
├── pkg/                  # Shared packages
│   ├── config/          # Configuration
│   └── featureflags/    # Feature flags
│
├── tests/               # Test suites
│   ├── compatibility/   # API compatibility tests
│   └── performance/     # Performance tests
│
└── docs/                # Documentation
    └── adr/            # Architecture Decision Records
```

## Key Components

### API Layer (`/api`)
- **Handlers**: HTTP request handlers using Huma v2 framework
- **DTOs**: Request/response structures with validation
- **Middleware**: Cross-cutting concerns (logging, rate limiting, CORS)
- **Mappers**: Convert between domain models and DTOs

### Core Layer (`/core`)
- **Domain Models**: Business entities (Feed, FeedItem, etc.)
- **Services**: Business logic implementation
  - Feed Service: RSS/Atom feed parsing
  - Search Service: Feed search functionality
  - Metadata Service: Web page metadata extraction
  - Thumbnail Color Service: Image color extraction
- **Interfaces**: Port definitions for external dependencies

### Infrastructure Layer (`/infrastructure`)
- **Cache**: Memory and Redis implementations
- **HTTP Client**: Standard HTTP client with retry logic
- **Logger**: Structured logging implementation

## Dependency Flow

```
cmd/api/main.go
    ↓
API Handlers ← DTOs
    ↓
Core Services ← Domain Models
    ↓
Infrastructure ← Interfaces
```

## Key Features

1. **Clean Architecture**: Strict separation of concerns with dependency inversion
2. **Testability**: All components are unit testable with mocked dependencies
3. **Scalability**: Stateless design with external cache support
4. **API Compatibility**: Maintains backward compatibility with v1 API
5. **Performance**: Concurrent processing with goroutines and channels
6. **Caching**: Multi-level caching for feeds, metadata, and thumbnail colors
7. **Feature Flags**: Gradual feature rollout support

## Services

### Feed Service
- Parses RSS/Atom feeds using gofeed library
- Enriches feed items with metadata
- Supports pagination and filtering

### Metadata Service
- Extracts Open Graph tags from web pages
- Discovers article thumbnails
- Uses colly for web scraping

### Thumbnail Color Service
- Extracts prominent colors from images
- Uses K-means clustering algorithm
- Caches results for performance

### Search Service
- Full-text search across feeds
- Configurable search fields
- Pagination support

## Configuration

Configuration is loaded from environment variables:
- `API_PORT`: Server port (default: 8080)
- `CACHE_TYPE`: Cache type (memory/redis)
- `REDIS_ADDRESS`: Redis server address
- `REDIS_PASSWORD`: Redis password
- `REDIS_DB`: Redis database number
- `LOG_LEVEL`: Logging level
- `RATE_LIMIT`: Requests per minute