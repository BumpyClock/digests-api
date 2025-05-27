// ABOUTME: Content enrichment service that combines metadata extraction and thumbnail color extraction
// ABOUTME: Provides a unified interface for enriching feed content with additional data

package services

import (
	"context"
	"time"
	
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
)

// ContentEnrichmentService combines metadata and color extraction
type ContentEnrichmentService struct {
	metadata      *MetadataService
	thumbnailColor *ThumbnailColorService
	colorCacheTTL time.Duration
}

// NewContentEnrichmentService creates a new unified enrichment service
func NewContentEnrichmentService(deps interfaces.Dependencies, colorCacheTTL time.Duration) *ContentEnrichmentService {
	thumbnailService := NewThumbnailColorService(deps)
	thumbnailService.cacheTTL = colorCacheTTL
	
	return &ContentEnrichmentService{
		metadata:       NewMetadataService(deps),
		thumbnailColor: thumbnailService,
		colorCacheTTL:  colorCacheTTL,
	}
}

// ExtractMetadata extracts metadata from a URL
func (s *ContentEnrichmentService) ExtractMetadata(ctx context.Context, url string) (*interfaces.MetadataResult, error) {
	return s.metadata.ExtractMetadata(ctx, url)
}

// ExtractMetadataBatch extracts metadata for multiple URLs
func (s *ContentEnrichmentService) ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*interfaces.MetadataResult {
	return s.metadata.ExtractMetadataBatch(ctx, urls)
}

// ExtractColor extracts the prominent color from an image URL
func (s *ContentEnrichmentService) ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	return s.thumbnailColor.ExtractColor(ctx, imageURL)
}

// ExtractColorBatch extracts colors for multiple URLs
func (s *ContentEnrichmentService) ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor {
	return s.thumbnailColor.ExtractColorBatch(ctx, imageURLs)
}

// GetCachedColor retrieves a cached color without computing
func (s *ContentEnrichmentService) GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	return s.thumbnailColor.GetCachedColor(ctx, imageURL)
}

// SetColorCacheTTL updates the cache duration for colors
func (s *ContentEnrichmentService) SetColorCacheTTL(ttl time.Duration) {
	s.colorCacheTTL = ttl
	// Pass to thumbnail service if needed
	s.thumbnailColor.cacheTTL = ttl
}