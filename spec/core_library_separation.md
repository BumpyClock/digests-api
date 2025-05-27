# Core Library Separation Strategy

## Overview
This document outlines the strategy to separate business logic from the API implementation, creating a reusable core library that can be compiled for Windows, macOS, and other platforms.

## Current State Analysis

### Tightly Coupled Components
1. **HTTP Handlers contain business logic**
   - `parseHandler` directly processes feeds
   - `searchHandler` contains search logic
   - `discoverHandler` has discovery logic embedded

2. **Global Dependencies**
   - HTTP client used directly in business logic
   - Cache accessed globally
   - Logger used throughout

3. **Web-specific implementations**
   - Error responses tied to HTTP status codes
   - Request/response structs mixed with domain models
   - Middleware logic intertwined with core functionality

## Proposed Architecture

### Layer Separation
```
┌─────────────────────────────────────────────────┐
│             Applications Layer                   │
├─────────────────────┬─────────────┬────────────┤
│   HTTP API (Gin)   │ Windows App │ macOS App  │
├────────────────────┴─────────────┴────────────┤
│              Core Library (Pure Go)             │
├────────────────────────────────────────────────┤
│            Infrastructure Interfaces            │
└────────────────────────────────────────────────┘
```

### Core Library Structure
```
core/
├── feed/
│   ├── parser.go        # Feed parsing logic
│   ├── processor.go     # Feed processing
│   ├── discovery.go     # Feed discovery
│   └── models.go        # Domain models
├── podcast/
│   ├── processor.go     # Podcast-specific logic
│   └── models.go
├── search/
│   ├── engine.go        # Search logic
│   └── models.go
├── reader/
│   ├── extractor.go     # Reader view logic
│   └── models.go
├── share/
│   ├── service.go       # Share functionality
│   └── models.go
├── audio/
│   ├── processor.go     # Audio processing
│   └── models.go
├── metadata/
│   ├── extractor.go     # Metadata extraction
│   └── models.go
├── interfaces/
│   ├── cache.go         # Cache interface
│   ├── http_client.go   # HTTP client interface
│   ├── storage.go       # Storage interface
│   └── logger.go        # Logger interface
└── errors/
    └── types.go         # Domain-specific errors
```

## Implementation Plan

### Phase 1: Define Core Interfaces

```go
// core/interfaces/cache.go
package interfaces

import (
    "context"
    "time"
)

type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

// core/interfaces/http_client.go
package interfaces

import (
    "context"
    "io"
)

type HTTPClient interface {
    Get(ctx context.Context, url string) (HTTPResponse, error)
    Post(ctx context.Context, url string, body io.Reader) (HTTPResponse, error)
}

type HTTPResponse interface {
    StatusCode() int
    Body() io.ReadCloser
    Header(key string) string
}

// core/interfaces/logger.go
package interfaces

type Logger interface {
    Debug(msg string, fields map[string]interface{})
    Info(msg string, fields map[string]interface{})
    Warn(msg string, fields map[string]interface{})
    Error(msg string, fields map[string]interface{})
}
```

### Phase 2: Create Core Services

```go
// core/feed/service.go
package feed

import (
    "context"
    "github.com/digests/core/interfaces"
)

type Service struct {
    cache      interfaces.Cache
    httpClient interfaces.HTTPClient
    logger     interfaces.Logger
}

func NewService(cache interfaces.Cache, client interfaces.HTTPClient, logger interfaces.Logger) *Service {
    return &Service{
        cache:      cache,
        httpClient: client,
        logger:     logger,
    }
}

func (s *Service) ParseFeeds(ctx context.Context, urls []string, page, itemsPerPage int) ([]Feed, error) {
    // Pure business logic here
    // No HTTP concerns, just feed processing
}

func (s *Service) DiscoverFeed(ctx context.Context, url string) (string, error) {
    // Feed discovery logic
    // Uses httpClient interface, not direct HTTP
}
```

### Phase 3: Create Domain Models

```go
// core/feed/models.go
package feed

import "time"

// Pure domain models with no HTTP/JSON tags
type Feed struct {
    ID          string
    Title       string
    Description string
    URL         string
    Items       []FeedItem
    Metadata    FeedMetadata
}

type FeedItem struct {
    ID          string
    Title       string
    Description string
    Link        string
    Published   time.Time
    Author      string
    Content     string
    Thumbnail   Thumbnail
}

// Separate DTOs for API layer
// api/dto/feed_dto.go
type FeedResponse struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    // ... with JSON tags
}
```

### Phase 4: Thin API Handlers

```go
// api/handlers/feed_handler.go
package handlers

import (
    "net/http"
    "github.com/digests/core/feed"
    "github.com/digests/api/dto"
)

type FeedHandler struct {
    feedService *feed.Service
}

func (h *FeedHandler) ParseFeeds(w http.ResponseWriter, r *http.Request) {
    // 1. Parse HTTP request
    var req dto.ParseRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request")
        return
    }
    
    // 2. Call core service
    feeds, err := h.feedService.ParseFeeds(r.Context(), req.URLs, req.Page, req.ItemsPerPage)
    if err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    // 3. Convert to DTO and respond
    response := dto.ToFeedResponse(feeds)
    respondJSON(w, http.StatusOK, response)
}
```

### Phase 5: Dependency Injection Container

