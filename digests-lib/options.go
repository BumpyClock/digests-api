// ABOUTME: Configuration options for the Digests library client
// ABOUTME: Provides functional options pattern for flexible client configuration

package digests

import (
	"time"
	
	"digests-app-api/core/interfaces"
	"digests-app-api/core/workers"
	"digests-app-api/infrastructure/cache/memory"
	"digests-app-api/infrastructure/http"
	"digests-app-api/infrastructure/logger"
)

// Option is a functional option for configuring the client
type Option func(*Config) error

// WithCache sets a custom cache implementation
func WithCache(cache interfaces.Cache) Option {
	return func(c *Config) error {
		c.Cache = cache
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client interfaces.HTTPClient) Option {
	return func(c *Config) error {
		c.HTTPClient = client
		return nil
	}
}

// WithLogger sets a custom logger
func WithLogger(logger interfaces.Logger) Option {
	return func(c *Config) error {
		c.Logger = logger
		return nil
	}
}

// WithShareStorage sets a custom share storage implementation
func WithShareStorage(storage interfaces.ShareStorage) Option {
	return func(c *Config) error {
		c.ShareStorage = storage
		return nil
	}
}

// WithWorkerConfig sets the worker pool configuration
func WithWorkerConfig(config workers.WorkerConfig) Option {
	return func(c *Config) error {
		c.WorkerConfig = config
		return nil
	}
}

// WithColorCacheTTL sets the TTL for color cache entries
func WithColorCacheTTL(ttl time.Duration) Option {
	return func(c *Config) error {
		c.ColorCacheTTL = ttl
		return nil
	}
}

// WithBackgroundProcessing enables or disables background processing
func WithBackgroundProcessing(enabled bool) Option {
	return func(c *Config) error {
		c.EnableBackgroundProcessing = enabled
		return nil
	}
}

// WithEnrichmentService sets a custom enrichment service
func WithEnrichmentService(service interfaces.ContentEnrichmentService) Option {
	return func(c *Config) error {
		c.EnrichmentService = service
		return nil
	}
}

// FeedOption is a functional option for feed parsing
type FeedOption func(*FeedOptions)

// FeedOptions holds options for feed parsing
type FeedOptions struct {
	EnrichmentConfig EnrichmentConfig
	Pagination       *PaginationOptions
}

// EnrichmentConfig controls which enrichment features are enabled
type EnrichmentConfig struct {
	ExtractMetadata bool
	ExtractColors   bool
}

// PaginationOptions holds pagination parameters
type PaginationOptions struct {
	Page         int
	ItemsPerPage int
}

// WithEnrichment sets enrichment options for feed parsing
func WithEnrichment(metadata, colors bool) FeedOption {
	return func(o *FeedOptions) {
		o.EnrichmentConfig.ExtractMetadata = metadata
		o.EnrichmentConfig.ExtractColors = colors
	}
}

// WithoutEnrichment disables all enrichment
func WithoutEnrichment() FeedOption {
	return func(o *FeedOptions) {
		o.EnrichmentConfig.ExtractMetadata = false
		o.EnrichmentConfig.ExtractColors = false
	}
}

// WithPagination sets pagination options
func WithPagination(page, itemsPerPage int) FeedOption {
	return func(o *FeedOptions) {
		o.Pagination = &PaginationOptions{
			Page:         page,
			ItemsPerPage: itemsPerPage,
		}
	}
}

// defaultConfig returns the default client configuration
func defaultConfig() Config {
	return Config{
		Cache:                      memory.NewMemoryCache(),
		HTTPClient:                 http.NewDefaultHTTPClient(),
		Logger:                     logger.NewDefaultLogger(),
		ShareStorage:               nil, // Must be provided if share functionality is used
		WorkerConfig:               workers.DefaultWorkerConfig(),
		ColorCacheTTL:              7 * 24 * time.Hour, // 7 days
		EnableBackgroundProcessing: false,
	}
}

// defaultFeedOptions returns the default feed parsing options
func defaultFeedOptions() FeedOptions {
	return FeedOptions{
		EnrichmentConfig: EnrichmentConfig{
			ExtractMetadata: true,
			ExtractColors:   true,
		},
		Pagination: nil,
	}
}