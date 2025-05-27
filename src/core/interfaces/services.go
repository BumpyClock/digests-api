// ABOUTME: Service interfaces for the core business logic
// ABOUTME: Defines contracts for services used throughout the application

package interfaces

import (
	"context"
	"digests-app-api/core/domain"
)

// MetadataResult contains extracted metadata from a webpage
type MetadataResult struct {
	Title       string
	Description string
	Thumbnail   string // Primary image URL
	Images      []string
	ThemeColor  string
	Domain      string
	Favicon     string
}

// ContentEnrichmentService provides unified content enrichment capabilities
type ContentEnrichmentService interface {
	// Metadata extraction
	ExtractMetadata(ctx context.Context, url string) (*MetadataResult, error)
	ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*MetadataResult
	
	// Color extraction
	ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
	ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor
	GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
}