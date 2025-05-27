// ABOUTME: Feed domain model represents an RSS/Atom feed with its metadata
// ABOUTME: Provides validation logic to ensure feed data integrity

package domain

import (
	"errors"
	"net/url"
	"time"
)

// Feed represents an RSS or Atom feed
type Feed struct {
	// ID is the unique identifier for the feed
	ID string

	// Title is the human-readable title of the feed
	Title string

	// Description provides a brief description of the feed's content
	Description string

	// URL is the feed's source URL (the actual RSS/Atom URL)
	URL string

	// Link is the website URL associated with the feed
	Link string

	// Items contains the feed entries
	Items []FeedItem

	// LastUpdated indicates when the feed was last refreshed
	LastUpdated time.Time

	// Additional metadata fields
	Language    string     // Feed language (e.g., "en-US")
	Favicon     string     // URL to the feed's favicon
	Author      *Author    // Feed author information
	Categories  string     // Feed categories as a string
	FeedType    string     // Type: "article", "podcast", or "rss"
	Image       string     // Feed image URL
	Subtitle    string     // Feed subtitle (for podcasts)
	Published   *time.Time // Feed publication date
}

// Author represents author information
type Author struct {
	Name  string
	Email string
}

// NewFeed creates a new Feed instance with validation
func NewFeed(id, title, description, feedURL string) (*Feed, error) {
	feed := &Feed{
		ID:          id,
		Title:       title,
		Description: description,
		URL:         feedURL,
		Items:       []FeedItem{},
		LastUpdated: time.Now(),
	}

	if err := feed.Validate(); err != nil {
		return nil, err
	}

	return feed, nil
}

// Validate checks if the feed has valid required fields
func (f *Feed) Validate() error {
	if f.Title == "" {
		return errors.New("feed title cannot be empty")
	}

	if f.URL == "" {
		return errors.New("feed URL cannot be empty")
	}

	// Validate URL format
	_, err := url.Parse(f.URL)
	if err != nil {
		return errors.New("feed URL is not valid format")
	}

	return nil
}