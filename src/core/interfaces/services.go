// ABOUTME: Service interfaces for the core business logic
// ABOUTME: Defines contracts for services used throughout the application

package interfaces

import (
	"context"
	"digests-app-api/core/domain"
)

// ThumbnailColorService extracts colors from thumbnail images
type ThumbnailColorService interface {
	ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
	ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor
	GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
}

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

// MetadataService extracts metadata from web pages
type MetadataService interface {
	ExtractMetadata(ctx context.Context, url string) (*MetadataResult, error)
	ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*MetadataResult
}