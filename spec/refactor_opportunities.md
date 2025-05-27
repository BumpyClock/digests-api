# Digests API Refactoring Opportunities

## Overview
This document outlines the identified opportunities to simplify and improve the maintainability of the Digests API codebase while preserving all current functionality. The primary goal is to separate business logic into a reusable core library package that the HTTP API layer will consume.

## Core Architectural Goal: Platform-Agnostic Core Library

### Vision
Transform the codebase from a monolithic web API into a modular architecture with:
- **Core Library Package**: Pure Go business logic with no web dependencies
- **API Layer**: Thin HTTP handlers using Huma framework for automatic OpenAPI documentation and validation
- **Infrastructure Adapters**: Pluggable implementations for cache, HTTP client, etc.

### Target Architecture
```
┌─────────────────────────────────────────────────┐
│           HTTP API Layer (Huma)                 │
├─────────────────────────────────────────────────┤
│         Core Library Package (Pure Go)          │
├─────────────────────────────────────────────────┤
│        Infrastructure Interfaces                │
└─────────────────────────────────────────────────┘
```

### Core Package Structure
```
core/
├── feed/            # Feed parsing and processing
├── podcast/         # Podcast-specific logic
├── search/          # Search functionality
├── reader/          # Reader view extraction
├── share/           # Content sharing
├── audio/           # Audio processing
├── metadata/        # Metadata extraction
├── interfaces/      # Dependency interfaces
└── domain/          # Pure domain models
```

This architectural change guides all other refactoring decisions.

## 1. Consolidate Global Variables & Configuration

### Current State
- Global variables scattered throughout `server.go`
- No centralized configuration management
- Hard-coded values mixed with configurable parameters

### Proposed Changes
- Create a `config` package with a central `Config` struct
- Implement environment-based configuration loading
- Use dependency injection for shared resources (cache, logger, HTTP client)
- Separate core configuration from infrastructure configuration

### Implementation
```go
// config/config.go
type Config struct {
    Server   ServerConfig
    Cache    CacheConfig
    External ExternalAPIConfig
}

// core/interfaces/dependencies.go
type Dependencies struct {
    Cache      Cache
    HTTPClient HTTPClient
    Logger     Logger
}

// Dependency injection container
type Container struct {
    Config   *Config
    Core     *core.Services
    Infra    *infrastructure.Services
}
```

### Benefits
- Easier testing with mock configurations
- Clear separation of concerns
- Simplified deployment configuration

## 2. Simplify Cache Implementation

### Current State
- Three cache implementations (Redis, GoCache, SQLite)
- SQLite implementation appears unused
- Cache interface tightly coupled to feed-specific methods

### Proposed Changes
- Simplify the core Cache interface to be generic
- Focus on cache implementations needed for API:
  - Redis for distributed caching
  - GoCache for in-memory fallback
- Move cache to infrastructure layer with core interface
- Remove unused SQLite implementation

### Implementation
```go
// core/interfaces/cache.go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

// infrastructure/cache/factory.go
func NewCache(config CacheConfig) (interfaces.Cache, error) {
    if config.Redis.Enabled {
        redis, err := NewRedisCache(config.Redis)
        if err == nil {
            return redis, nil
        }
        log.Warn("Redis unavailable, falling back to memory cache")
    }
    return NewMemoryCache(config.Memory), nil
}
```

### Benefits
- Cleaner cache abstraction
- Focused on API needs
- Easy to test with mock cache
- Simpler configuration

## 3. Refactor Feed Processing Logic into Core Library

### Current State
- `parser.go` contains 1364 lines of mixed concerns
- HTTP handling mixed with feed processing
- Podcast logic intertwined with general feed logic
- Direct cache and HTTP client usage

### Proposed Changes
- Extract to core library modules:
  - `core/feed/parser.go` - Pure feed parsing logic
  - `core/feed/processor.go` - Item processing and transformation
  - `core/feed/fetcher.go` - Feed fetching with injected HTTP client
  - `core/podcast/processor.go` - Podcast-specific logic
  - `core/common/datetime.go` - Date/time utilities
- Move HTTP concerns to API layer with Huma:
  - `api/handlers/feed.go` - Huma handlers with automatic validation
  - `api/dto/feed.go` - Request/response DTOs with validation tags
  - `api/dto/pagination.go` - Pagination for API responses

