package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
)

func TestNewFeedService(t *testing.T) {
	deps := interfaces.Dependencies{}
	
	service := NewFeedService(deps)
	
	if service == nil {
		t.Error("NewFeedService returned nil")
	}
}

func TestNewFeedService_StoresDependencies(t *testing.T) {
	// Create mock dependencies
	deps := interfaces.Dependencies{
		Cache:      nil, // We'll implement mock later
		HTTPClient: nil, // We'll implement mock later
		Logger:     nil, // We'll implement mock later
	}
	
	service := NewFeedService(deps)
	
	if service.deps.Cache != deps.Cache {
		t.Error("NewFeedService did not store Cache dependency")
	}
	if service.deps.HTTPClient != deps.HTTPClient {
		t.Error("NewFeedService did not store HTTPClient dependency")
	}
	if service.deps.Logger != deps.Logger {
		t.Error("NewFeedService did not store Logger dependency")
	}
}

func TestParseSingleFeed_EmptyURL(t *testing.T) {
	deps := interfaces.Dependencies{}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feed, err := service.ParseSingleFeed(ctx, "")
	
	if err == nil {
		t.Error("ParseSingleFeed should return error for empty URL")
	}
	if feed != nil {
		t.Error("ParseSingleFeed should return nil feed for empty URL")
	}
}

func TestParseSingleFeed_InvalidURL(t *testing.T) {
	deps := interfaces.Dependencies{}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feed, err := service.ParseSingleFeed(ctx, "not a valid url")
	
	if err == nil {
		t.Error("ParseSingleFeed should return error for invalid URL")
	}
	if feed != nil {
		t.Error("ParseSingleFeed should return nil feed for invalid URL")
	}
}

func TestParseSingleFeed_CallsHTTPClient(t *testing.T) {
	called := false
	expectedURL := "https://example.com/feed.xml"
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			called = true
			if url != expectedURL {
				t.Errorf("HTTPClient.Get called with wrong URL: got %v, want %v", url, expectedURL)
			}
			return &mockResponse{
				statusCode: 200,
				body:       "",
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	service.ParseSingleFeed(ctx, expectedURL)
	
	if !called {
		t.Error("ParseSingleFeed should call HTTPClient.Get")
	}
}

func TestParseSingleFeed_HTTPGetError(t *testing.T) {
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			return nil, errors.New("network error")
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feed, err := service.ParseSingleFeed(ctx, "https://example.com/feed.xml")
	
	if err == nil {
		t.Error("ParseSingleFeed should return error when HTTP GET fails")
	}
	if feed != nil {
		t.Error("ParseSingleFeed should return nil feed when HTTP GET fails")
	}
}

func TestParseSingleFeed_Non200StatusCode(t *testing.T) {
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			return &mockResponse{
				statusCode: 404,
				body:       "Not Found",
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feed, err := service.ParseSingleFeed(ctx, "https://example.com/feed.xml")
	
	if err == nil {
		t.Error("ParseSingleFeed should return error for non-200 status code")
	}
	if feed != nil {
		t.Error("ParseSingleFeed should return nil feed for non-200 status code")
	}
}

func TestParseFeedContent_EmptyContent(t *testing.T) {
	deps := interfaces.Dependencies{}
	service := &FeedService{deps: deps}
	
	feed, err := service.parseFeedContent([]byte{})
	
	if err == nil {
		t.Error("parseFeedContent should return error for empty content")
	}
	if feed != nil {
		t.Error("parseFeedContent should return nil feed for empty content")
	}
}

func TestParseFeedContent_ValidRSS(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <description>Test Description</description>
    <link>https://example.com</link>
    <item>
      <title>Test Item</title>
      <description>Test Item Description</description>
      <link>https://example.com/item1</link>
      <pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
    </item>
  </channel>
</rss>`
	
	deps := interfaces.Dependencies{}
	service := &FeedService{deps: deps}
	
	feed, err := service.parseFeedContent([]byte(rssContent))
	
	if err != nil {
		t.Errorf("parseFeedContent returned error: %v", err)
	}
	if feed == nil {
		t.Fatal("parseFeedContent returned nil feed")
	}
	if feed.Title != "Test Feed" {
		t.Errorf("Feed title = %v, want %v", feed.Title, "Test Feed")
	}
	if len(feed.Items) != 1 {
		t.Errorf("Feed items count = %v, want %v", len(feed.Items), 1)
	}
}

func TestParseFeedContent_ValidAtom(t *testing.T) {
	atomContent := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <subtitle>Test Atom Description</subtitle>
  <link href="https://example.com/"/>
  <entry>
    <title>Test Atom Entry</title>
    <link href="https://example.com/entry1"/>
    <summary>Test Entry Summary</summary>
    <published>2006-01-02T15:04:05Z</published>
  </entry>
</feed>`
	
	deps := interfaces.Dependencies{}
	service := &FeedService{deps: deps}
	
	feed, err := service.parseFeedContent([]byte(atomContent))
	
	if err != nil {
		t.Errorf("parseFeedContent returned error: %v", err)
	}
	if feed == nil {
		t.Fatal("parseFeedContent returned nil feed")
	}
	if feed.Title != "Test Atom Feed" {
		t.Errorf("Feed title = %v, want %v", feed.Title, "Test Atom Feed")
	}
	if len(feed.Items) != 1 {
		t.Errorf("Feed items count = %v, want %v", len(feed.Items), 1)
	}
}

func TestParseFeedContent_InvalidXML(t *testing.T) {
	invalidContent := `not valid xml`
	
	deps := interfaces.Dependencies{}
	service := &FeedService{deps: deps}
	
	feed, err := service.parseFeedContent([]byte(invalidContent))
	
	if err == nil {
		t.Error("parseFeedContent should return error for invalid XML")
	}
	if feed != nil {
		t.Error("parseFeedContent should return nil feed for invalid XML")
	}
}

func TestGetCachedFeed_CacheMiss(t *testing.T) {
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return nil, nil // Cache miss
		},
	}
	
	deps := interfaces.Dependencies{
		Cache: mockCache,
	}
	service := &FeedService{deps: deps}
	
	ctx := context.Background()
	feed, err := service.getCachedFeed(ctx, "https://example.com/feed.xml")
	
	if err != nil {
		t.Errorf("getCachedFeed returned error on cache miss: %v", err)
	}
	if feed != nil {
		t.Error("getCachedFeed should return nil feed on cache miss")
	}
}

