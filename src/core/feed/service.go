// ABOUTME: Feed service handles RSS/Atom feed parsing and caching
// ABOUTME: Provides business logic for feed operations independent of HTTP layer

package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
	
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"digests-app-api/pkg/utils/duration"
	"digests-app-api/pkg/utils/html"
	"digests-app-api/pkg/utils/parse"
	utiltime "digests-app-api/pkg/utils/time"
	"github.com/mmcdole/gofeed"
)

// FeedService handles feed parsing and management
type FeedService struct {
	deps interfaces.Dependencies
}

// NewFeedService creates a new feed service instance
func NewFeedService(deps interfaces.Dependencies) *FeedService {
	return &FeedService{
		deps: deps,
	}
}

// ParseFeeds parses one or more feeds concurrently
func (s *FeedService) ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	if len(urls) == 0 {
		return []*domain.Feed{}, nil
	}

	// Single URL - parse directly without concurrency overhead
	if len(urls) == 1 {
		feed, err := s.parseSingleFeed(ctx, urls[0])
		if err != nil {
			return nil, err
		}
		return []*domain.Feed{feed}, nil
	}

	// Multiple URLs - parse concurrently
	return s.parseFeedsConcurrently(ctx, urls)
}

// parseSingleFeed handles parsing of a single feed with caching
func (s *FeedService) parseSingleFeed(ctx context.Context, feedURL string) (*domain.Feed, error) {
	// Validate URL
	if feedURL == "" {
		return nil, errors.New("feed URL cannot be empty")
	}

	parsedURL, err := url.Parse(feedURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid URL format: %s", feedURL)
	}

	// Check cache first
	cachedFeed, err := s.getCachedFeed(ctx, feedURL)
	if err == nil && cachedFeed != nil {
		return cachedFeed, nil
	}

	// Fetch and parse the feed
	feed, err := s.fetchAndParseFeed(ctx, feedURL)
	if err != nil {
		return nil, err
	}

	// Cache the feed (ignore cache errors)
	_ = s.cacheFeed(ctx, feedURL, feed)

	return feed, nil
}

// fetchAndParseFeed fetches a feed from URL and parses it
func (s *FeedService) fetchAndParseFeed(ctx context.Context, feedURL string) (*domain.Feed, error) {
	// Check if we have HTTP client
	if s.deps.HTTPClient == nil {
		return nil, errors.New("HTTP client not configured")
	}

	// Fetch the feed
	resp, err := s.deps.HTTPClient.Get(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body().Close()

	// Check status code
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("feed returned status code %d", resp.StatusCode())
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse feed content
	return s.parseFeedContent(bodyBytes, feedURL)
}

// ParseSingleFeed is kept for backward compatibility but delegates to ParseFeeds
func (s *FeedService) ParseSingleFeed(ctx context.Context, feedURL string) (*domain.Feed, error) {
	feeds, err := s.ParseFeeds(ctx, []string{feedURL})
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return nil, errors.New("no feed returned")
	}
	return feeds[0], nil
}

// ParseFeedsWithConfig parses multiple RSS/Atom feeds with enrichment configuration
func (s *FeedService) ParseFeedsWithConfig(ctx context.Context, urls []string, config interface{}) ([]*domain.Feed, error) {
	// For now, just delegate to ParseFeeds
	// The actual enrichment configuration will be handled at the API layer
	// This method exists to satisfy the interface and prepare for future enhancements
	return s.ParseFeeds(ctx, urls)
}