### Implementation
```go
// core/feed/service.go
type FeedService struct {
    fetcher Fetcher
    parser  Parser
    cache   interfaces.Cache
}

func (s *FeedService) ProcessFeeds(ctx context.Context, urls []string) ([]domain.Feed, error) {
    // Pure business logic, no HTTP concerns
}
```

### Benefits
- Feed processing completely separated from HTTP layer
- Better code organization
- Easier unit testing without HTTP mocking
- Clearer separation of concerns

## 4. Improve Error Handling

### Current State
- String-based errors throughout
- Duplicate error middleware
- Inconsistent error responses

### Proposed Changes
- Create custom error types
- Implement error wrapping with context
- Consolidate error middleware

### Implementation
```go
// errors/types.go
type AppError struct {
    Code    string
    Message string
    Err     error
}

func (e AppError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### Benefits
- Better error tracking
- Consistent error responses
- Easier debugging

## 5. Create HTTP Client Interface

### Current State
- Multiple HTTP clients (global httpClient, colly, custom)
- Direct HTTP client usage in business logic
- Duplicate User-Agent definitions
- Inconsistent timeout handling

### Proposed Changes
- Define HTTP client interface in core
- Create infrastructure implementations:
  - Standard HTTP client with retry logic
  - Colly wrapper for web scraping
  - Mock client for testing
- Centralize User-Agent and timeout configuration
- Implement circuit breakers for resilience

### Implementation
```go
// core/interfaces/http.go
type HTTPClient interface {
    Get(ctx context.Context, url string) (Response, error)
    Post(ctx context.Context, url string, body io.Reader) (Response, error)
}

type Response interface {
    StatusCode() int
    Body() io.ReadCloser
    Header(key string) string
}

// infrastructure/http/client.go
type StandardHTTPClient struct {
    client    *http.Client
    userAgent string
    retries   int
    timeout   time.Duration
}

func (c *StandardHTTPClient) Get(ctx context.Context, url string) (Response, error) {
    // Implementation with retry logic and circuit breaker
}
```

### Benefits
- Core logic independent of HTTP implementation
- Consistent HTTP behavior
- Easy to mock for testing
- Centralized configuration

## 6. Simplify Data Models & Separate Domain from DTOs

### Current State
- Overlapping feed response structures
- Unused struct fields
- Models mixed with business logic
- JSON tags in domain models

### Proposed Changes
- Create pure domain models in core (no JSON/XML tags)
- Separate API DTOs with serialization tags
- Remove unused fields
- Implement mappers between domain and DTOs

### Structure
```
core/domain/
├── feed.go         # Pure domain models
├── podcast.go
└── metadata.go

api/dto/
├── requests.go     # HTTP request DTOs with Huma validation tags
├── responses.go    # HTTP response DTOs
└── mappers.go      # Domain <-> DTO conversion
```

### Implementation Example
```go
// core/domain/feed.go
type Feed struct {
    ID          string
    Title       string
    Description string
    Items       []FeedItem
}

// api/dto/responses.go
type FeedResponse struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Items       []FeedItemResponse `json:"items"`
}

// api/dto/mappers.go
func ToFeedResponse(feed domain.Feed) FeedResponse {
    // Convert domain model to API response
}
```

### Benefits
- Core library has no web dependencies
- Cleaner data structures
- API-specific serialization isolated in DTOs
- Better type safety

## 7. Extract Utilities

### Current State
- Utility functions scattered across files
- `utils.go` contains only one handler function
- Duplicate functionality

### Proposed Changes
- Create focused utility packages:
  - `utils/image` - Image processing
  - `utils/url` - URL validation and sanitization
  - `utils/html` - HTML parsing utilities
- Remove generic `utils.go`

### Benefits
- Better code reusability
- Easier to test utilities
- Clearer function purposes

## 8. Improve Middleware Architecture

### Current State
- Middleware defined in multiple places
- Duplicate implementations
- No clear middleware chain

### Proposed Changes
- Create a `middleware` package
- Implement middleware chaining
- Remove duplicate implementations

### Implementation
```go
// middleware/chain.go
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler
```

### Benefits
- Cleaner request pipeline
- Easier to add/remove middleware
- Better performance monitoring

## 9. Simplify External API Integration

### Current State
- Direct API calls in handlers
- Authentication logic mixed with business logic
- No clear API client abstraction

### Proposed Changes
- Create dedicated API clients
- Implement response caching strategies
- Centralize authentication

### Structure
```
external/
├── podcastindex/
│   └── client.go
├── feedsearch/
│   └── client.go
└── texttospeech/
    └── client.go
