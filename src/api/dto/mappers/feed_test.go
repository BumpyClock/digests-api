package mappers

import (
	"testing"
	"time"

	"digests-app-api/core/domain"
)

func TestToFeedResponse(t *testing.T) {
	now := time.Now()
	feed := &domain.Feed{
		ID:          "feed-123",
		Title:       "Test Feed",
		Description: "Test Description",
		URL:         "https://example.com/feed.xml",
		Items: []domain.FeedItem{
			{
				ID:          "item-1",
				Title:       "Item 1",
				Description: "Description 1",
				Link:        "https://example.com/1",
				Published:   now.Add(-1 * time.Hour),
				Author:      "Author 1",
			},
			{
				ID:          "item-2",
				Title:       "Item 2",
				Description: "Description 2",
				Link:        "https://example.com/2",
				Published:   now.Add(-2 * time.Hour),
				Author:      "",
			},
		},
		LastUpdated: now,
	}

	response := ToFeedResponse(feed)

	// Test basic fields
	if response.ID != feed.ID {
		t.Errorf("ID = %s, want %s", response.ID, feed.ID)
	}

	if response.Title != feed.Title {
		t.Errorf("Title = %s, want %s", response.Title, feed.Title)
	}

	if response.Description != feed.Description {
		t.Errorf("Description = %s, want %s", response.Description, feed.Description)
	}

	if response.URL != feed.URL {
		t.Errorf("URL = %s, want %s", response.URL, feed.URL)
	}

	if !response.LastUpdated.Equal(feed.LastUpdated) {
		t.Errorf("LastUpdated = %v, want %v", response.LastUpdated, feed.LastUpdated)
	}

	// Test items mapping
	if len(response.Items) != len(feed.Items) {
		t.Fatalf("Items length = %d, want %d", len(response.Items), len(feed.Items))
	}

	// Verify first item
	if response.Items[0].ID != feed.Items[0].ID {
		t.Errorf("Items[0].ID = %s, want %s", response.Items[0].ID, feed.Items[0].ID)
	}

	if response.Items[0].Title != feed.Items[0].Title {
		t.Errorf("Items[0].Title = %s, want %s", response.Items[0].Title, feed.Items[0].Title)
	}

	if response.Items[0].Author != feed.Items[0].Author {
		t.Errorf("Items[0].Author = %s, want %s", response.Items[0].Author, feed.Items[0].Author)
	}
}

func TestToFeedResponse_NilFeed(t *testing.T) {
	response := ToFeedResponse(nil)

	if response != nil {
		t.Error("ToFeedResponse should return nil for nil feed")
	}
}

func TestToFeedResponse_EmptyItems(t *testing.T) {
	feed := &domain.Feed{
		ID:          "feed-123",
		Title:       "Test Feed",
		Description: "Test Description",
		URL:         "https://example.com/feed.xml",
		Items:       []domain.FeedItem{},
		LastUpdated: time.Now(),
	}

	response := ToFeedResponse(feed)

	if response.Items == nil {
		t.Error("Items should not be nil")
	}

	if len(response.Items) != 0 {
		t.Errorf("Items length = %d, want 0", len(response.Items))
	}
}

func TestToFeedItemResponse(t *testing.T) {
	published := time.Now().Add(-1 * time.Hour)
	item := &domain.FeedItem{
		ID:          "item-123",
		Title:       "Test Item",
		Description: "Test Description",
		Link:        "https://example.com/item",
		Published:   published,
		Author:      "Test Author",
	}

	response := ToFeedItemResponse(item)

	if response.ID != item.ID {
		t.Errorf("ID = %s, want %s", response.ID, item.ID)
	}

	if response.Title != item.Title {
		t.Errorf("Title = %s, want %s", response.Title, item.Title)
	}

	if response.Description != item.Description {
		t.Errorf("Description = %s, want %s", response.Description, item.Description)
	}

	if response.Link != item.Link {
		t.Errorf("Link = %s, want %s", response.Link, item.Link)
	}

	if !response.Published.Equal(item.Published) {
		t.Errorf("Published = %v, want %v", response.Published, item.Published)
	}

	if response.Author != item.Author {
		t.Errorf("Author = %s, want %s", response.Author, item.Author)
	}
}

func TestToFeedItemResponse_NilItem(t *testing.T) {
	response := ToFeedItemResponse(nil)

	if response != nil {
		t.Error("ToFeedItemResponse should return nil for nil item")
	}
}

func TestToFeedItemResponse_EmptyAuthor(t *testing.T) {
	item := &domain.FeedItem{
		ID:          "item-123",
		Title:       "Test Item",
		Description: "Test Description",
		Link:        "https://example.com/item",
		Published:   time.Now(),
		Author:      "", // Empty author
	}

	response := ToFeedItemResponse(item)

	if response.Author != "" {
		t.Errorf("Author = %s, want empty string", response.Author)
	}
}