// ABOUTME: FeedItem domain model represents an individual entry within a feed
// ABOUTME: Provides validation to ensure item has required fields

package domain

import "time"

// FeedItem represents an individual item/entry in a feed
type FeedItem struct {
	// ID is the unique identifier for the item
	ID string

	// Title is the item's headline
	Title string

	// Description contains the item's content or summary
	Description string

	// Link is the URL to the full article
	Link string

	// Published is when the item was published
	Published time.Time

	// Author is the creator of the item
	Author string

	// Additional content fields
	Content        string     // Plain text content
	ContentEncoded string     // HTML encoded content
	Created        *time.Time // Creation timestamp
	Categories     []string   // Item categories
	
	// Media fields
	Enclosures []Enclosure // Media enclosures
	Thumbnail  string       // Thumbnail image URL
	Duration   string       // Media duration (e.g., "00:28:19")
	
	// Podcast-specific fields
	Episode     int    // Episode number
	Season      int    // Season number
	EpisodeType string // Episode type (e.g., "full", "trailer")
	Subtitle    string // Episode subtitle
	Summary     string // Episode summary
	Image       string // Episode image
}

// Enclosure represents media attachment information
type Enclosure struct {
	URL    string // Media file URL
	Length string // File size in bytes
	Type   string // MIME type
}

// IsValid checks if the feed item has all required fields
func (fi *FeedItem) IsValid() bool {
	if fi.Title == "" {
		return false
	}

	if fi.Link == "" {
		return false
	}

	return true
}

// RGBColor represents an RGB color value
type RGBColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}