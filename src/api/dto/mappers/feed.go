// ABOUTME: Mappers for converting between domain models and API DTOs
// ABOUTME: Provides clean separation between business logic and API layer

package mappers

import (
	"digests-app-api/api/dto/responses"
	"digests-app-api/core/domain"
)

// ToFeedResponse converts a domain Feed to a FeedResponse DTO
func ToFeedResponse(feed *domain.Feed) *responses.FeedResponse {
	if feed == nil {
		return nil
	}

	response := &responses.FeedResponse{
		ID:          feed.ID,
		Title:       feed.Title,
		Description: feed.Description,
		URL:         feed.URL,
		LastUpdated: feed.LastUpdated,
		Items:       make([]responses.FeedItemResponse, 0, len(feed.Items)),
	}

	// Map items
	for _, item := range feed.Items {
		if itemResponse := ToFeedItemResponse(&item); itemResponse != nil {
			response.Items = append(response.Items, *itemResponse)
		}
	}

	return response
}

// ToFeedItemResponse converts a domain FeedItem to a FeedItemResponse DTO
func ToFeedItemResponse(item *domain.FeedItem) *responses.FeedItemResponse {
	if item == nil {
		return nil
	}

	return &responses.FeedItemResponse{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Link:        item.Link,
		Published:   item.Published,
		Author:      item.Author,
	}
}

// ToFeedResponses converts multiple domain Feeds to FeedResponse DTOs
func ToFeedResponses(feeds []*domain.Feed) []responses.FeedResponse {
	responses := make([]responses.FeedResponse, 0, len(feeds))
	
	for _, feed := range feeds {
		if response := ToFeedResponse(feed); response != nil {
			responses = append(responses, *response)
		}
	}
	
	return responses
}