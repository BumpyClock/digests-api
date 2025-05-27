// ABOUTME: Extended feed service with feature flag support
// ABOUTME: Demonstrates gradual rollout of new feed parser implementation

package feed

import (
	"context"
	"fmt"
	"io"
	"sync"

	"digests-app-api/core/domain"
	"digests-app-api/pkg/featureflags"
)

// ParseSingleFeedWithFlags parses a feed using feature flags to determine implementation
func (s *FeedService) ParseSingleFeedWithFlags(ctx context.Context, feedURL string) (*domain.Feed, error) {
	// Check if new parser is enabled
	if featureflags.IsEnabled(ctx, featureflags.NewFeedParser) {
		s.deps.Logger.Debug("Using new feed parser", map[string]interface{}{
			"url":     feedURL,
			"feature": "new_feed_parser",
		})
		return s.parseSingleFeedV2(ctx, feedURL)
	}
	
	// Fall back to original implementation
	s.deps.Logger.Debug("Using legacy feed parser", map[string]interface{}{
		"url":     feedURL,
		"feature": "legacy_parser",
	})
	return s.ParseSingleFeed(ctx, feedURL)
}

// parseSingleFeedV2 is the new implementation of feed parsing
// This demonstrates how we can gradually roll out a new parser
func (s *FeedService) parseSingleFeedV2(ctx context.Context, feedURL string) (*domain.Feed, error) {
	// This is a simulated "new" parser - in reality it would have different logic
	// For demo purposes, we'll use the same logic but add a marker
	feed, err := s.ParseSingleFeed(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	
	// Add a marker to indicate this was parsed with v2
	if feed != nil {
		// In a real implementation, this might include:
		// - Better error handling
		// - Support for more feed formats
		// - Performance improvements
		// - Additional metadata extraction
		feed.Description = "[v2] " + feed.Description
	}
	
	return feed, nil
}

// ParseFeedsWithFlags parses multiple feeds using feature flags
func (s *FeedService) ParseFeedsWithFlags(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	// Check if caching is enabled via feature flag
	if !featureflags.IsEnabled(ctx, featureflags.CacheEnabled) {
		s.deps.Logger.Info("Cache disabled by feature flag", nil)
		return s.parseFeedsWithoutCache(ctx, urls)
	}
	
	// Use regular implementation with caching
	return s.ParseFeeds(ctx, urls)
}

// parseFeedsWithoutCache bypasses cache when feature flag disables it
func (s *FeedService) parseFeedsWithoutCache(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	if len(urls) == 0 {
		return []*domain.Feed{}, nil
	}
	
	// Create a semaphore to limit concurrent requests
	sem := make(chan struct{}, 10)
	
	// Channel to collect results
	type result struct {
		feed *domain.Feed
		err  error
		idx  int
	}
	results := make(chan result, len(urls))
	
	// Parse feeds concurrently
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(idx int, feedURL string) {
			defer wg.Done()
			
			select {
			case <-ctx.Done():
				results <- result{err: ctx.Err(), idx: idx}
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			
			// Parse without checking cache
			feed, err := s.parseFeedFromURL(ctx, feedURL)
			results <- result{feed: feed, err: err, idx: idx}
		}(i, url)
	}
	
	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect results
	feeds := make([]*domain.Feed, len(urls))
	for res := range results {
		if res.err != nil {
			s.deps.Logger.Error("Failed to parse feed", map[string]interface{}{
				"url":   urls[res.idx],
				"error": res.err.Error(),
			})
			continue
		}
		feeds[res.idx] = res.feed
	}
	
	// Filter out nil feeds
	validFeeds := make([]*domain.Feed, 0, len(feeds))
	for _, feed := range feeds {
		if feed != nil {
			validFeeds = append(validFeeds, feed)
		}
	}
	
	return validFeeds, nil
}

// parseFeedFromURL fetches and parses a feed without caching
func (s *FeedService) parseFeedFromURL(ctx context.Context, feedURL string) (*domain.Feed, error) {
	// Fetch feed content
	resp, err := s.deps.HTTPClient.Get(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body().Close()
	
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse feed content
	return s.parseFeedContent(body, feedURL)
}