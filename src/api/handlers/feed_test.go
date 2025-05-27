package handlers

import (
	"context"
	"errors"
	"testing"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/danielgtaylor/huma/v2/humatest"
)

// mockFeedService is a mock implementation of the feed service
type mockFeedService struct {
	parseFeedsFunc func(ctx context.Context, urls []string) ([]*domain.Feed, error)
}

func (m *mockFeedService) ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	if m.parseFeedsFunc != nil {
		return m.parseFeedsFunc(ctx, urls)
	}
	return nil, nil
}

func (m *mockFeedService) ParseSingleFeed(ctx context.Context, url string) (*domain.Feed, error) {
	return nil, nil
}

// mockEnrichmentService is a mock implementation of the content enrichment service
type mockEnrichmentService struct{}

func (m *mockEnrichmentService) ExtractMetadata(ctx context.Context, url string) (*interfaces.MetadataResult, error) {
	return nil, nil
}

func (m *mockEnrichmentService) ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*interfaces.MetadataResult {
	return make(map[string]*interfaces.MetadataResult)
}

func (m *mockEnrichmentService) ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	return nil, nil
}

func (m *mockEnrichmentService) ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor {
	return make(map[string]*domain.RGBColor)
}

func (m *mockEnrichmentService) GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	return nil, nil
}

func TestNewFeedHandler(t *testing.T) {
	mockService := &mockFeedService{}
	mockEnrichment := &mockEnrichmentService{}
	handler := NewFeedHandler(mockService, mockEnrichment)
	
	if handler == nil {
		t.Error("NewFeedHandler returned nil")
	}
	
	if handler.feedService == nil {
		t.Error("FeedHandler.feedService is nil")
	}
}

func TestFeedHandler_RegisterRoutes(t *testing.T) {
	mockService := &mockFeedService{}
	mockEnrichment := &mockEnrichmentService{}
	handler := NewFeedHandler(mockService, mockEnrichment)
	
	// Create test API
	_, api := humatest.New(t)
	
	// Register routes
	handler.RegisterRoutes(api)
	
	// Check if routes are registered by checking OpenAPI spec
	openapi := api.OpenAPI()
	
	// Check POST /feeds endpoint
	if openapi.Paths == nil || openapi.Paths["/feeds"] == nil {
		t.Error("POST /feeds endpoint not registered")
	} else if openapi.Paths["/feeds"].Post == nil {
		t.Error("POST method not registered for /feeds")
	}
}

func TestFeedHandler_ParseFeeds_Success(t *testing.T) {
	// Create mock service
	mockService := &mockFeedService{
		parseFeedsFunc: func(ctx context.Context, urls []string) ([]*domain.Feed, error) {
			// Verify correct URLs are passed
			if len(urls) != 2 {
				t.Errorf("Expected 2 URLs, got %d", len(urls))
			}
			
			// Return test feeds
			return []*domain.Feed{
				{
					ID:          "feed1",
					Title:       "Feed 1",
					Description: "Description 1",
					URL:         urls[0],
					Items:       []domain.FeedItem{},
				},
				{
					ID:          "feed2",
					Title:       "Feed 2",
					Description: "Description 2",
					URL:         urls[1],
					Items:       []domain.FeedItem{},
				},
			}, nil
		},
	}
	
	mockEnrichment := &mockEnrichmentService{}
	handler := NewFeedHandler(mockService, mockEnrichment)
	_, api := humatest.New(t)
	handler.RegisterRoutes(api)
	
	// Make request
	resp := api.Post("/feeds", map[string]interface{}{
		"urls": []string{
			"https://example.com/feed1.xml",
			"https://example.com/feed2.xml",
		},
	})
	
	// Check response
	if resp.Code != 200 {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}
}

func TestFeedHandler_ParseFeeds_ValidationError(t *testing.T) {
	mockService := &mockFeedService{}
	mockEnrichment := &mockEnrichmentService{}
	handler := NewFeedHandler(mockService, mockEnrichment)
	_, api := humatest.New(t)
	handler.RegisterRoutes(api)
	
	// Make request with empty URLs
	resp := api.Post("/feeds", map[string]interface{}{
		"urls": []string{},
	})
	
	// Check response
	if resp.Code != 422 {
		t.Errorf("Expected status 422 for validation error, got %d", resp.Code)
	}
}

func TestFeedHandler_ParseFeeds_ServiceError(t *testing.T) {
	// Create mock service that returns error
	mockService := &mockFeedService{
		parseFeedsFunc: func(ctx context.Context, urls []string) ([]*domain.Feed, error) {
			return nil, errors.New("service error")
		},
	}
	
	mockEnrichment := &mockEnrichmentService{}
	handler := NewFeedHandler(mockService, mockEnrichment)
	_, api := humatest.New(t)
	handler.RegisterRoutes(api)
	
	// Make request
	resp := api.Post("/feeds", map[string]interface{}{
		"urls": []string{"https://example.com/feed.xml"},
	})
	
	// Check response
	if resp.Code != 500 {
		t.Errorf("Expected status 500 for service error, got %d", resp.Code)
	}
}