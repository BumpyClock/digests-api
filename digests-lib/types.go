// ABOUTME: Public types for the Digests library API
// ABOUTME: Provides user-friendly types that wrap internal domain models

package digests

import (
	"time"
	
	"digests-app-api/core/domain"
)

// Feed represents a parsed RSS/Atom feed
type Feed struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	URL         string     `json:"url"`
	Link        string     `json:"link,omitempty"`
	Language    string     `json:"language,omitempty"`
	LastUpdated time.Time  `json:"last_updated"`
	FeedType    string     `json:"feed_type"`
	Items       []FeedItem `json:"items"`
}

// FeedItem represents an item in a feed
type FeedItem struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	Content         string    `json:"content,omitempty"`
	Link            string    `json:"link"`
	Published       time.Time `json:"published"`
	Author          string    `json:"author,omitempty"`
	Thumbnail       string    `json:"thumbnail,omitempty"`
	ThumbnailColor  *RGBColor `json:"thumbnail_color,omitempty"`
	Categories      []string  `json:"categories,omitempty"`
	
	// Podcast-specific fields
	Duration        string    `json:"duration,omitempty"`
	Episode         int       `json:"episode,omitempty"`
	Season          int       `json:"season,omitempty"`
	Image           string    `json:"image,omitempty"`
	AudioURL        string    `json:"audio_url,omitempty"`
	VideoURL        string    `json:"video_url,omitempty"`
}

// RGBColor represents an RGB color
type RGBColor struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// SearchResult represents a feed search result
type SearchResult struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
	FeedURL     string `json:"feed_url"`
}

// Share represents a collection of shared URLs
type Share struct {
	ID        string    `json:"id"`
	URLs      []string  `json:"urls"`
	CreatedAt time.Time `json:"created_at"`
}

// FeedResult represents the result of a feed parsing operation with metadata
type FeedResult struct {
	Feeds      []*Feed           `json:"feeds"`
	Metadata   *ResultMetadata   `json:"metadata"`
}

// ResultMetadata contains metadata about the operation
type ResultMetadata struct {
	TotalItems    int           `json:"total_items"`
	ParseTime     time.Duration `json:"parse_time"`
	CacheHits     int           `json:"cache_hits"`
	CacheMisses   int           `json:"cache_misses"`
	EnrichedItems int           `json:"enriched_items,omitempty"`
	Errors        []string      `json:"errors,omitempty"`
}

// Error definitions
var (
	ErrNoFeedReturned = &DigestsError{Message: "no feed returned"}
	ErrInvalidURL     = &DigestsError{Message: "invalid URL"}
	ErrEmptyQuery     = &DigestsError{Message: "search query cannot be empty"}
)

// DigestsError represents a library-specific error
type DigestsError struct {
	Message string
}

func (e *DigestsError) Error() string {
	return e.Message
}

// Conversion functions

// domainFeedToPublic converts a domain feed to public API type
func domainFeedToPublic(df *domain.Feed) *Feed {
	feed := &Feed{
		ID:          df.ID,
		Title:       df.Title,
		Description: df.Description,
		URL:         df.URL,
		Link:        df.Link,
		Language:    df.Language,
		LastUpdated: df.LastUpdated,
		FeedType:    df.FeedType,
		Items:       make([]FeedItem, len(df.Items)),
	}
	
	for i, di := range df.Items {
		feed.Items[i] = domainItemToPublic(&di)
	}
	
	return feed
}

// domainItemToPublic converts a domain feed item to public API type
func domainItemToPublic(di *domain.FeedItem) FeedItem {
	item := FeedItem{
		ID:          di.ID,
		Title:       di.Title,
		Description: di.Description,
		Content:     di.Content,
		Link:        di.Link,
		Published:   di.Published,
		Author:      di.Author,
		Thumbnail:   di.Thumbnail,
		Categories:  di.Categories,
		Duration:    di.Duration,
		Episode:     di.Episode,
		Season:      di.Season,
		Image:       di.Image,
		AudioURL:    di.AudioURL,
		VideoURL:    di.VideoURL,
	}
	
	if di.ThumbnailColor != nil {
		item.ThumbnailColor = &RGBColor{
			R: di.ThumbnailColor.R,
			G: di.ThumbnailColor.G,
			B: di.ThumbnailColor.B,
		}
	}
	
	return item
}

// applyPagination applies pagination to a slice of feeds
func applyPagination(feeds []*Feed, opts *PaginationOptions) []*Feed {
	if opts == nil || len(feeds) == 0 {
		return feeds
	}
	
	// Calculate start and end indices
	start := (opts.Page - 1) * opts.ItemsPerPage
	if start >= len(feeds) {
		return []*Feed{}
	}
	
	end := start + opts.ItemsPerPage
	if end > len(feeds) {
		end = len(feeds)
	}
	
	return feeds[start:end]
}