// ABOUTME: Huma API server configuration and setup
// ABOUTME: Provides OpenAPI documentation and request/response validation

package api

import (
	"time"

	"digests-app-api/api/middleware"
	"digests-app-api/core/interfaces"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// APIConfig holds configuration for the API
type APIConfig struct {
	Logger      interfaces.Logger
	RateLimit   int           // requests per window
	RateWindow  time.Duration // rate limit window
}

// NewAPI creates and configures a new Huma API instance
func NewAPI() (huma.API, chi.Router) {
	// Create Chi router
	router := chi.NewRouter()
	
	// Create Huma API configuration
	config := huma.DefaultConfig("Digests API", "1.0.0")
	config.Info.Description = "API for managing RSS/Atom feeds and discovering new feeds"
	
	// Create Huma API with Chi adapter
	api := humachi.New(router, config)
	
	// The OpenAPI spec is automatically available at /openapi.json
	// The Swagger UI is automatically available at /docs
	
	return api, router
}

// NewAPIWithMiddleware creates a new API with middleware configured
func NewAPIWithMiddleware(cfg APIConfig) (huma.API, chi.Router) {
	// Create Chi router
	router := chi.NewRouter()
	
	// Apply middleware
	if cfg.Logger != nil {
		router.Use(middleware.RequestLoggingMiddleware(cfg.Logger))
	}
	
	if cfg.RateLimit > 0 && cfg.RateWindow > 0 {
		limiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateWindow)
		router.Use(middleware.RateLimitMiddleware(limiter))
	}
	
	// Create Huma API configuration
	config := huma.DefaultConfig("Digests API", "1.0.0")
	config.Info.Description = "API for managing RSS/Atom feeds and discovering new feeds"
	
	// Create Huma API with Chi adapter
	api := humachi.New(router, config)
	
	return api, router
}