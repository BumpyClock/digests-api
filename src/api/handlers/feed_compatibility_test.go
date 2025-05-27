package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digests-app-api/api"
	"digests-app-api/api/dto/requests"
	"digests-app-api/api/dto/responses"
	"digests-app-api/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestV1APICompatibility verifies our API matches the v1 schema
func TestV1APICompatibility(t *testing.T) {
	// Create mock service
	mockService := &mockFeedServiceCompat{}
	
	// Create handler
	handler := NewFeedHandler(mockService)
	
	// Create API
	api, router := api.NewAPI()
	handler.RegisterRoutes(api)
	
	// Test request matching the example
	reqBody := requests.ParseFeedsRequest{
		URLs: []string{
			"https://feeds.megaphone.fm/VMP1684715893",
			"https://rss.wbur.org/circle-round-club/podcast",
		},
	}
	
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	// Make request
	router.ServeHTTP(rec, req)
	
	// Check status
	assert.Equal(t, http.StatusOK, rec.Code)
	
	// Parse response
	var response responses.ParseFeedsV1Response
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Verify structure matches v1 API
	assert.NotEmpty(t, response.Feeds)
	
	// Check first feed has all required fields
	feed := response.Feeds[0]
	assert.NotEmpty(t, feed.GUID, "GUID should be generated")
	assert.Equal(t, "ok", feed.Status)
	assert.NotEmpty(t, feed.FeedTitle)
	assert.NotEmpty(t, feed.FeedURL)
	assert.NotEmpty(t, feed.LastUpdated)
	assert.NotEmpty(t, feed.LastRefreshed)
	assert.NotEmpty(t, feed.Items)
	
	// Verify response structure matches expected JSON fields
	responseJSON, _ := json.Marshal(response)
	var responseMap map[string]interface{}
	json.Unmarshal(responseJSON, &responseMap)
	
	// Check top level has feeds array
	feeds, ok := responseMap["feeds"].([]interface{})
	assert.True(t, ok, "Response should have feeds array")
	assert.NotEmpty(t, feeds)
	
	// Check feed structure
	firstFeed := feeds[0].(map[string]interface{})
	expectedFields := []string{
		"guid", "status", "feedTitle", "feedUrl", 
		"description", "link", "lastUpdated", "lastRefreshed", "items",
	}
	
	for _, field := range expectedFields {
		assert.Contains(t, firstFeed, field, "Feed should contain field: %s", field)
	}
}

type mockFeedServiceCompat struct{}

func (m *mockFeedServiceCompat) ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	feeds := make([]*domain.Feed, len(urls))
	
	// Create mock feeds similar to real ones
	feeds[0] = &domain.Feed{
		ID:          "feed-1",
		Title:       "Test Podcast",
		Description: "A test podcast feed",
		URL:         urls[0],
		Items: []domain.FeedItem{
			{
				ID:          "item-1",
				Title:       "Episode 1",
				Description: "First episode",
				Link:        "https://example.com/episode1",
				Author:      "Test Author",
				Published:   time.Now(),
			},
		},
		LastUpdated: time.Now(),
	}
	
	if len(urls) > 1 {
		feeds[1] = &domain.Feed{
			ID:          "feed-2",
			Title:       "CRC",
			Description: "Circle Round Club feed",
			URL:         urls[1],
			Items: []domain.FeedItem{
				{
					ID:          "item-2",
					Title:       "Granny Snowstorm",
					Description: "A story episode",
					Link:        "https://example.com/episode2",
					Author:      "WBUR",
					Published:   time.Now(),
				},
			},
			LastUpdated: time.Now(),
		}
	}
	
	return feeds, nil
}

func (m *mockFeedServiceCompat) ParseSingleFeed(ctx context.Context, url string) (*domain.Feed, error) {
	feeds, err := m.ParseFeeds(ctx, []string{url})
	if err != nil || len(feeds) == 0 {
		return nil, err
	}
	return feeds[0], nil
}