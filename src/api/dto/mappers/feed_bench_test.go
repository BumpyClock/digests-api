package mappers

import (
	"fmt"
	"testing"
	"time"

	"digests-app-api/core/domain"
)

func BenchmarkToFeedResponse(b *testing.B) {
	// Create a sample feed with many items
	feed := &domain.Feed{
		ID:          "feed-123",
		Title:       "Benchmark Feed",
		Description: "A feed for benchmarking DTO mapping performance",
		URL:         "http://example.com/feed.rss",
		Items:       make([]domain.FeedItem, 100),
		LastUpdated: time.Now(),
	}
	
	for i := 0; i < 100; i++ {
		feed.Items[i] = domain.FeedItem{
			ID:          fmt.Sprintf("item-%d", i),
			Title:       fmt.Sprintf("Item %d", i),
			Description: fmt.Sprintf("Description for item %d with some longer text to simulate real content", i),
			Link:        fmt.Sprintf("http://example.com/item-%d", i),
			Author:      fmt.Sprintf("Author %d", i),
			Published:   time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToFeedResponse(feed)
	}
}

func BenchmarkToFeedItemResponse(b *testing.B) {
	item := &domain.FeedItem{
		ID:          "item-123",
		Title:       "Benchmark Item",
		Description: "A longer description with some content to simulate real feed items that might contain multiple sentences and paragraphs.",
		Link:        "http://example.com/item-123",
		Author:      "Benchmark Author",
		Published:   time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToFeedItemResponse(item)
	}
}

func BenchmarkToFeedResponses_Small(b *testing.B) {
	// Benchmark with 10 feeds
	feeds := make([]*domain.Feed, 10)
	for i := 0; i < 10; i++ {
		feeds[i] = &domain.Feed{
			ID:          fmt.Sprintf("feed-%d", i),
			Title:       fmt.Sprintf("Feed %d", i),
			Description: fmt.Sprintf("Description for feed %d", i),
			URL:         fmt.Sprintf("http://example.com/feed-%d.rss", i),
			Items:       make([]domain.FeedItem, 10),
			LastUpdated: time.Now(),
		}
		
		for j := 0; j < 10; j++ {
			feeds[i].Items[j] = domain.FeedItem{
				ID:          fmt.Sprintf("item-%d-%d", i, j),
				Title:       fmt.Sprintf("Item %d-%d", i, j),
				Description: fmt.Sprintf("Description %d-%d", i, j),
				Link:        fmt.Sprintf("http://example.com/item-%d-%d", i, j),
				Published:   time.Now(),
			}
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToFeedResponses(feeds)
	}
}

func BenchmarkToFeedResponses_Large(b *testing.B) {
	// Benchmark with 100 feeds
	feeds := make([]*domain.Feed, 100)
	for i := 0; i < 100; i++ {
		feeds[i] = &domain.Feed{
			ID:          fmt.Sprintf("feed-%d", i),
			Title:       fmt.Sprintf("Feed %d", i),
			Description: fmt.Sprintf("Description for feed %d", i),
			URL:         fmt.Sprintf("http://example.com/feed-%d.rss", i),
			Items:       make([]domain.FeedItem, 20),
			LastUpdated: time.Now(),
		}
		
		for j := 0; j < 20; j++ {
			feeds[i].Items[j] = domain.FeedItem{
				ID:          fmt.Sprintf("item-%d-%d", i, j),
				Title:       fmt.Sprintf("Item %d-%d", i, j),
				Description: fmt.Sprintf("Description %d-%d", i, j),
				Link:        fmt.Sprintf("http://example.com/item-%d-%d", i, j),
				Published:   time.Now(),
			}
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToFeedResponses(feeds)
	}
}

// BenchmarkMemoryAllocation tests memory allocations during mapping
func BenchmarkMemoryAllocation(b *testing.B) {
	feed := &domain.Feed{
		ID:          "feed-123",
		Title:       "Benchmark Feed",
		Description: "Testing memory allocations",
		URL:         "http://example.com/feed.rss",
		Items:       make([]domain.FeedItem, 50),
		LastUpdated: time.Now(),
	}
	
	for i := 0; i < 50; i++ {
		feed.Items[i] = domain.FeedItem{
			ID:          fmt.Sprintf("item-%d", i),
			Title:       fmt.Sprintf("Item %d", i),
			Description: fmt.Sprintf("Description %d", i),
			Link:        fmt.Sprintf("http://example.com/item-%d", i),
			Published:   time.Now(),
		}
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToFeedResponse(feed)
	}
}