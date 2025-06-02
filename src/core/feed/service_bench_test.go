package feed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
)

// Mock implementations for benchmarking
type benchMockCache struct{}

func (m *benchMockCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil // Cache miss
}

func (m *benchMockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *benchMockCache) Delete(ctx context.Context, key string) error {
	return nil
}

type benchMockHTTPClient struct {
	responseTime time.Duration
}

type benchMockResponse struct {
	statusCode int
	body       io.ReadCloser
	headers    map[string]string
}

func (r *benchMockResponse) StatusCode() int {
	return r.statusCode
}

func (r *benchMockResponse) Body() io.ReadCloser {
	return r.body
}

func (r *benchMockResponse) Header(key string) string {
	return r.headers[key]
}

func (m *benchMockHTTPClient) Get(ctx context.Context, url string) (interfaces.Response, error) {
	// Simulate network delay
	if m.responseTime > 0 {
		time.Sleep(m.responseTime)
	}
	
	// Return a sample RSS feed
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Benchmark Feed</title>
		<description>A feed for benchmarking</description>
		<link>http://example.com</link>
		<item>
			<title>Item 1</title>
			<description>Description 1</description>
			<link>http://example.com/1</link>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
		<item>
			<title>Item 2</title>
			<description>Description 2</description>
			<link>http://example.com/2</link>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
	</channel>
</rss>`
	
	return &benchMockResponse{
		statusCode: 200,
		body:       io.NopCloser(strings.NewReader(rss)),
		headers:    make(map[string]string),
	}, nil
}

func (m *benchMockHTTPClient) Post(ctx context.Context, url string, body io.Reader) (interfaces.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

type benchMockLogger struct{}

func (m *benchMockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *benchMockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *benchMockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *benchMockLogger) Error(msg string, fields map[string]interface{}) {}

// Benchmarks
func BenchmarkParseSingleFeed(b *testing.B) {
	service := NewFeedService(interfaces.Dependencies{
		Cache:      &benchMockCache{},
		HTTPClient: &benchMockHTTPClient{},
		Logger:     &benchMockLogger{},
	})
	
	ctx := context.Background()
	url := "http://example.com/feed.rss"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseSingleFeed(ctx, url)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFeeds_10URLs(b *testing.B) {
	service := NewFeedService(interfaces.Dependencies{
		Cache:      &benchMockCache{},
		HTTPClient: &benchMockHTTPClient{},
		Logger:     &benchMockLogger{},
	})
	
	ctx := context.Background()
	urls := make([]string, 10)
	for i := 0; i < 10; i++ {
		urls[i] = fmt.Sprintf("http://example.com/feed%d.rss", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseFeeds(ctx, urls)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFeeds_100URLs(b *testing.B) {
	service := NewFeedService(interfaces.Dependencies{
		Cache:      &benchMockCache{},
		HTTPClient: &benchMockHTTPClient{responseTime: 10 * time.Millisecond}, // Simulate network latency
		Logger:     &benchMockLogger{},
	})
	
	ctx := context.Background()
	urls := make([]string, 100)
	for i := 0; i < 100; i++ {
		urls[i] = fmt.Sprintf("http://example.com/feed%d.rss", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseFeeds(ctx, urls)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFeedContent(b *testing.B) {
	service := &FeedService{}
	
	// Sample RSS content
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Benchmark Feed</title>
		<description>A feed for benchmarking</description>
		<link>http://example.com</link>
		<item>
			<title>Item 1</title>
			<description>Description 1</description>
			<link>http://example.com/1</link>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
		<item>
			<title>Item 2</title>
			<description>Description 2</description>
			<link>http://example.com/2</link>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
	</channel>
</rss>`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.parseFeedContent(content, "https://example.com/feed.xml")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPaginateItems(b *testing.B) {
	// Create a large slice of items
	items := make([]domain.FeedItem, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = domain.FeedItem{
			ID:          fmt.Sprintf("item-%d", i),
			Title:       fmt.Sprintf("Item %d", i),
			Description: fmt.Sprintf("Description %d", i),
			Link:        fmt.Sprintf("http://example.com/%d", i),
			Published:   time.Now(),
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PaginateItems(items, 5, 50) // Page 5, 50 items per page
	}
}