// parseFeedContent parses feed content from bytes
func (s *FeedService) parseFeedContent(content []byte, feedURL string) (*domain.Feed, error) {
	if len(content) == 0 {
		return nil, errors.New("empty feed content")
	}

	// Parse using gofeed
	parser := gofeed.NewParser()
	parsedFeed, err := parser.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	// Convert to domain model
	feed := &domain.Feed{
		ID:          parsedFeed.Link, // Use feed link as ID for now
		Title:       parsedFeed.Title,
		Description: parsedFeed.Description,
		URL:         feedURL,         // Keep the actual RSS URL
		Link:        parsedFeed.Link, // Website link
		Items:       make([]domain.FeedItem, 0, len(parsedFeed.Items)),
		Language:    parsedFeed.Language,
		FeedType:    detectFeedType(parsedFeed),
	}

	// Set last updated time
	if parsedFeed.UpdatedParsed != nil {
		feed.LastUpdated = *parsedFeed.UpdatedParsed
	} else if parsedFeed.PublishedParsed != nil {
		feed.LastUpdated = *parsedFeed.PublishedParsed
	} else {
		feed.LastUpdated = time.Now()
	}

	// Set author
	if parsedFeed.Author != nil {
		feed.Author = &domain.Author{
			Name:  parsedFeed.Author.Name,
			Email: parsedFeed.Author.Email,
		}
	}
	
	// iTunes extensions take precedence for author
	if parsedFeed.ITunesExt != nil && parsedFeed.ITunesExt.Author != "" {
		feed.Author = &domain.Author{
			Name: parsedFeed.ITunesExt.Author,
		}
	}

	// Set image
	if parsedFeed.Image != nil {
		feed.Image = parsedFeed.Image.URL
	}

	// Set categories
	if len(parsedFeed.Categories) > 0 {
		feed.Categories = strings.Join(parsedFeed.Categories, ", ")
	}

	// iTunes extensions for podcasts
	if parsedFeed.ITunesExt != nil {
		if parsedFeed.ITunesExt.Image != "" {
			feed.Image = parsedFeed.ITunesExt.Image
		}
		feed.Subtitle = parsedFeed.ITunesExt.Subtitle
	}

	// Published date
	if parsedFeed.PublishedParsed != nil {
		feed.Published = parsedFeed.PublishedParsed
	} else if parsedFeed.Published != "" {
		// Try to parse the string
		if t := utiltime.ParseFlexibleTime(parsedFeed.Published); !t.IsZero() {
			feed.Published = &t
		}
	}

	// Convert items
	for _, item := range parsedFeed.Items {
		feedItem := s.convertItemToDomain(item, parsedFeed)
		feed.Items = append(feed.Items, feedItem)
	}

	return feed, nil
}

// convertItemToDomain converts a gofeed item to domain item
func (s *FeedService) convertItemToDomain(item *gofeed.Item, feed *gofeed.Feed) domain.FeedItem {
	feedItem := domain.FeedItem{
		ID:          item.GUID,
		Title:       item.Title,
		Description: html.StripHTML(item.Description),
		Link:        item.Link,
	}

	// Use GUID or create hash from link
	if feedItem.ID == "" && item.Link != "" {
		feedItem.ID = item.Link
	}

	// Set published time
	if item.PublishedParsed != nil {
		feedItem.Published = *item.PublishedParsed
	} else if item.Published != "" {
		feedItem.Published = utiltime.ParseFlexibleTime(item.Published)
	}

	// Set created time (same as published in old implementation)
	feedItem.Created = &feedItem.Published

	// Set author
	if item.ITunesExt != nil && item.ITunesExt.Author != "" {
		feedItem.Author = item.ITunesExt.Author
	} else if item.Author != nil && item.Author.Name != "" {
		feedItem.Author = item.Author.Name
	}

	// Set categories as comma-separated string
	if len(item.Categories) > 0 {
		feedItem.Categories = item.Categories
	}

	// Content handling - parse HTML for plain text, keep encoded version
	if item.Content != "" {
		feedItem.Content = html.StripHTML(item.Content)
		feedItem.ContentEncoded = item.Content
	} else if item.Description != "" {
		// If no content, use description
		feedItem.Content = html.StripHTML(item.Description)
		feedItem.ContentEncoded = item.Description
	}

	// Set enclosures
	for _, enc := range item.Enclosures {
		feedItem.Enclosures = append(feedItem.Enclosures, domain.Enclosure{
			URL:    enc.URL,
			Length: enc.Length,
			Type:   enc.Type,
		})
	}

	// Thumbnail discovery logic (following old implementation)
	feedItem.Thumbnail = s.findThumbnail(item, feed)

	// iTunes extensions
	if item.ITunesExt != nil {
		// Parse duration to seconds
		feedItem.Duration = duration.ParseToSeconds(item.ITunesExt.Duration)
		feedItem.Episode = parse.IntOrZero(item.ITunesExt.Episode)
		feedItem.Season = parse.IntOrZero(item.ITunesExt.Season)
		feedItem.EpisodeType = item.ITunesExt.EpisodeType
		feedItem.Subtitle = item.ITunesExt.Subtitle
		feedItem.Summary = item.ITunesExt.Summary
		if item.ITunesExt.Image != "" {
			feedItem.Image = item.ITunesExt.Image
			// Override thumbnail with iTunes image
			feedItem.Thumbnail = item.ITunesExt.Image
		}
	}

	return feedItem
}