```

### Benefits
- Easier to mock external services
- Better error handling
- Clearer API boundaries

## 10. Code Organization

### Current State
- Monolithic `main` package
- Mixed responsibilities
- Business logic tied to HTTP handlers
- Difficult to navigate

### Proposed Structure
```
.
├── core/                    # Business logic package
│   ├── feed/
│   ├── podcast/
│   ├── search/
│   ├── reader/
│   ├── share/
│   ├── audio/
│   ├── metadata/
│   ├── interfaces/          # Dependency interfaces
│   └── domain/              # Pure domain models
├── infrastructure/          # Interface implementations
│   ├── cache/
│   │   ├── redis/
│   │   └── memory/
│   ├── http/
│   │   ├── client.go
│   │   └── retryable.go
│   └── storage/
├── api/                     # Huma-based HTTP API
│   ├── handlers/            # Huma handlers
│   ├── middleware/          # HTTP middleware
│   ├── dto/                 # DTOs with validation tags
│   ├── docs/                # Generated OpenAPI docs
│   └── server.go            # Huma API setup
├── cmd/
│   └── api/                 # API server entry point
└── pkg/                     # Shared utilities
    ├── config/
    └── logging/
```

### Benefits
- Core logic completely separated
- Clear module boundaries
- API consumes core as a package
- Better for team collaboration

## 11. Remove Redundant Features

### Current State
- Share feature uses simple 6-character keys
- Multiple thumbnail discovery implementations
- Duplicate color extraction logic

### Proposed Changes
- Use UUIDs for share links
- Consolidate thumbnail discovery
- Unify color extraction caching

### Benefits
- Less code to maintain
- More secure share links
- Consistent behavior

## 12. Improve Async Processing

### Current State
- Manual goroutine management
- No proper context propagation
- No circuit breakers

### Proposed Changes
- Implement worker pool pattern
- Add context to all async operations
- Implement circuit breakers for external calls

### Implementation
```go
// workers/pool.go
type WorkerPool struct {
    workers   int
    taskQueue chan Task
}

func (p *WorkerPool) Submit(ctx context.Context, task Task) error
```

### Benefits
- Better resource management
- Graceful shutdowns
- Improved reliability

## Implementation Priority

### Phase 0: Foundation for Core Library (Week 1-2)
1. Define core interfaces (cache, HTTP client, logger)
2. Create core package structure
3. Set up dependency injection framework
4. Establish domain models without web dependencies

### Phase 1: Core Business Logic Extraction (Week 3-5)
1. Extract feed parsing logic to core
2. Move search functionality to core
3. Separate podcast processing
4. Create service layer with injected dependencies
5. Implement error types in core

### Phase 2: Infrastructure Layer (Week 6-7)
1. Implement infrastructure adapters
2. Standardize HTTP client with interface
3. Create cache factory for multiple implementations
4. Build configuration management

### Phase 3: API Layer Refactoring with Huma (Week 8-9)
1. Integrate Huma framework for API layer
2. Create Huma handlers with automatic validation
3. Implement DTO layer with validation tags
4. Set up automatic OpenAPI documentation
5. Consolidate middleware for Huma
6. Update routes to use core services

### Phase 4: Optimization and Cleanup (Week 10+)
1. Remove redundant features
2. Improve async processing
3. Consolidate utilities
4. Performance optimization

## Testing Strategy
- Write tests for core library with mock dependencies
- Maintain existing API contracts with integration tests
- Test each platform adapter separately
- Use interface-based testing for easy mocking
- Implement gradual rollout for critical changes

### Core Library Testing
```go
// core/feed/service_test.go
func TestFeedService_ParseFeeds(t *testing.T) {
    mockCache := mocks.NewMockCache()
    mockHTTP := mocks.NewMockHTTPClient()
    service := feed.NewService(mockCache, mockHTTP, logger)
    
    // Test pure business logic without HTTP/framework concerns
}
```

## Risks and Mitigation
- **Risk**: Breaking existing functionality
  - **Mitigation**: Comprehensive test coverage before refactoring
- **Risk**: Performance regression
  - **Mitigation**: Benchmark critical paths before and after
- **Risk**: API compatibility
  - **Mitigation**: Maintain existing endpoints and response formats

## Success Metrics
- **Core Library Independence**: Zero HTTP/web framework imports in core package
- **Clean Separation**: Core package can be imported and used independently
- **Test Coverage**: >85% for core library, >70% overall
- **Build Performance**: <1 minute for API build
- **Code Duplication**: 40% reduction through separation
- **API Compatibility**: 100% backward compatibility maintained
- **API Documentation**: 100% endpoints documented with OpenAPI via Huma
- **Validation Coverage**: All API inputs validated automatically
- **Developer Experience**: Clear boundary between core and API layers

## Migration Approach

### Incremental Migration
1. Start with least complex modules (e.g., metadata extraction)
2. Maintain parallel implementations during transition
3. Use feature flags to switch between old and new implementations
4. Migrate one endpoint at a time
5. Keep existing tests passing throughout

### Example Migration: Feed Parser
```go
// Step 1: Create core service
// core/feed/service.go
type Service struct {
    cache      interfaces.Cache
    httpClient interfaces.HTTPClient
}

