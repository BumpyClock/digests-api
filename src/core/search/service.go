// ABOUTME: Search service handles RSS feed discovery through external APIs
// ABOUTME: Provides business logic for feed search operations independent of HTTP layer

package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
)

// SearchService handles feed discovery operations
type SearchService struct {
	deps interfaces.Dependencies
}

// NewSearchService creates a new search service instance
func NewSearchService(deps interfaces.Dependencies) *SearchService {
	return &SearchService{
		deps: deps,
	}
}

// validateQuery validates search query parameters
func (s *SearchService) validateQuery(query string) error {
	if query == "" {
		return errors.New("search query cannot be empty")
	}

	if len(query) < 2 {
		return errors.New("search query must be at least 2 characters")
	}

	if len(query) > 100 {
		return errors.New("search query cannot exceed 100 characters")
	}

	return nil
}

// SearchRSSFeeds searches for RSS feeds using an external API
func (s *SearchService) SearchRSSFeeds(ctx context.Context, query string) ([]domain.SearchResult, error) {
	// Validate query
	if err := s.validateQuery(query); err != nil {
		return nil, err
	}

	// Check cache first
	cacheKey := fmt.Sprintf("search:rss:%s", query)
	if s.deps.Cache != nil {
		data, err := s.deps.Cache.Get(ctx, cacheKey)
		if err == nil && data != nil {
			// Deserialize cached results
			var results []domain.SearchResult
			if err := json.Unmarshal(data, &results); err == nil {
				return results, nil
			}
		}
	}

	// Check if we have HTTP client
	if s.deps.HTTPClient == nil {
		return nil, errors.New("HTTP client not configured")
	}

	// Call external API (using a mock URL for now)
	// In a real implementation, this would be a configurable API endpoint
	apiURL := fmt.Sprintf("https://api.feedsearch.example.com/search?q=%s", url.QueryEscape(query))
	
	resp, err := s.deps.HTTPClient.Get(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search feeds: %w", err)
	}
	defer resp.Body().Close()

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode())
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse API response
	var apiResponse struct {
		Results []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			FeedURL     string `json:"feedUrl"`
			SiteURL     string `json:"siteUrl"`
			Language    string `json:"language"`
		} `json:"results"`
	}

	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	// Convert to domain model
	results := make([]domain.SearchResult, 0, len(apiResponse.Results))
	for _, r := range apiResponse.Results {
		results = append(results, domain.SearchResult{
			Title:       r.Title,
			Description: r.Description,
			URL:         r.FeedURL,
			SiteURL:     r.SiteURL,
			Language:    r.Language,
			Score:       0, // API doesn't provide scores in this example
		})
	}

	// Cache results for 24 hours
	if s.deps.Cache != nil && len(results) > 0 {
		if data, err := json.Marshal(results); err == nil {
			_ = s.deps.Cache.Set(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return results, nil
}