// findThumbnail finds thumbnail from various sources
func (s *FeedService) findThumbnail(item *gofeed.Item, feed *gofeed.Feed) string {
	// Priority order matching old implementation:
	// 1. iTunes extension image
	if item.ITunesExt != nil && item.ITunesExt.Image != "" {
		return item.ITunesExt.Image
	}
	
	// 2. Image enclosures
	for _, enc := range item.Enclosures {
		if enc.URL != "" && strings.HasPrefix(enc.Type, "image/") {
			return enc.URL
		}
	}
	
	// 3. Item image
	if item.Image != nil && item.Image.URL != "" {
		return item.Image.URL
	}
	
	// 4. Feed level iTunes image
	if feed != nil && feed.ITunesExt != nil && feed.ITunesExt.Image != "" {
		return feed.ITunesExt.Image
	}
	
	// 5. Feed image
	if feed != nil && feed.Image != nil && feed.Image.URL != "" {
		return feed.Image.URL
	}
	
	return ""
}

// detectFeedType determines if this is an article feed or podcast
func detectFeedType(feed *gofeed.Feed) string {
	// Quick check: iTunes extension at feed level = podcast
	if feed.ITunesExt != nil {
		return "podcast"
	}
	
	// Check first few items for podcast indicators
	const maxItemsToCheck = 3
	itemsToCheck := len(feed.Items)
	if itemsToCheck > maxItemsToCheck {
		itemsToCheck = maxItemsToCheck
	}
	
	for i := 0; i < itemsToCheck; i++ {
		item := feed.Items[i]
		
		// Item has iTunes extension = podcast
		if item.ITunesExt != nil {
			return "podcast"
		}
		
		// Check for audio/video enclosures
		for _, enc := range item.Enclosures {
			if strings.HasPrefix(enc.Type, "audio/") || strings.HasPrefix(enc.Type, "video/") {
				return "podcast"
			}
		}
	}
	
	// Everything else is an article feed
	return "article"
}

// getCachedFeed retrieves a feed from cache
func (s *FeedService) getCachedFeed(ctx context.Context, feedURL string) (*domain.Feed, error) {
	if s.deps.Cache == nil {
		return nil, nil // No cache configured
	}

	key := fmt.Sprintf("feed:%s", feedURL)
	data, err := s.deps.Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, nil // Cache miss
	}

	// Deserialize feed
	var feed domain.Feed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

// cacheFeed stores a feed in cache
func (s *FeedService) cacheFeed(ctx context.Context, feedURL string, feed *domain.Feed) error {
	if s.deps.Cache == nil {
		return nil // No cache configured
	}

	// Serialize feed
	data, err := json.Marshal(feed)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("feed:%s", feedURL)
	return s.deps.Cache.Set(ctx, key, data, 1*time.Hour)
}

// parseFeedsConcurrently handles concurrent parsing of multiple feeds
func (s *FeedService) parseFeedsConcurrently(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	// Create channels for results
	type feedResult struct {
		feed *domain.Feed
		err  error
		url  string
	}

	resultsChan := make(chan feedResult, len(urls))
	
	// Use semaphore to limit concurrent operations to 10
	semaphore := make(chan struct{}, 10)
	
	// WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Launch goroutines for each URL
	for _, url := range urls {
		wg.Add(1)
		go func(feedURL string) {
			defer wg.Done()

			// Check for context cancellation
			select {
			case <-ctx.Done():
				resultsChan <- feedResult{url: feedURL, err: ctx.Err()}
				return
			default:
			}

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Parse the feed
			feed, err := s.parseSingleFeed(ctx, feedURL)
			resultsChan <- feedResult{
				feed: feed,
				err:  err,
				url:  feedURL,
			}
		}(url)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	feeds := make([]*domain.Feed, 0, len(urls))
	var firstError error

	for result := range resultsChan {
		if result.err != nil {
			// Log error but continue processing other feeds
			if s.deps.Logger != nil {
				s.deps.Logger.Error("Failed to parse feed", map[string]interface{}{
					"url":   result.url,
					"error": result.err.Error(),
				})
			}
			// Capture first error for context cancellation
			if firstError == nil && errors.Is(result.err, context.Canceled) {
				firstError = result.err
			}
			continue
		}
		if result.feed != nil {
			feeds = append(feeds, result.feed)
		}
	}

	// Return context cancellation error if that was the cause
	if firstError != nil {
		return feeds, firstError
	}

	return feeds, nil
}