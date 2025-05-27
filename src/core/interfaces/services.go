// ABOUTME: Service interfaces for core business logic
// ABOUTME: Defines contracts for feed, search, and share services

package interfaces

import (
	"context"
	"digests-app-api/core/domain"
)

// FeedService defines the interface for feed parsing operations
type FeedService interface {
	// ParseFeeds parses multiple RSS/Atom feeds
	ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error)
	
	// ParseSingleFeed parses a single RSS/Atom feed
	ParseSingleFeed(ctx context.Context, url string) (*domain.Feed, error)
	
	// ParseFeedsWithConfig parses multiple RSS/Atom feeds with enrichment configuration
	ParseFeedsWithConfig(ctx context.Context, urls []string, config interface{}) ([]*domain.Feed, error)
}

// SearchService defines the interface for RSS feed discovery operations
type SearchService interface {
	// SearchRSSFeeds searches for RSS feeds using an external API
	SearchRSSFeeds(ctx context.Context, query string) ([]domain.SearchResult, error)
}

// ShareService defines the interface for URL sharing operations
type ShareService interface {
	// CreateShare creates a new share with the given URLs
	CreateShare(ctx context.Context, urls []string) (*domain.Share, error)
	
	// GetShare retrieves a share by ID
	GetShare(ctx context.Context, id string) (*domain.Share, error)
}

// ContentEnrichmentService defines the interface for content enrichment operations
type ContentEnrichmentService interface {
	// ExtractMetadata extracts metadata from a URL
	ExtractMetadata(ctx context.Context, url string) (*MetadataResult, error)
	
	// ExtractMetadataBatch extracts metadata for multiple URLs
	ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*MetadataResult
	
	// ExtractColor extracts the prominent color from an image URL
	ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
	
	// ExtractColorBatch extracts colors for multiple URLs
	ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor
	
	// GetCachedColor retrieves a cached color without computing
	GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
}

// MetadataResult represents the result of metadata extraction
type MetadataResult struct {
	Title       string
	Description string
	Thumbnail   string
	Author      string
	Published   string
	Images      []string
	ThemeColor  string
	Favicon     string
	Domain      string
}