func TestGetCachedFeed_CacheHit(t *testing.T) {
	expectedFeed := &domain.Feed{
		ID:          "test-id",
		Title:       "Cached Feed",
		Description: "Cached Description",
		URL:         "https://example.com/feed.xml",
		Items:       []domain.FeedItem{},
	}
	
	// Serialize the feed for cache
	feedJSON, _ := json.Marshal(expectedFeed)
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return feedJSON, nil // Cache hit
		},
	}
	
	deps := interfaces.Dependencies{
		Cache: mockCache,
	}
	service := &FeedService{deps: deps}
	
	ctx := context.Background()
	feed, err := service.getCachedFeed(ctx, "https://example.com/feed.xml")
	
	if err != nil {
		t.Errorf("getCachedFeed returned error on cache hit: %v", err)
	}
	if feed == nil {
		t.Fatal("getCachedFeed returned nil feed on cache hit")
	}
	if feed.Title != expectedFeed.Title {
		t.Errorf("Feed title = %v, want %v", feed.Title, expectedFeed.Title)
	}
}

func TestCacheFeed(t *testing.T) {
	var capturedKey string
	var capturedValue []byte
	var capturedTTL time.Duration
	
	mockCache := &mockCache{
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			capturedKey = key
			capturedValue = value
			capturedTTL = ttl
			return nil
		},
	}
	
	deps := interfaces.Dependencies{
		Cache: mockCache,
	}
	service := &FeedService{deps: deps}
	
	feed := &domain.Feed{
		ID:          "test-id",
		Title:       "Test Feed",
		Description: "Test Description",
		URL:         "https://example.com/feed.xml",
		Items:       []domain.FeedItem{},
	}
	
	ctx := context.Background()
	err := service.cacheFeed(ctx, "https://example.com/feed.xml", feed)
	
	if err != nil {
		t.Errorf("cacheFeed returned error: %v", err)
	}
	
	// Verify cache key
	expectedKey := "feed:https://example.com/feed.xml"
	if capturedKey != expectedKey {
		t.Errorf("Cache key = %v, want %v", capturedKey, expectedKey)
	}
	
	// Verify TTL is 1 hour
	if capturedTTL != 1*time.Hour {
		t.Errorf("Cache TTL = %v, want %v", capturedTTL, 1*time.Hour)
	}
	
	// Verify serialized data
	var cachedFeed domain.Feed
	if err := json.Unmarshal(capturedValue, &cachedFeed); err != nil {
		t.Errorf("Failed to unmarshal cached data: %v", err)
	}
	if cachedFeed.Title != feed.Title {
		t.Errorf("Cached feed title = %v, want %v", cachedFeed.Title, feed.Title)
	}
}

