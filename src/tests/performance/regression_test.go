// ABOUTME: Performance regression tests comparing old vs new implementations
// ABOUTME: Ensures new features don't degrade performance

package performance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"digests-app-api/core/domain"
	"digests-app-api/core/feed"
	"digests-app-api/core/interfaces"
	"digests-app-api/infrastructure/cache/memory"
	"digests-app-api/infrastructure/logger/standard"
	"digests-app-api/pkg/featureflags"
	"github.com/stretchr/testify/assert"
)

// MockHTTPClient for performance testing
type perfMockHTTPClient struct {
	delay time.Duration
}

func (m *perfMockHTTPClient) Get(ctx context.Context, url string) (interfaces.HTTPResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	rss := `<?xml version="1.0"?>
	<rss version="2.0">
		<channel>
			<title>Performance Test Feed</title>
			<description>Feed for performance testing</description>
			<link>http://example.com</link>
			<item>
				<title>Test Item</title>
				<description>Test Description</description>
				<link>http://example.com/item</link>
			</item>
		</channel>
	</rss>`
	
	return &perfMockResponse{
		body: []byte(rss),
	}, nil
}

func (m *perfMockHTTPClient) Post(ctx context.Context, url string, contentType string, body io.Reader) (interfaces.HTTPResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

type perfMockResponse struct {
	body []byte
}

func (r *perfMockResponse) StatusCode() int { return 200 }
func (r *perfMockResponse) Body() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(r.body))
}
func (r *perfMockResponse) Header(key string) string { return "" }

// BenchmarkOldVsNewImplementation compares performance
func BenchmarkOldVsNewImplementation(b *testing.B) {
	// Setup
	deps := interfaces.Dependencies{
		Cache:      memory.NewMemoryCache(),
		HTTPClient: &perfMockHTTPClient{delay: 5 * time.Millisecond},
		Logger:     standard.NewStandardLogger(),
	}
	
	service := feed.NewFeedService(deps)
	urls := []string{
		"http://example.com/feed1.rss",
		"http://example.com/feed2.rss",
		"http://example.com/feed3.rss",
		"http://example.com/feed4.rss",
		"http://example.com/feed5.rss",
	}
	
	b.Run("OldParser", func(b *testing.B) {
		flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
			featureflags.NewFeedParser: false,
		})
		ctx := featureflags.WithManager(context.Background(), flags)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := service.ParseFeedsWithFlags(ctx, urls)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("NewParser", func(b *testing.B) {
		flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
			featureflags.NewFeedParser: true,
		})
		ctx := featureflags.WithManager(context.Background(), flags)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := service.ParseFeedsWithFlags(ctx, urls)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestEnsureNoPerformanceRegression validates performance requirements
func TestEnsureNoPerformanceRegression(t *testing.T) {
	// Setup
	deps := interfaces.Dependencies{
		Cache:      memory.NewMemoryCache(),
		HTTPClient: &perfMockHTTPClient{delay: 10 * time.Millisecond},
		Logger:     standard.NewStandardLogger(),
	}
	
	service := feed.NewFeedService(deps)
	urls := make([]string, 20)
	for i := 0; i < 20; i++ {
		urls[i] = fmt.Sprintf("http://example.com/feed%d.rss", i)
	}
	
	// Test old implementation
	oldFlags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
		featureflags.NewFeedParser: false,
	})
	oldCtx := featureflags.WithManager(context.Background(), oldFlags)
	
	oldStart := time.Now()
	_, err := service.ParseFeedsWithFlags(oldCtx, urls)
	oldDuration := time.Since(oldStart)
	assert.NoError(t, err)
	
	// Test new implementation
	newFlags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
		featureflags.NewFeedParser: true,
	})
	newCtx := featureflags.WithManager(context.Background(), newFlags)
	
	newStart := time.Now()
	_, err = service.ParseFeedsWithFlags(newCtx, urls)
	newDuration := time.Since(newStart)
	assert.NoError(t, err)
	
	// New implementation should not be more than 10% slower
	maxAllowedDuration := oldDuration + (oldDuration / 10)
	assert.LessOrEqual(t, newDuration, maxAllowedDuration,
		"New implementation is too slow: old=%v, new=%v", oldDuration, newDuration)
	
	t.Logf("Performance comparison: old=%v, new=%v (%.2f%% difference)",
		oldDuration, newDuration,
		float64(newDuration-oldDuration)/float64(oldDuration)*100)
}

