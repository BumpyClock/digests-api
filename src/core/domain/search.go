// ABOUTME: Search domain models for feed discovery results
// ABOUTME: Defines structures for search results from external feed discovery APIs

package domain

// SearchResult represents a discovered RSS feed from search
type SearchResult struct {
	// Title is the feed's title
	Title string

	// Description is the feed's description
	Description string

	// URL is the feed's RSS/Atom URL
	URL string

	// SiteURL is the website URL (not the feed URL)
	SiteURL string

	// Language is the feed's language (e.g., "en", "es")
	Language string

	// Score is the relevance score (0-100)
	Score int
}