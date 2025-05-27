// ABOUTME: Response DTOs for feed-related API endpoints
// ABOUTME: Provides structured responses with JSON serialization

package responses

import "time"

// FeedResponse represents a feed in API responses
type FeedResponse struct {
	ID          string             `json:"id" doc:"Unique identifier for the feed"`
	Title       string             `json:"title" doc:"Feed title"`
	Description string             `json:"description" doc:"Feed description"`
	URL         string             `json:"url" doc:"Feed URL"`
	Items       []FeedItemResponse `json:"items" doc:"Feed items"`
	LastUpdated time.Time          `json:"last_updated" doc:"When the feed was last updated"`
}

// FeedItemResponse represents a feed item in API responses
type FeedItemResponse struct {
	ID          string    `json:"id" doc:"Unique identifier for the item"`
	Title       string    `json:"title" doc:"Item title"`
	Description string    `json:"description" doc:"Item description or content"`
	Link        string    `json:"link" doc:"Link to the full article"`
	Published   time.Time `json:"published" doc:"Publication date"`
	Author      string    `json:"author,omitempty" doc:"Author of the item"`
}

// ParseFeedsResponse represents the response for parsing multiple feeds
type ParseFeedsResponse struct {
	Feeds      []FeedResponse `json:"feeds" doc:"List of parsed feeds"`
	TotalFeeds int            `json:"total_feeds" doc:"Total number of feeds"`
	Page       int            `json:"page" doc:"Current page number"`
	PerPage    int            `json:"per_page" doc:"Items per page"`
}

// FeedErrorResponse represents an error response for a specific feed
type FeedErrorResponse struct {
	URL     string `json:"url" doc:"Feed URL that failed"`
	Error   string `json:"error" doc:"Error message"`
	Code    string `json:"code,omitempty" doc:"Error code"`
}