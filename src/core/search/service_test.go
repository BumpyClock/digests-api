package search

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
)

func TestNewSearchService(t *testing.T) {
	deps := interfaces.Dependencies{}
	
	service := NewSearchService(deps)
	
	if service == nil {
		t.Error("NewSearchService returned nil")
	}
}

func TestValidateQuery_EmptyQuery(t *testing.T) {
	service := &SearchService{}
	
	err := service.validateQuery("")
	
	if err == nil {
		t.Error("validateQuery should return error for empty query")
	}
}

func TestValidateQuery_TooShort(t *testing.T) {
	service := &SearchService{}
	
	err := service.validateQuery("a")
	
	if err == nil {
		t.Error("validateQuery should return error for query length < 2")
	}
}

func TestValidateQuery_TooLong(t *testing.T) {
	service := &SearchService{}
	
	// Create a 101 character string
	longQuery := ""
	for i := 0; i < 101; i++ {
		longQuery += "a"
	}
	
	err := service.validateQuery(longQuery)
	
	if err == nil {
		t.Error("validateQuery should return error for query length > 100")
	}
}

func TestValidateQuery_ValidQuery(t *testing.T) {
	service := &SearchService{}
	
	testCases := []string{
		"go",
		"golang",
		"programming feeds",
		"tech news RSS",
	}
	
	for _, query := range testCases {
		err := service.validateQuery(query)
		if err != nil {
			t.Errorf("validateQuery returned error for valid query %q: %v", query, err)
		}
	}
}

func TestSearchRSSFeeds_ValidatesQuery(t *testing.T) {
	deps := interfaces.Dependencies{}
	service := NewSearchService(deps)
	
	ctx := context.Background()
	results, err := service.SearchRSSFeeds(ctx, "")
	
	if err == nil {
		t.Error("SearchRSSFeeds should return error for invalid query")
	}
	if results != nil {
		t.Error("SearchRSSFeeds should return nil results for invalid query")
	}
}

func TestSearchRSSFeeds_ChecksCacheFirst(t *testing.T) {
	cachedResults := []domain.SearchResult{
		{
			Title:       "Cached Feed",
			Description: "From Cache",
			URL:         "https://example.com/feed.xml",
		},
	}
	
	resultsJSON, _ := json.Marshal(cachedResults)
	httpClientCalled := false
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			expectedKey := "search:rss:golang"
			if key != expectedKey {
				t.Errorf("Cache key = %v, want %v", key, expectedKey)
			}
			return resultsJSON, nil
		},
	}
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			httpClientCalled = true
			return nil, errors.New("should not be called")
		},
	}
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockClient,
	}
	service := NewSearchService(deps)
	
	ctx := context.Background()
	results, err := service.SearchRSSFeeds(ctx, "golang")
	
	if err != nil {
		t.Errorf("SearchRSSFeeds returned error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchRSSFeeds returned %d results, want 1", len(results))
	}
	if results[0].Title != "Cached Feed" {
		t.Errorf("Result title = %v, want 'Cached Feed'", results[0].Title)
	}
	if httpClientCalled {
		t.Error("HTTP client should not be called when cache hit")
	}
}

func TestSearchRSSFeeds_CallsExternalAPI(t *testing.T) {
	apiResponse := `{
		"results": [
			{
				"title": "Go Blog",
				"description": "Official Go Blog",
				"feedUrl": "https://blog.golang.org/feed.atom",
				"siteUrl": "https://blog.golang.org",
				"language": "en"
			}
		]
	}`
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return nil, nil // Cache miss
		},
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			return nil
		},
	}
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			// Verify API URL contains search query
			if !strings.Contains(url, "golang") {
				t.Errorf("API URL should contain search query")
			}
			return &mockResponse{
				statusCode: 200,
				body:       apiResponse,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockClient,
	}
	service := NewSearchService(deps)
	
	ctx := context.Background()
	results, err := service.SearchRSSFeeds(ctx, "golang")
	
	if err != nil {
		t.Errorf("SearchRSSFeeds returned error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchRSSFeeds returned %d results, want 1", len(results))
	}
	if results[0].Title != "Go Blog" {
		t.Errorf("Result title = %v, want 'Go Blog'", results[0].Title)
	}
	if results[0].URL != "https://blog.golang.org/feed.atom" {
		t.Errorf("Result URL = %v, want feed URL", results[0].URL)
	}
}

func TestSearchRSSFeeds_CachesResults(t *testing.T) {
	cacheCalled := false
	var capturedTTL time.Duration
	
	apiResponse := `{"results": [{"title": "Test Feed", "feedUrl": "https://test.com/feed.xml"}]}`
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return nil, nil // Cache miss
		},
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			cacheCalled = true
			capturedTTL = ttl
			expectedKey := "search:rss:test query"
			if key != expectedKey {
				t.Errorf("Cache key = %v, want %v", key, expectedKey)
			}
			return nil
		},
	}
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			return &mockResponse{
				statusCode: 200,
				body:       apiResponse,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockClient,
	}
	service := NewSearchService(deps)
	
	ctx := context.Background()
	_, err := service.SearchRSSFeeds(ctx, "test query")
	
	if err != nil {
		t.Errorf("SearchRSSFeeds returned error: %v", err)
	}
	if !cacheCalled {
		t.Error("SearchRSSFeeds should cache results")
	}
	if capturedTTL != 24*time.Hour {
		t.Errorf("Cache TTL = %v, want 24 hours", capturedTTL)
	}
}

func TestSearchRSSFeeds_EmptyResults(t *testing.T) {
	apiResponse := `{"results": []}`
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return nil, nil // Cache miss
		},
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			return nil
		},
	}
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			return &mockResponse{
				statusCode: 200,
				body:       apiResponse,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockClient,
	}
	service := NewSearchService(deps)
	
	ctx := context.Background()
	results, err := service.SearchRSSFeeds(ctx, "nonexistent")
	
	if err != nil {
		t.Errorf("SearchRSSFeeds returned error: %v", err)
	}
	if results == nil {
		t.Error("SearchRSSFeeds should return empty slice, not nil")
	}
	if len(results) != 0 {
		t.Errorf("SearchRSSFeeds returned %d results, want 0", len(results))
	}
}