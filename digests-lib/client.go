// ABOUTME: Main client for the Digests library providing feed parsing and enrichment
// ABOUTME: Offers a clean API for using core functionality without HTTP dependencies

package digests

import (
	"context"
	"time"
	
	"digests-app-api/core/domain"
	"digests-app-api/core/feed"
	"digests-app-api/core/interfaces"
	"digests-app-api/core/search"
	"digests-app-api/core/services"
	"digests-app-api/core/share"
	"digests-app-api/core/workers"
)

// Client is the main entry point for the Digests library
type Client struct {
	// Core services
	feedService       interfaces.FeedService
	searchService     interfaces.SearchService
	shareService      interfaces.ShareService
	enrichmentService interfaces.ContentEnrichmentService
	
	// Worker for background processing
	enrichmentWorker  *workers.EnrichmentWorker
	
	// Dependencies
	deps              interfaces.Dependencies
	
	// Configuration
	config            Config
}

// Config holds the configuration for the client
type Config struct {
	// Cache configuration
	Cache interfaces.Cache
	
	// HTTP client configuration
	HTTPClient interfaces.HTTPClient
	
	// Logger configuration
	Logger interfaces.Logger
	
	// Storage configuration
	ShareStorage interfaces.ShareStorage
	
	// Enrichment service (optional - will be created if not provided)
	EnrichmentService interfaces.ContentEnrichmentService
	
	// Worker configuration
	WorkerConfig workers.WorkerConfig
	
	// Color cache TTL
	ColorCacheTTL time.Duration
	
	// Enable background processing
	EnableBackgroundProcessing bool
}

// NewClient creates a new Digests client with the given options
func NewClient(options ...Option) (*Client, error) {
	// Start with default config
	config := defaultConfig()
	
	// Apply options
	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}
	
	// Validate dependencies
	if err := validateConfig(&config); err != nil {
		return nil, err
	}
	
	// Create dependencies
	deps := interfaces.Dependencies{
		HTTPClient: config.HTTPClient,
		Cache:      config.Cache,
		Logger:     config.Logger,
	}
	
	// Create services
	feedService := feed.NewFeedService(deps)
	searchService := search.NewSearchService(deps)
	shareService := share.NewShareService(config.ShareStorage)
	
	// Create enrichment service if not provided
	var enrichmentService interfaces.ContentEnrichmentService
	if config.EnrichmentService != nil {
		enrichmentService = config.EnrichmentService
	} else {
		enrichmentService = services.NewContentEnrichmentService(deps, config.ColorCacheTTL)
	}
	
	// Create client
	client := &Client{
		feedService:       feedService,
		searchService:     searchService,
		shareService:      shareService,
		enrichmentService: enrichmentService,
		deps:              deps,
		config:            config,
	}
	
	// Create and start worker if background processing is enabled
	if config.EnableBackgroundProcessing {
		client.enrichmentWorker = workers.NewEnrichmentWorker(enrichmentService, config.WorkerConfig)
		if err := client.enrichmentWorker.Start(); err != nil {
			return nil, err
		}
	}
	
	return client, nil
}

// Close gracefully shuts down the client
func (c *Client) Close() error {
	if c.enrichmentWorker != nil {
		return c.enrichmentWorker.Stop()
	}
	return nil
}

// ParseFeeds parses multiple RSS/Atom feeds
func (c *Client) ParseFeeds(ctx context.Context, urls []string, opts ...FeedOption) ([]*Feed, error) {
	// Apply feed options
	options := defaultFeedOptions()
	for _, opt := range opts {
		opt(&options)
	}
	
	// Parse feeds using core service
	domainFeeds, err := c.feedService.ParseFeeds(ctx, urls)
	if err != nil {
		return nil, err
	}
	
	// Perform enrichment if enabled
	if options.EnrichmentConfig.ExtractMetadata || options.EnrichmentConfig.ExtractColors {
		c.enrichFeeds(ctx, domainFeeds, options.EnrichmentConfig)
	}
	
	// Convert to public types
	feeds := make([]*Feed, len(domainFeeds))
	for i, df := range domainFeeds {
		feeds[i] = domainFeedToPublic(df)
	}
	
	// Apply pagination if requested
	if options.Pagination != nil {
		feeds = applyPagination(feeds, options.Pagination)
	}
	
	return feeds, nil
}

// ParseFeed parses a single RSS/Atom feed
func (c *Client) ParseFeed(ctx context.Context, url string, opts ...FeedOption) (*Feed, error) {
	feeds, err := c.ParseFeeds(ctx, []string{url}, opts...)
	if err != nil {
		return nil, err
	}
	
	if len(feeds) == 0 {
		return nil, ErrNoFeedReturned
	}
	
	return feeds[0], nil
}

// Search searches for RSS feeds
func (c *Client) Search(ctx context.Context, query string) ([]*SearchResult, error) {
	domainResults, err := c.searchService.SearchRSSFeeds(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// Convert to public types
	results := make([]*SearchResult, len(domainResults))
	for i, dr := range domainResults {
		results[i] = &SearchResult{
			Title:       dr.Title,
			Description: dr.Description,
			URL:         dr.URL,
			FeedURL:     dr.FeedURL,
		}
	}
	
	return results, nil
}

// CreateShare creates a new share with the given URLs
func (c *Client) CreateShare(ctx context.Context, urls []string) (*Share, error) {
	domainShare, err := c.shareService.CreateShare(ctx, urls)
	if err != nil {
		return nil, err
	}
	
	return &Share{
		ID:        domainShare.ID,
		URLs:      domainShare.URLs,
		CreatedAt: domainShare.CreatedAt,
	}, nil
}

// GetShare retrieves a share by ID
func (c *Client) GetShare(ctx context.Context, id string) (*Share, error) {
	domainShare, err := c.shareService.GetShare(ctx, id)
	if err != nil {
		return nil, err
	}
	
	return &Share{
		ID:        domainShare.ID,
		URLs:      domainShare.URLs,
		CreatedAt: domainShare.CreatedAt,
	}, nil
}

// enrichFeeds performs metadata and color enrichment on feeds
func (c *Client) enrichFeeds(ctx context.Context, feeds []*domain.Feed, config EnrichmentConfig) {
	// This is a simplified version - the actual implementation would handle
	// enrichment similar to how the HTTP handler does it
	// For now, we'll leave this as a placeholder
}

// validateConfig validates the client configuration
func validateConfig(config *Config) error {
	if config.HTTPClient == nil {
		return NewError(ErrorTypeConfiguration, "HTTP client is required")
	}
	
	if config.Cache == nil {
		return NewError(ErrorTypeConfiguration, "cache is required")
	}
	
	if config.Logger == nil {
		return NewError(ErrorTypeConfiguration, "logger is required")
	}
	
	// Share storage is optional - only validate if share methods are used
	
	return nil
}