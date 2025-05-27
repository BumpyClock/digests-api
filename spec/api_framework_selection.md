# API Framework Selection: Huma for FastAPI-like Experience

## Overview
This document outlines the integration of Huma framework to provide FastAPI-like features including automatic OpenAPI documentation, request/response validation, and type-safe routing while maintaining clean separation between core business logic and API layer.

## Why Huma?

### FastAPI-like Features
1. **Automatic OpenAPI 3.1 Generation** - No manual annotations needed
2. **Built-in Validation** - Using struct tags similar to FastAPI's Pydantic
3. **Type-safe Routing** - Compile-time safety for requests/responses
4. **Error Handling** - Structured error responses with proper HTTP codes
5. **Multiple Router Support** - Works with Chi, Gin, Echo, Fiber, or stdlib

### Example Comparison

**FastAPI (Python)**
```python
@app.post("/feeds", response_model=FeedResponse)
async def parse_feeds(request: FeedRequest, page: int = 1):
    return feed_service.parse(request.urls, page)
```

**Huma (Go)**
```go
type FeedRequest struct {
    URLs         []string `json:"urls" minItems:"1" maxItems:"100" doc:"List of feed URLs to parse"`
    Page         int      `json:"page,omitempty" minimum:"1" default:"1" doc:"Page number"`
    ItemsPerPage int      `json:"itemsPerPage,omitempty" minimum:"1" maximum:"100" default:"50"`
}

func (h *Handler) RegisterRoutes(api huma.API) {
    huma.Post(api, "/feeds", h.ParseFeeds)
}

func (h *Handler) ParseFeeds(ctx context.Context, input *FeedRequest) (*FeedResponse, error) {
    // Validation happens automatically
    feeds, err := h.feedService.ParseFeeds(ctx, input.URLs, input.Page, input.ItemsPerPage)
    if err != nil {
        return nil, huma.Error400BadRequest("Failed to parse feeds", err)
    }
    return toFeedResponse(feeds), nil
}
```

## Architecture Integration

### Layer Structure
```
api/
├── handlers/           # Huma handlers (thin layer)
│   ├── feed.go
│   ├── podcast.go
│   └── search.go
├── dto/               # Request/Response structs with validation
│   ├── requests.go
│   └── responses.go
├── middleware/        # HTTP middleware
├── docs/              # Generated OpenAPI docs
└── server.go          # Huma API setup
```

### Core Service Integration

```go
// api/server.go
package api

import (
    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humachi"
    "github.com/go-chi/chi/v5"
    "github.com/digests/core"
)

type Server struct {
    api      huma.API
    core     *core.Services
    handlers *Handlers
}

func NewServer(coreServices *core.Services) *Server {
    // Create router
    router := chi.NewMux()
    
    // Create Huma API
    config := huma.DefaultConfig("Digests API", "1.0.0")
    config.Info.Description = "RSS Feed aggregation and processing API"
    api := humachi.New(router, config)
    
    // Initialize handlers with core services
    handlers := NewHandlers(coreServices)
    
    // Register routes
    handlers.RegisterRoutes(api)
    
    return &Server{
        api:      api,
        core:     coreServices,
        handlers: handlers,
    }
}
```

### Handler Implementation

