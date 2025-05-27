// Package api provides the HTTP API layer for the Digests application.
// It uses the Huma framework to provide automatic OpenAPI documentation,
// request/response validation, and a clean handler interface.
//
// # Architecture
//
// The API package is structured as follows:
//
// - server.go: Huma API configuration and setup
// - handlers/: HTTP request handlers
// - dto/: Data Transfer Objects for requests and responses
// - middleware/: HTTP middleware for cross-cutting concerns
//
// # Key Features
//
// 1. Automatic OpenAPI Generation
//
// The API automatically generates OpenAPI 3.0 documentation:
// - JSON spec available at /openapi.json
// - Interactive Swagger UI at /docs
//
// 2. Request/Response Validation
//
// Huma provides automatic validation based on struct tags:
//
//	type ParseFeedsRequest struct {
//	    URLs         []string `json:"urls" minItems:"1" maxItems:"100"`
//	    Page         int      `json:"page,omitempty" minimum:"1" default:"1"`
//	    ItemsPerPage int      `json:"items_per_page,omitempty" minimum:"1" maximum:"100" default:"50"`
//	}
//
// 3. Middleware Support
//
// The API includes middleware for:
// - Request logging with unique request IDs
// - Rate limiting per IP address
// - CORS handling (when configured)
// - Authentication (future)
//
// # Usage Example
//
//	// Create API with middleware
//	cfg := api.APIConfig{
//	    Logger:     logger,
//	    RateLimit:  100,
//	    RateWindow: time.Minute,
//	}
//	humaAPI := api.NewAPIWithMiddleware(cfg)
//	
//	// Register handlers
//	feedHandler := handlers.NewFeedHandler(feedService)
//	feedHandler.RegisterRoutes(humaAPI)
//	
//	// Get HTTP handler
//	router := humaAPI.Adapter()
//	
//	// Start server
//	http.ListenAndServe(":8080", router)
//
// # Error Handling
//
// The API uses a consistent error format based on RFC 7807:
//
//	{
//	    "status": 400,
//	    "title": "Bad Request",
//	    "detail": "URL parameter is required",
//	    "instance": "/feed"
//	}
//
// Domain errors are automatically mapped to appropriate HTTP status codes.
//
package api