func TestParseSingleFeed_ChecksCacheFirst(t *testing.T) {
	cachedFeed := &domain.Feed{
		ID:          "cached-id",
		Title:       "Cached Feed",
		Description: "From Cache",
		URL:         "https://example.com/feed.xml",
		Items:       []domain.FeedItem{},
	}
	
	feedJSON, _ := json.Marshal(cachedFeed)
	httpClientCalled := false
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return feedJSON, nil // Cache hit
		},
	}
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			httpClientCalled = true
			return nil, errors.New("should not be called")
		},
	}
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feed, err := service.ParseSingleFeed(ctx, "https://example.com/feed.xml")
	
	if err != nil {
		t.Errorf("ParseSingleFeed returned error: %v", err)
	}
	if feed == nil {
		t.Fatal("ParseSingleFeed returned nil feed")
	}
	if feed.Title != cachedFeed.Title {
		t.Errorf("Feed title = %v, want %v", feed.Title, cachedFeed.Title)
	}
	if httpClientCalled {
		t.Error("HTTP client should not be called when cache hit")
	}
}

func TestParseSingleFeed_CachesSuccessfulResult(t *testing.T) {
	feedURL := "https://example.com/feed.xml"
	cacheCalled := false
	
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
  </channel>
</rss>`
	
	mockCache := &mockCache{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return nil, nil // Cache miss
		},
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			cacheCalled = true
			expectedKey := "feed:" + feedURL
			if key != expectedKey {
				t.Errorf("Cache key = %v, want %v", key, expectedKey)
			}
			if ttl != 1*time.Hour {
				t.Errorf("Cache TTL = %v, want %v", ttl, 1*time.Hour)
			}
			return nil
		},
	}
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			return &mockResponse{
				statusCode: 200,
				body:       rssContent,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feed, err := service.ParseSingleFeed(ctx, feedURL)
	
	if err != nil {
		t.Errorf("ParseSingleFeed returned error: %v", err)
	}
	if feed == nil {
		t.Fatal("ParseSingleFeed returned nil feed")
	}
	if !cacheCalled {
		t.Error("ParseSingleFeed should cache successful result")
	}
}

func TestParseFeeds_NilURLs(t *testing.T) {
	deps := interfaces.Dependencies{}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feeds, err := service.ParseFeeds(ctx, nil)
	
	if err == nil {
		t.Error("ParseFeeds should return error for nil urls")
	}
	if feeds != nil {
		t.Error("ParseFeeds should return nil feeds for nil urls")
	}
}

func TestParseFeeds_EmptyURLs(t *testing.T) {
	deps := interfaces.Dependencies{}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feeds, err := service.ParseFeeds(ctx, []string{})
	
	if err != nil {
		t.Errorf("ParseFeeds returned error for empty urls: %v", err)
	}
	if feeds == nil {
		t.Error("ParseFeeds should return empty slice, not nil")
	}
	if len(feeds) != 0 {
		t.Errorf("ParseFeeds returned %d feeds, want 0", len(feeds))
	}
}

func TestParseFeeds_SingleURL(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
  </channel>
</rss>`
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			return &mockResponse{
				statusCode: 200,
				body:       rssContent,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	ctx := context.Background()
	feeds, err := service.ParseFeeds(ctx, []string{"https://example.com/feed.xml"})
	
	if err != nil {
		t.Errorf("ParseFeeds returned error: %v", err)
	}
	if len(feeds) != 1 {
		t.Errorf("ParseFeeds returned %d feeds, want 1", len(feeds))
	}
	if feeds[0].Title != "Test Feed" {
		t.Errorf("Feed title = %v, want %v", feeds[0].Title, "Test Feed")
	}
}

func TestParseFeeds_MultipleURLs(t *testing.T) {
	callCount := 0
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			callCount++
			title := fmt.Sprintf("Feed %d", callCount)
			rssContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>%s</title>
    <link>%s</link>
  </channel>
</rss>`, title, url)
			
			return &mockResponse{
				statusCode: 200,
				body:       rssContent,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	urls := []string{
		"https://example.com/feed1.xml",
		"https://example.com/feed2.xml",
		"https://example.com/feed3.xml",
	}
	
	ctx := context.Background()
	feeds, err := service.ParseFeeds(ctx, urls)
	
	if err != nil {
		t.Errorf("ParseFeeds returned error: %v", err)
	}
	if len(feeds) != 3 {
		t.Errorf("ParseFeeds returned %d feeds, want 3", len(feeds))
	}
}

func TestParseFeeds_ContinuesOnError(t *testing.T) {
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			if url == "https://example.com/fail.xml" {
				return nil, errors.New("network error")
			}
			
			rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Success Feed</title>
    <link>https://example.com</link>
  </channel>
</rss>`
			return &mockResponse{
				statusCode: 200,
				body:       rssContent,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	urls := []string{
		"https://example.com/feed1.xml",
		"https://example.com/fail.xml",
		"https://example.com/feed2.xml",
	}
	
	ctx := context.Background()
	feeds, err := service.ParseFeeds(ctx, urls)
	
	if err != nil {
		t.Errorf("ParseFeeds returned error: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("ParseFeeds returned %d feeds, want 2 (continuing on error)", len(feeds))
	}
}

func TestParseFeeds_RespectsContextCancellation(t *testing.T) {
	slowClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return &mockResponse{statusCode: 200, body: ""}, nil
			}
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: slowClient,
	}
	service := NewFeedService(deps)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel context immediately
	cancel()
	
	urls := []string{"https://example.com/feed1.xml", "https://example.com/feed2.xml"}
	feeds, err := service.ParseFeeds(ctx, urls)
	
	if err == nil {
		t.Error("ParseFeeds should return error for cancelled context")
	}
	if len(feeds) != 0 {
		t.Errorf("ParseFeeds returned %d feeds, want 0 for cancelled context", len(feeds))
	}
}

func TestParseFeeds_LimitsConcurrency(t *testing.T) {
	// Track concurrent requests
	var concurrent int32
	var maxConcurrent int32
	var mu sync.Mutex
	
	mockClient := &mockHTTPClient{
		getFunc: func(ctx context.Context, url string) (interfaces.Response, error) {
			// Increment concurrent counter
			mu.Lock()
			concurrent++
			if concurrent > maxConcurrent {
				maxConcurrent = concurrent
			}
			mu.Unlock()
			
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			
			// Decrement concurrent counter
			mu.Lock()
			concurrent--
			mu.Unlock()
			
			return &mockResponse{
				statusCode: 200,
				body: `<rss version="2.0"><channel><title>Test</title></channel></rss>`,
			}, nil
		},
	}
	
	deps := interfaces.Dependencies{
		HTTPClient: mockClient,
	}
	service := NewFeedService(deps)
	
	// Create 20 URLs
	urls := make([]string, 20)
	for i := 0; i < 20; i++ {
		urls[i] = fmt.Sprintf("https://example.com/feed%d.xml", i)
	}
	
	ctx := context.Background()
	feeds, err := service.ParseFeeds(ctx, urls)
	
	if err != nil {
		t.Errorf("ParseFeeds returned error: %v", err)
	}
	if len(feeds) != 20 {
		t.Errorf("ParseFeeds returned %d feeds, want 20", len(feeds))
	}
	
	// Verify max concurrent was <= 10
	if maxConcurrent > 10 {
		t.Errorf("Max concurrent requests = %d, want <= 10", maxConcurrent)
	}
}