// Step 2: Create Huma handler in API
// api/handlers/feed.go
func (h *Handler) ParseFeeds(ctx context.Context, input *dto.ParseFeedsInput) (*dto.ParseFeedsOutput, error) {
    // Validation happens automatically via Huma
    feeds, err := h.coreService.ParseFeeds(ctx, input.URLs, input.Page, input.ItemsPerPage)
    if err != nil {
        return nil, huma.Error400BadRequest("Failed to parse feeds", err)
    }
    return &dto.ParseFeedsOutput{Body: dto.ToFeedResponse(feeds)}, nil
}

// Step 3: Remove old implementation once stable
```

## API Framework: Huma

### Why Huma?
Huma provides FastAPI-like features for Go, including:
- **Automatic OpenAPI 3.1 Generation**: No manual annotations needed
- **Built-in Validation**: Using struct tags similar to FastAPI's Pydantic
- **Type-safe Routing**: Compile-time safety for requests/responses
- **Structured Error Handling**: Consistent error responses

### Huma Integration Example
```go
// api/dto/requests.go - Request DTOs with validation
type ParseFeedsInput struct {
    Body struct {
        URLs         []string `json:"urls" minItems:"1" maxItems:"100" doc:"Feed URLs to parse"`
        Page         int      `json:"page,omitempty" minimum:"1" default:"1"`
        ItemsPerPage int      `json:"itemsPerPage,omitempty" minimum:"1" maximum:"100" default:"50"`
    }
}

// api/handlers/feed.go - Huma handler
func (h *Handler) RegisterRoutes(api huma.API) {
    huma.Register(api, huma.Operation{
        OperationID: "parse-feeds",
        Method:      "POST",
        Path:        "/feeds",
        Summary:     "Parse RSS/Atom feeds",
        Tags:        []string{"Feeds"},
    }, h.ParseFeeds)
}

// api/server.go - API setup
func NewAPI(core *core.Services) huma.API {
    router := chi.NewMux()
    config := huma.DefaultConfig("Digests API", "1.0.0")
    api := humachi.New(router, config)
    
    handlers := NewHandlers(core)
    handlers.RegisterRoutes(api)
    
    return api
}
```

### Benefits of Huma
1. **Developer Experience**: Similar to FastAPI with automatic docs
2. **API Documentation**: Always up-to-date at `/openapi.json` and `/docs`
3. **Validation**: Automatic request validation with clear error messages
4. **Clean Architecture**: Keeps API layer thin, business logic in core

## Summary

The refactoring plan centers around creating a clean separation between business logic (core package) and HTTP concerns (API layer), with Huma providing a modern API experience. This approach enables:

1. **Complete Separation**: Core package contains only business logic, no HTTP dependencies
2. **Better Testing**: Core logic can be tested without HTTP/framework mocking
3. **Modern API Layer**: Huma provides automatic OpenAPI docs and validation like FastAPI
4. **Cleaner Architecture**: Clear boundaries with dependency injection
5. **Easier Maintenance**: Changes to business logic don't affect API layer
6. **Package Reusability**: Core can be imported by other Go projects

The HTTP API layer will consume the core package, keeping all web-specific concerns (routing, validation, serialization) separate from business logic. The migration can be done incrementally while maintaining backward compatibility, ensuring zero downtime and minimal risk.