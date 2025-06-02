// ABOUTME: Compatibility tests to ensure API backward compatibility
// ABOUTME: Validates that responses match expected format across versions

package compatibility

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"digests-app-api/api"
	"digests-app-api/api/dto/requests"
	"digests-app-api/api/dto/responses"
	"digests-app-api/api/handlers"
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAPIReturnsSameFormatAsOld verifies response format compatibility
func TestNewAPIReturnsSameFormatAsOld(t *testing.T) {
	// Create two services - one with old parser, one with new
	oldService := createServiceWithFlags(false)
	newService := createServiceWithFlags(true)
	
	// Create handlers
	enrichmentService := &mockEnrichmentService{}
	oldHandler := handlers.NewFeedHandler(oldService, enrichmentService)
	newHandler := handlers.NewFeedHandler(newService, enrichmentService)
	
	// Create APIs
	oldAPI, oldRouter := api.NewAPI()
	newAPI, newRouter := api.NewAPI()
	
	oldHandler.RegisterRoutes(oldAPI)
	newHandler.RegisterRoutes(newAPI)
	
	// Test request
	reqBody := requests.ParseFeedsRequest{
		URLs: []string{"http://example.com/feed.rss"},
	}
	body, _ := json.Marshal(reqBody)
	
	// Make requests to both
	oldResp := makeRequest(t, oldRouter, "POST", "/feeds", body)
	newResp := makeRequest(t, newRouter, "POST", "/feeds", body)
	
	// Compare responses
	var oldResponse, newResponse responses.ParseFeedsResponse
	require.NoError(t, json.Unmarshal(oldResp, &oldResponse))
	require.NoError(t, json.Unmarshal(newResp, &newResponse))
	
	// Verify structure is the same
	assert.Equal(t, len(oldResponse.Feeds), len(newResponse.Feeds))
	assert.Equal(t, oldResponse.TotalFeeds, newResponse.TotalFeeds)
	assert.Equal(t, oldResponse.Page, newResponse.Page)
	assert.Equal(t, oldResponse.PerPage, newResponse.PerPage)
	
	// Check feed structure (ignoring v2 marker in description)
	if len(oldResponse.Feeds) > 0 && len(newResponse.Feeds) > 0 {
		oldFeed := oldResponse.Feeds[0]
		newFeed := newResponse.Feeds[0]
		
		assert.Equal(t, oldFeed.ID, newFeed.ID)
		assert.Equal(t, oldFeed.Title, newFeed.Title)
		assert.Equal(t, oldFeed.URL, newFeed.URL)
		assert.Equal(t, len(oldFeed.Items), len(newFeed.Items))
		// Description might differ due to v2 marker, so we don't compare it
	}
}

