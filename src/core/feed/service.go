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
	"strconv"
	"strings"
	"sync"
	"time"
	
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/mmcdole/gofeed"
)

// FeedService handles feed parsing and management
type FeedService struct {
	deps              interfaces.Dependencies
	thumbnailColorSvc interfaces.ThumbnailColorService
}

// NewFeedService creates a new feed service instance
func NewFeedService(deps interfaces.Dependencies) *FeedService {
	return &FeedService{
		deps: deps,
	}
}

// SetThumbnailColorService sets the thumbnail color service
func (s *FeedService) SetThumbnailColorService(svc interfaces.ThumbnailColorService) {
	s.thumbnailColorSvc = svc
}

// ParseSingleFeed parses a feed from the given URL
func (s *FeedService) ParseSingleFeed(ctx context.Context, feedURL string) (*domain.Feed, error) {
	// Validate URL
	if feedURL == "" {
		return nil, errors.New("feed URL cannot be empty")
	}

	parsedURL, err := url.Parse(feedURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid URL format")
	}

	// Check cache first
	cachedFeed, err := s.getCachedFeed(ctx, feedURL)
	if err == nil && cachedFeed != nil {
		return cachedFeed, nil
	}

	// Check if we have HTTP client
	if s.deps.HTTPClient == nil {
		return nil, errors.New("HTTP client not configured")
	}

	// Fetch the feed
	resp, err := s.deps.HTTPClient.Get(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body().Close()

	// Check status code
	if resp.StatusCode() != 200 {
		return nil, errors.New("feed returned non-200 status code")
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body())
	if err != nil {
		return nil, err
	}

	// Parse feed content
	feed, err := s.parseFeedContent(bodyBytes, feedURL)
	if err != nil {
		return nil, err
	}

	// Cache the feed (ignore cache errors)
	_ = s.cacheFeed(ctx, feedURL, feed)

	return feed, nil
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

	// Set image and favicon
	if parsedFeed.Image != nil {
		feed.Image = parsedFeed.Image.URL
		feed.Favicon = parsedFeed.Image.URL // Use as favicon fallback
	}

	// Set categories
	if len(parsedFeed.Categories) > 0 {
		feed.Categories = strings.Join(parsedFeed.Categories, ", ")
	}

	// iTunes extensions for podcasts
	if parsedFeed.ITunesExt != nil {
		if parsedFeed.ITunesExt.Image != "" {
			feed.Image = parsedFeed.ITunesExt.Image
			feed.Favicon = parsedFeed.ITunesExt.Image
		}
		feed.Subtitle = parsedFeed.ITunesExt.Subtitle
	}

	// Published date
	if parsedFeed.PublishedParsed != nil {
		feed.Published = parsedFeed.PublishedParsed
	} else if parsedFeed.Published != "" {
		// Try to parse the string
		if t := parseTime(parsedFeed.Published); !t.IsZero() {
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
		Description: parseHTMLToText(item.Description),
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
		feedItem.Published = parseTime(item.Published)
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
		feedItem.Content = parseHTMLToText(item.Content)
		feedItem.ContentEncoded = item.Content
	} else if item.Description != "" {
		// If no content, use description
		feedItem.Content = parseHTMLToText(item.Description)
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
		feedItem.Duration = parseDurationToSeconds(item.ITunesExt.Duration)
		feedItem.Episode = parseIntOrZero(item.ITunesExt.Episode)
		feedItem.Season = parseIntOrZero(item.ITunesExt.Season)
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

// parseDurationToSeconds parses duration string to seconds
func parseDurationToSeconds(durationStr string) string {
	if durationStr == "" {
		return ""
	}

	// Try integer first (already in seconds)
	if seconds, err := strconv.Atoi(durationStr); err == nil {
		return strconv.Itoa(seconds)
	}

	// Parse HH:MM:SS or MM:SS format
	parts := strings.Split(durationStr, ":")
	var totalSeconds int
	
	switch len(parts) {
	case 3: // HH:MM:SS
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		totalSeconds = hours*3600 + minutes*60 + seconds
	case 2: // MM:SS
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		totalSeconds = minutes*60 + seconds
	default:
		return durationStr // Return as-is if can't parse
	}
	
	return strconv.Itoa(totalSeconds)
}

// parseHTMLToText extracts text content from HTML
func parseHTMLToText(html string) string {
	// Simple HTML tag removal - in production you'd use a proper HTML parser
	// This is a simplified version
	text := html
	// Remove script and style content
	text = strings.ReplaceAll(text, "<script>", "<script><!--")
	text = strings.ReplaceAll(text, "</script>", "--></script>")
	text = strings.ReplaceAll(text, "<style>", "<style><!--")
	text = strings.ReplaceAll(text, "</style>", "--></style>")
	
	// Remove HTML tags
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text, ">")
		if start < end && start >= 0 && end >= 0 {
			text = text[:start] + " " + text[end+1:]
		} else {
			break
		}
	}
	
	// Clean up whitespace
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&#8230;", "...")
	text = strings.ReplaceAll(text, "&#8217;", "'")
	text = strings.ReplaceAll(text, "&#8220;", "\"")
	text = strings.ReplaceAll(text, "&#8221;", "\"")
	text = strings.TrimSpace(text)
	
	return text
}

// parseTime attempts to parse time from various formats
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	
	// Try various formats
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC850,
		time.ANSIC,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z07:00",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}
	
	return time.Time{}
}

// detectFeedType determines if this is an article feed, podcast, or generic RSS
func detectFeedType(feed *gofeed.Feed) string {
	// Check for podcast indicators
	if feed.ITunesExt != nil {
		return "podcast"
	}
	
	// Check items for enclosures
	for _, item := range feed.Items {
		if len(item.Enclosures) > 0 {
			for _, enc := range item.Enclosures {
				if strings.HasPrefix(enc.Type, "audio/") || strings.HasPrefix(enc.Type, "video/") {
					return "podcast"
				}
			}
		}
	}
	
	// Check for news/article indicators
	if strings.Contains(strings.ToLower(feed.Title), "news") ||
		strings.Contains(strings.ToLower(feed.Description), "news") ||
		strings.Contains(strings.ToLower(feed.Title), "blog") {
		return "article"
	}
	
	return "rss"
}

// parseIntOrZero safely parses a string to int, returns 0 on error
func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
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

// ParseFeeds parses multiple feeds concurrently
func (s *FeedService) ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	if urls == nil {
		return nil, errors.New("urls cannot be nil")
	}

	if len(urls) == 0 {
		return []*domain.Feed{}, nil
	}

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
			feed, err := s.ParseSingleFeed(ctx, feedURL)
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