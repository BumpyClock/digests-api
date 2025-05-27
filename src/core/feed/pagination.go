// ABOUTME: Pagination utilities for feed items
// ABOUTME: Provides functions to paginate feed items for API responses

package feed

import "digests-app-api/core/domain"

// PaginateItems returns a paginated slice of feed items
func PaginateItems(items []domain.FeedItem, page, perPage int) []domain.FeedItem {
	// Handle invalid page
	if page < 1 {
		page = 1
	}

	// Handle invalid perPage
	if perPage < 1 {
		perPage = 10
	}

	// Calculate start and end indices
	start := (page - 1) * perPage
	end := start + perPage

	// Check if start is beyond items
	if start >= len(items) {
		return []domain.FeedItem{}
	}

	// Adjust end if it's beyond items
	if end > len(items) {
		end = len(items)
	}

	// Return the paginated slice
	return items[start:end]
}