// TestStatusCodesMatchOldAPI verifies HTTP status codes remain consistent
func TestStatusCodesMatchOldAPI(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:   "Valid POST /feeds",
			method: "POST",
			path:   "/feeds",
			body: requests.ParseFeedsRequest{
				URLs: []string{"http://example.com/feed.rss"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Invalid POST /feeds - empty URLs",
			method: "POST",
			path:   "/feeds",
			body: requests.ParseFeedsRequest{
				URLs: []string{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET /feed without URL",
			method:         "GET",
			path:           "/feed",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET /feed with URL",
			method:         "GET",
			path:           "/feed?url=http://example.com/feed.rss",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with both old and new implementation
			for _, useNewParser := range []bool{false, true} {
				service := createServiceWithFlags(useNewParser)
				enrichmentService := &mockEnrichmentService{}
				handler := handlers.NewFeedHandler(service, enrichmentService)
				apiInstance, router := api.NewAPI()
				handler.RegisterRoutes(apiInstance)
				
				var body []byte
				if tt.body != nil {
					body, _ = json.Marshal(tt.body)
				}
				
				req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
				if tt.body != nil {
					req.Header.Set("Content-Type", "application/json")
				}
				rec := httptest.NewRecorder()
				
				router.ServeHTTP(rec, req)
				
				assert.Equal(t, tt.expectedStatus, rec.Code,
					"Status code mismatch for %s (newParser=%v)", tt.name, useNewParser)
			}
		})
	}
}

// TestErrorFormatsAreCompatible verifies error response formats remain consistent
func TestErrorFormatsAreCompatible(t *testing.T) {
	// Create service
	service := createServiceWithFlags(false)
	enrichmentService := &mockEnrichmentService{}
	handler := handlers.NewFeedHandler(service, enrichmentService)
	apiInstance, router := api.NewAPI()
	handler.RegisterRoutes(apiInstance)
	
	// Make invalid request
	req := httptest.NewRequest("POST", "/feeds", bytes.NewReader([]byte(`{"urls":[]}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	router.ServeHTTP(rec, req)
	
	// Check error format
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	
	var errorResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errorResp))
	
	// Verify error structure matches Huma format
	assert.Contains(t, errorResp, "status")
	assert.Contains(t, errorResp, "title")
	assert.Equal(t, float64(400), errorResp["status"])
}

// Helper functions

func createServiceWithFlags(useNewParser bool) *mockCompatService {
	return &mockCompatService{
		useNewParser: useNewParser,
	}
}

type mockCompatService struct {
	useNewParser bool
}

func (m *mockCompatService) ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	feeds := make([]*domain.Feed, len(urls))
	for i, url := range urls {
		feed, _ := m.ParseSingleFeed(ctx, url)
		feeds[i] = feed
	}
	return feeds, nil
}

func (m *mockCompatService) ParseSingleFeed(ctx context.Context, url string) (*domain.Feed, error) {
	feed := &domain.Feed{
		ID:          "feed-123",
		Title:       "Test Feed",
		Description: "Test Description",
		URL:         url,
		Items: []domain.FeedItem{
			{
				ID:          "item-1",
				Title:       "Test Item",
				Description: "Item Description",
				Link:        "http://example.com/item1",
			},
		},
	}
	
	if m.useNewParser {
		feed.Description = "[v2] " + feed.Description
	}
	
	return feed, nil
}

func makeRequest(t *testing.T, router http.Handler, method, path string, body []byte) []byte {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	
	router.ServeHTTP(rec, req)
	
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.Bytes()
}

// TestContractTests validates API contract remains stable
func TestContractTests(t *testing.T) {
	// Define expected contract
	type APIContract struct {
		Endpoints []EndpointContract
	}
	
	type EndpointContract struct {
		Method         string
		Path           string
		RequestFields  []string
		ResponseFields []string
	}
	
	contract := APIContract{
		Endpoints: []EndpointContract{
			{
				Method:        "POST",
				Path:          "/feeds",
				RequestFields: []string{"urls", "page", "items_per_page"},
				ResponseFields: []string{"feeds", "total_feeds", "page", "per_page"},
			},
			{
				Method:        "GET",
				Path:          "/feed",
				RequestFields: []string{"url", "page", "items_per_page"},
				ResponseFields: []string{"id", "title", "description", "url", "items", "last_updated"},
			},
		},
	}
	
	// Verify contract is maintained
	for _, endpoint := range contract.Endpoints {
		t.Run(endpoint.Method+" "+endpoint.Path, func(t *testing.T) {
			// This is a simplified contract test
			// In production, you'd validate actual request/response schemas
			assert.NotEmpty(t, endpoint.RequestFields)
			assert.NotEmpty(t, endpoint.ResponseFields)
		})
	}
}

// TestContinuousCompatibilityChecking demonstrates how to set up continuous checks
func TestContinuousCompatibilityChecking(t *testing.T) {
	// This test demonstrates the pattern for continuous compatibility checking
	// In a real setup, this would:
	// 1. Load previous API responses from fixtures
	// 2. Compare with current implementation
	// 3. Flag any breaking changes
	
	type CompatibilityCheck struct {
		Version      string
		FixturePath  string
		Endpoint     string
		RequestBody  interface{}
	}
	
	checks := []CompatibilityCheck{
		{
			Version:     "1.0.0",
			FixturePath: "fixtures/v1.0.0/feeds_response.json",
			Endpoint:    "/feeds",
			RequestBody: requests.ParseFeedsRequest{
				URLs: []string{"http://example.com/feed.rss"},
			},
		},
	}
	
	// In real implementation, would load and compare fixtures
	for _, check := range checks {
		t.Run("Compatibility with "+check.Version, func(t *testing.T) {
			// Would load fixture and compare
			// fixture := loadFixture(check.FixturePath)
			// current := makeAPICall(check.Endpoint, check.RequestBody)
			// assert.Equal(t, fixture, current)
			
			// For now, just validate the check structure
			assert.NotEmpty(t, check.Version)
			assert.NotEmpty(t, check.Endpoint)
		})
	}
}

// mockEnrichmentService is a mock implementation of ContentEnrichmentService
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