```go
// api/handlers/feed.go
package handlers

import (
    "context"
    "github.com/danielgtaylor/huma/v2"
    "github.com/digests/core/feed"
    "github.com/digests/api/dto"
)

type FeedHandler struct {
    feedService *feed.Service
}

func (h *FeedHandler) RegisterRoutes(api huma.API) {
    // Automatic OpenAPI documentation for this endpoint
    huma.Register(api, huma.Operation{
        OperationID: "parse-feeds",
        Method:      "POST",
        Path:        "/feeds",
        Summary:     "Parse RSS/Atom feeds",
        Description: "Parse multiple feed URLs and return aggregated feed items with pagination",
        Tags:        []string{"Feeds"},
    }, h.ParseFeeds)
    
    huma.Register(api, huma.Operation{
        OperationID: "discover-feed",
        Method:      "POST", 
        Path:        "/discover",
        Summary:     "Discover RSS feed URL",
        Description: "Discover the RSS/Atom feed URL for a given website",
        Tags:        []string{"Feeds"},
    }, h.DiscoverFeed)
}

func (h *FeedHandler) ParseFeeds(ctx context.Context, input *dto.ParseFeedsRequest) (*dto.ParseFeedsResponse, error) {
    // Input validation is automatic based on struct tags
    
    // Call core service
    feeds, err := h.feedService.ParseFeeds(ctx, input.URLs, input.Page, input.ItemsPerPage)
    if err != nil {
        // Huma provides structured error responses
        return nil, huma.Error500InternalServerError("Failed to parse feeds", err)
    }
    
    // Convert domain models to DTOs
    return dto.ToFeedResponse(feeds), nil
}
```

### DTO with Validation

```go
// api/dto/requests.go
package dto

// Huma uses struct tags for validation and documentation
type ParseFeedsRequest struct {
    URLs         []string `json:"urls" minItems:"1" maxItems:"100" doc:"List of feed URLs to parse" example:"[\"https://example.com/feed.xml\"]"`
    Page         int      `json:"page,omitempty" minimum:"1" default:"1" doc:"Page number for pagination"`
    ItemsPerPage int      `json:"itemsPerPage,omitempty" minimum:"1" maximum:"100" default:"50" doc:"Number of items per page"`
}

type DiscoverFeedRequest struct {
    URL string `json:"url" required:"true" format:"uri" doc:"Website URL to discover feed from" example:"https://example.com"`
}

// Response DTOs
type FeedResponse struct {
    Body struct {
        Feeds []Feed `json:"feeds" doc:"List of parsed feeds"`
        Meta  Meta   `json:"meta" doc:"Pagination metadata"`
    }
}

type Feed struct {
    ID          string     `json:"id" doc:"Unique feed identifier"`
    Title       string     `json:"title" doc:"Feed title" example:"Example Blog"`
    Description string     `json:"description,omitempty" doc:"Feed description"`
    URL         string     `json:"url" format:"uri" doc:"Feed URL"`
    Items       []FeedItem `json:"items" doc:"Feed items"`
}
```

### Validation Examples

```go
// Struct tag validation options in Huma:

type ValidationExample struct {
    // String validation
    Name     string   `json:"name" minLength:"3" maxLength:"50" pattern:"^[a-zA-Z]+$"`
    Email    string   `json:"email" format:"email"`
    URL      string   `json:"url" format:"uri"`
    
    // Number validation  
    Age      int      `json:"age" minimum:"0" maximum:"150"`
    Price    float64  `json:"price" minimum:"0" exclusiveMinimum:"true"`
    
    // Array validation
    Tags     []string `json:"tags" minItems:"1" maxItems:"10" uniqueItems:"true"`
    
    // Required fields
    Required string   `json:"required" required:"true"`
    
    // Enums
    Status   string   `json:"status" enum:"active,inactive,pending"`
    
    // Custom validation
    Custom   string   `json:"custom" doc:"Custom field" example:"example"`
}
```

### OpenAPI Documentation Access

```go
// api/server.go
func (s *Server) Start(port string) error {
    // OpenAPI documentation automatically available at:
    // - /openapi.json - JSON format
    // - /openapi.yaml - YAML format
    // - /docs - Interactive Swagger UI (if enabled)
    
    // Enable Swagger UI
    s.api.UseMiddleware(middleware.SwaggerUI())
    
    return http.ListenAndServe(":"+port, s.api)
}
```

### Middleware Integration