// TestMonitorMemoryUsage ensures no memory leaks
func TestMonitorMemoryUsage(t *testing.T) {
	// Setup
	deps := interfaces.Dependencies{
		Cache:      memory.NewMemoryCache(),
		HTTPClient: &perfMockHTTPClient{delay: 1 * time.Millisecond},
		Logger:     standard.NewStandardLogger(),
	}
	
	service := feed.NewFeedService(deps)
	urls := []string{"http://example.com/feed.rss"}
	
	// Test both implementations
	for _, useNewParser := range []bool{false, true} {
		t.Run(fmt.Sprintf("NewParser=%v", useNewParser), func(t *testing.T) {
			flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
				featureflags.NewFeedParser: useNewParser,
			})
			ctx := featureflags.WithManager(context.Background(), flags)
			
			// Get initial memory stats
			var m1 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)
			
			// Run operations
			for i := 0; i < 100; i++ {
				_, err := service.ParseFeedsWithFlags(ctx, urls)
				assert.NoError(t, err)
			}
			
			// Get final memory stats
			runtime.GC()
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)
			
			// Calculate memory growth
			heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
			
			// Log memory usage
			t.Logf("Memory usage - Initial: %v KB, Final: %v KB, Growth: %v KB",
				m1.HeapAlloc/1024, m2.HeapAlloc/1024, heapGrowth/1024)
			
			// Ensure reasonable memory usage (less than 10MB growth)
			assert.Less(t, heapGrowth, int64(10*1024*1024),
				"Excessive memory growth detected")
		})
	}
}

// TestCheckGoroutineLeaks ensures no goroutine leaks
func TestCheckGoroutineLeaks(t *testing.T) {
	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()
	
	// Setup
	deps := interfaces.Dependencies{
		Cache:      memory.NewMemoryCache(),
		HTTPClient: &perfMockHTTPClient{delay: 1 * time.Millisecond},
		Logger:     standard.NewStandardLogger(),
	}
	
	service := feed.NewFeedService(deps)
	urls := make([]string, 50)
	for i := 0; i < 50; i++ {
		urls[i] = fmt.Sprintf("http://example.com/feed%d.rss", i)
	}
	
	// Test both implementations
	for _, useNewParser := range []bool{false, true} {
		flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
			featureflags.NewFeedParser: useNewParser,
		})
		ctx := featureflags.WithManager(context.Background(), flags)
		
		// Run operations
		_, err := service.ParseFeedsWithFlags(ctx, urls)
		assert.NoError(t, err)
	}
	
	// Wait for goroutines to finish
	time.Sleep(100 * time.Millisecond)
	
	// Check final goroutine count
	finalGoroutines := runtime.NumGoroutine()
	goroutineGrowth := finalGoroutines - initialGoroutines
	
	t.Logf("Goroutine count - Initial: %d, Final: %d, Growth: %d",
		initialGoroutines, finalGoroutines, goroutineGrowth)
	
	// Allow for some growth but flag potential leaks
	assert.LessOrEqual(t, goroutineGrowth, 5,
		"Potential goroutine leak detected")
}

// TestValidateCacheHitRates ensures cache is working effectively
func TestValidateCacheHitRates(t *testing.T) {
	// Create a cache that tracks hits/misses
	cache := &instrumentedCache{
		cache: memory.NewMemoryCache(),
	}
	
	deps := interfaces.Dependencies{
		Cache:      cache,
		HTTPClient: &perfMockHTTPClient{delay: 10 * time.Millisecond},
		Logger:     standard.NewStandardLogger(),
	}
	
	service := feed.NewFeedService(deps)
	urls := []string{
		"http://example.com/feed1.rss",
		"http://example.com/feed2.rss",
	}
	
	ctx := context.Background()
	
	// First call - all misses
	_, err := service.ParseFeeds(ctx, urls)
	assert.NoError(t, err)
	
	initialMisses := cache.misses
	assert.Equal(t, len(urls), initialMisses)
	assert.Equal(t, 0, cache.hits)
	
	// Second call - all hits
	cache.resetStats()
	_, err = service.ParseFeeds(ctx, urls)
	assert.NoError(t, err)
	
	assert.Equal(t, len(urls), cache.hits)
	assert.Equal(t, 0, cache.misses)
	
	// Calculate hit rate
	totalRequests := cache.hits + cache.misses + initialMisses
	hitRate := float64(cache.hits) / float64(totalRequests) * 100
	
	t.Logf("Cache performance - Hits: %d, Misses: %d, Hit Rate: %.2f%%",
		cache.hits, cache.misses+initialMisses, hitRate)
	
	// Ensure good cache performance (at least 33% hit rate in this test)
	assert.GreaterOrEqual(t, hitRate, 33.0,
		"Cache hit rate is too low")
}

// instrumentedCache wraps a cache to track metrics
type instrumentedCache struct {
	cache  interfaces.Cache
	hits   int
	misses int
	mu     sync.Mutex
}

func (c *instrumentedCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	val, err := c.cache.Get(ctx, key)
	if err != nil {
		c.misses++
	} else {
		c.hits++
	}
	return val, err
}

func (c *instrumentedCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.cache.Set(ctx, key, value, ttl)
}

func (c *instrumentedCache) Delete(ctx context.Context, key string) error {
	return c.cache.Delete(ctx, key)
}

func (c *instrumentedCache) resetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hits = 0
	c.misses = 0
}