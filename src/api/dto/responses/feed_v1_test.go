package responses

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFeedV1Response_AllFieldsPresent ensures our response structure matches the current API
func TestFeedV1Response_AllFieldsPresent(t *testing.T) {
	// Create a complete response with all fields
	response := ParseFeedsV1Response{
		Feeds: []FeedV1Response{
			{
				Type:          "podcast",
				GUID:          "fb64054dde2c5f52384b5f5caab567b26b8faa8980ef04edfaebe280f17fecd3",
				Status:        "ok",
				SiteTitle:     "CRC",
				FeedTitle:     "CRC",
				FeedURL:       "https://rss.wbur.org/circle-round-club/podcast",
				Description:   "Thank you for Circling Round with us!",
				Link:          "www.wbur.org",
				LastUpdated:   time.Now().Format(time.RFC3339),
				LastRefreshed: time.Now().UTC().Format(time.RFC3339),
				Published:     time.Now().Format(time.RFC3339),
				Author: &AuthorV1{
					Name:  "WBUR",
					Email: "",
				},
				Language:   "en-us",
				Favicon:    "https://wordpress.wbur.org/wp-content/uploads/2023/04/circle-round-club.jpeg",
				Image:      "https://example.com/image.jpg",
				Categories: "Kids & Family",
				Subtitle:   "Stories for kids",
				Summary:    "Circle Round Club stories",
				Items: []ItemV1Response{
					{
						Type:        "podcast",
						ID:          "7516e2be-1f95-4671-aa7d-6e5a6f0c9c69",
						Title:       "Granny Snowstorm",
						Description: "Composer Eric Shimelonis playing the piano",
						Link:        "https://www.wbur.org/circle-round-club/2024/07/30/granny-snowstorm-crc",
						Author:      "WBUR",
						Published:   time.Now().Format(time.RFC3339),
						Created:     time.Now().Format(time.RFC3339),
						Content:     "Episode content",
						ContentEncoded: "<p>Episode content encoded</p>",
						Categories:  []string{"Kids & Family", "Stories"},
						Duration:    "00:28:19",
						Thumbnail:   "https://wordpress.wbur.org/wp-content/uploads/2024/07/grannySnowstorm.jpg",
						ThumbnailColor: &ColorV1{
							R: 220,
							G: 180,
							B: 140,
						},
						ThumbnailColorComputed: "set",
						Enclosures: []EnclosureV1{
							{
								URL:    "https://traffic.megaphone.fm/BUR9553652211.mp3",
								Length: "26827360",
								Type:   "audio/mpeg",
							},
						},
						Episode:     3,
						Season:      8,
						EpisodeType: "full",
						Subtitle:    "A winter tale",
						Summary:     "A story about winter",
						Image:       "https://example.com/episode.jpg",
						// Legacy fields
						URL:    "https://traffic.megaphone.fm/BUR9553652211.mp3",
						Length: "26827360",
					},
				},
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal back to map to check all fields
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	// Check structure
	feeds, ok := result["feeds"].([]interface{})
	assert.True(t, ok, "Should have feeds array")
	assert.NotEmpty(t, feeds)

	// Check first feed
	feed := feeds[0].(map[string]interface{})
	
	// Required feed fields from current API
	requiredFeedFields := []string{
		"guid", "status", "feedTitle", "feedUrl", "description", 
		"link", "lastUpdated", "lastRefreshed", "items",
	}
	
	for _, field := range requiredFeedFields {
		assert.Contains(t, feed, field, "Feed should contain required field: %s", field)
		assert.NotEmpty(t, feed[field], "Required field %s should not be empty", field)
	}

	// Optional feed fields
	optionalFeedFields := []string{
		"type", "siteTitle", "published", "author", "language", 
		"favicon", "categories", "image", "subtitle", "summary",
	}
	
	for _, field := range optionalFeedFields {
		assert.Contains(t, feed, field, "Feed should contain optional field: %s", field)
	}

	// Check items
	items, ok := feed["items"].([]interface{})
	assert.True(t, ok, "Should have items array")
	assert.NotEmpty(t, items)

	// Check first item
	item := items[0].(map[string]interface{})
	
	// Required item fields
	requiredItemFields := []string{
		"id", "title", "description", "link", "published",
	}
	
	for _, field := range requiredItemFields {
		assert.Contains(t, item, field, "Item should contain required field: %s", field)
		assert.NotEmpty(t, item[field], "Required field %s should not be empty", field)
	}

	// Check enclosures
	enclosures, ok := item["enclosures"].([]interface{})
	assert.True(t, ok, "Should have enclosures array")
	assert.NotEmpty(t, enclosures)
	
	enclosure := enclosures[0].(map[string]interface{})
	assert.Contains(t, enclosure, "url")
	assert.Contains(t, enclosure, "length")
	assert.Contains(t, enclosure, "type")
}

// TestFeedV1Response_MinimalFields tests response with minimal required fields
func TestFeedV1Response_MinimalFields(t *testing.T) {
	response := ParseFeedsV1Response{
		Feeds: []FeedV1Response{
			{
				GUID:          generateGUID("https://example.com/feed.rss"),
				Status:        "ok",
				FeedTitle:     "Test Feed",
				FeedURL:       "https://example.com/feed.rss",
				Description:   "Test Description",
				Link:          "https://example.com",
				LastUpdated:   time.Now().Format(time.RFC3339),
				LastRefreshed: time.Now().UTC().Format(time.RFC3339),
				Items:         []ItemV1Response{},
			},
		},
	}

	// Should marshal without errors
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	
	// Should be valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)
}