```go
// api/middleware/auth.go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Your auth logic here
        ctx := huma.ContextFromRequest(r)
        
        // Add user to context
        ctx = context.WithValue(ctx, "user", user)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Usage
router.Use(AuthMiddleware)
router.Use(middleware.Logger)
router.Use(middleware.Recoverer)
```

### Error Handling

```go
// Huma provides structured errors
func (h *Handler) GetFeed(ctx context.Context, input *GetFeedInput) (*FeedOutput, error) {
    feed, err := h.feedService.GetFeed(ctx, input.ID)
    
    if err != nil {
        switch err {
        case feed.ErrNotFound:
            return nil, huma.Error404NotFound("Feed not found")
        case feed.ErrInvalidURL:
            return nil, huma.Error400BadRequest("Invalid feed URL", err)
        default:
            return nil, huma.Error500InternalServerError("Internal error", err)
        }
    }
    
    return toFeedOutput(feed), nil
}
```

## Benefits of This Approach

### 1. **Clean Separation**
- Core business logic remains pure Go without web framework dependencies
- API layer only handles HTTP concerns and validation
- Easy to swap web frameworks if needed

### 2. **Developer Experience**
- Automatic OpenAPI documentation from code
- Type-safe request/response handling
- Built-in validation with clear error messages
- Similar to FastAPI's developer experience

### 3. **API Documentation**
- Always up-to-date OpenAPI spec
- Interactive Swagger UI for testing
- Client SDK generation possible
- API versioning support

### 4. **Testing**
```go
func TestParseFeedsHandler(t *testing.T) {
    // Create test API
    _, api := humatest.New(t)
    
    // Mock core service
    mockService := mocks.NewMockFeedService()
    handler := &FeedHandler{feedService: mockService}
    
    // Register routes
    handler.RegisterRoutes(api)
    
    // Test request
    resp := api.Post("/feeds", map[string]interface{}{
        "urls": []string{"https://example.com/feed.xml"},
        "page": 1,
    })
    
    assert.Equal(t, 200, resp.Result().StatusCode)
}
```

## Migration Strategy

### Phase 1: Setup Huma (Week 1)
1. Add Huma dependency
2. Create basic API structure
3. Setup OpenAPI documentation endpoint

### Phase 2: Migrate Endpoints (Week 2-3)
1. Start with simple endpoints (health, metadata)
2. Create DTOs with validation
3. Migrate complex endpoints one by one
4. Maintain backward compatibility

### Phase 3: Documentation (Week 4)
1. Add detailed descriptions to all endpoints
2. Include request/response examples
3. Setup Swagger UI
4. Generate client SDKs

## Example: Migrating Parse Endpoint

### Before (Current Implementation)
```go
func parseHandler(w http.ResponseWriter, r *http.Request) {
    var req ParseRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Manual validation
    if len(req.URLs) == 0 {
        http.Error(w, "No URLs provided", http.StatusBadRequest)
        return
    }
    
    // Business logic mixed with HTTP handling
    responses := processURLs(req.URLs, req.Page, req.ItemsPerPage)
    sendResponse(w, responses)
}
```

### After (With Huma)
```go
func (h *Handler) ParseFeeds(ctx context.Context, input *ParseFeedsInput) (*ParseFeedsOutput, error) {
    // Validation is automatic
    // Just call core service
    feeds, err := h.core.FeedService.ParseFeeds(ctx, input.URLs, input.Page, input.ItemsPerPage)
    if err != nil {
        return nil, err // Huma handles error responses
    }
    
    return &ParseFeedsOutput{
        Body: dto.ToFeedResponse(feeds),
    }, nil
}
```

## Alternative Frameworks

If Huma doesn't meet all needs, consider:

1. **go-swagger** - More mature, design-first approach
2. **Echo + Swagger** - Popular combo with manual documentation
3. **Fiber + Swagger** - Fast performance, Express-like API
4. **Gin + Swag** - Most popular, but requires comment annotations

Huma provides the best balance of automatic documentation, validation, and clean architecture separation for your use case.