```go
// internal/container/container.go
package container

import (
    "github.com/digests/core/feed"
    "github.com/digests/core/podcast"
    "github.com/digests/core/search"
    "github.com/digests/infrastructure/cache"
    "github.com/digests/infrastructure/http"
)

type Container struct {
    // Core services
    FeedService    *feed.Service
    PodcastService *podcast.Service
    SearchService  *search.Service
    
    // Infrastructure
    Cache      interfaces.Cache
    HTTPClient interfaces.HTTPClient
    Logger     interfaces.Logger
}

func NewContainer(config Config) (*Container, error) {
    // Initialize infrastructure
    cache := cache.NewRedisCache(config.Redis)
    httpClient := http.NewClient(config.HTTP)
    logger := logging.NewLogger(config.Log)
    
    // Initialize services with dependencies
    feedService := feed.NewService(cache, httpClient, logger)
    podcastService := podcast.NewService(cache, httpClient, logger)
    searchService := search.NewService(cache, httpClient, logger)
    
    return &Container{
        FeedService:    feedService,
        PodcastService: podcastService,
        SearchService:  searchService,
        Cache:         cache,
        HTTPClient:    httpClient,
        Logger:        logger,
    }, nil
}
```

## Platform-Specific Implementations

### Windows/macOS App Usage

```go
// cmd/desktop/main.go
package main

import (
    "github.com/digests/core/feed"
    "github.com/digests/infrastructure/cache/sqlite"
    "github.com/digests/infrastructure/http/native"
)

func main() {
    // Use SQLite for local caching
    cache := sqlite.NewCache("app.db")
    
    // Use native HTTP client
    httpClient := native.NewClient()
    
    // Initialize core services
    feedService := feed.NewService(cache, httpClient, logger)
    
    // Use in native UI
    feeds, err := feedService.ParseFeeds(ctx, urls, 1, 20)
    // Update UI with feeds
}
```

## Key Design Principles

### 1. **Dependency Inversion**
- Core defines interfaces
- Infrastructure implements interfaces
- Applications inject implementations

### 2. **Domain-Driven Design**
- Core contains only business logic
- No framework dependencies
- Pure Go with minimal external dependencies

### 3. **Hexagonal Architecture**
- Core is the center
- Adapters for different platforms
- Ports (interfaces) define boundaries

### 4. **Testing Strategy**
```go
// core/feed/service_test.go
func TestFeedService_ParseFeeds(t *testing.T) {
    // Use mock implementations
    mockCache := mocks.NewMockCache()
    mockClient := mocks.NewMockHTTPClient()
    mockLogger := mocks.NewMockLogger()
    
    service := feed.NewService(mockCache, mockClient, mockLogger)
    
    // Test pure business logic
    feeds, err := service.ParseFeeds(context.Background(), []string{"http://example.com/feed"}, 1, 10)
    assert.NoError(t, err)
    assert.Len(t, feeds, 1)
}
```

## Migration Strategy

### Step 1: Create Core Package Structure
```bash
mkdir -p core/{feed,podcast,search,reader,share,audio,metadata,interfaces,errors}
```

### Step 2: Extract Business Logic
1. Start with feed parsing (largest component)
2. Move logic piece by piece
3. Create interfaces as needed
4. Keep API working throughout

### Step 3: Refactor Handlers
1. Make handlers thin
2. Use dependency injection
3. Convert between DTOs and domain models

### Step 4: Create Platform Adapters
1. HTTP adapter for web API
2. SQLite adapter for desktop apps
3. Native HTTP clients for each platform

## Benefits

### 1. **Reusability**
- Same core logic for web, desktop, mobile
- Consistent behavior across platforms
- Single source of truth for business rules

### 2. **Testability**
- Easy to unit test core logic
- Mock external dependencies
- No HTTP/framework in tests

### 3. **Maintainability**
- Clear separation of concerns
- Platform-specific code isolated
- Business logic changes don't affect APIs

### 4. **Flexibility**
- Easy to add new platforms
- Switch infrastructure (cache, HTTP client)
- Gradual migration possible

## Compilation for Different Platforms

### Core Library
```bash
# Build core library
go build -buildmode=archive ./core/...
```

### Platform-Specific Builds
```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o digests.exe ./cmd/desktop

# macOS
GOOS=darwin GOARCH=amd64 go build -o digests ./cmd/desktop
GOOS=darwin GOARCH=arm64 go build -o digests-arm64 ./cmd/desktop

# Linux API Server
GOOS=linux GOARCH=amd64 go build -o api-server ./cmd/api
```

## Example: Feed Parser Extraction

### Before (Mixed Concerns)
```go
func parseHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP parsing
    var req ParseRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Business logic mixed in
    for _, url := range req.URLs {
        feed, err := gofeed.NewParser().ParseURL(url)
        // ... processing ...
    }
    
    // HTTP response
    json.NewEncoder(w).Encode(response)
}
```

### After (Separated)
```go
// core/feed/parser.go
func (s *Service) ParseFeed(ctx context.Context, url string) (*Feed, error) {
    // Pure business logic
    content, err := s.httpClient.Get(ctx, url)
    if err != nil {
        return nil, err
    }
    
    return s.parseFeedContent(content)
}

// api/handlers/feed.go
func (h *Handler) Parse(w http.ResponseWriter, r *http.Request) {
    // Only HTTP concerns
    req := parseRequest(r)
    feeds, err := h.feedService.ParseFeeds(r.Context(), req.URLs)
    respondJSON(w, feeds, err)
}
```

## Timeline Estimate

### Phase 1: Foundation (1-2 weeks)
- Set up core package structure
- Define interfaces
- Create dependency injection

### Phase 2: Core Extraction (3-4 weeks)
- Extract feed processing
- Extract search logic
- Extract other services

### Phase 3: API Refactoring (1-2 weeks)
- Thin handlers
- DTO conversions
- Integration testing

### Phase 4: Platform Support (2-3 weeks)
- Desktop app scaffolding
- Platform-specific adapters
- Build pipeline

## Success Criteria

1. **Zero HTTP imports in core package**
2. **100% unit test coverage for core**
3. **API remains backward compatible**
4. **Desktop app uses same core logic**
5. **Build time under 5 minutes for all platforms**