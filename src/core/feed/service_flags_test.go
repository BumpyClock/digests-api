package feed

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"digests-app-api/core/interfaces"
	"digests-app-api/pkg/featureflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var ErrCacheMiss = errors.New("key not found")

func createMockRSSReader() io.ReadCloser {
	rss := `<?xml version="1.0"?>
	<rss version="2.0">
		<channel>
			<title>Test Feed</title>
			<description>Test Description</description>
			<link>http://example.com</link>
			<item>
				<title>Test Item</title>
				<description>Test Item Description</description>
				<link>http://example.com/item1</link>
				<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
			</item>
		</channel>
	</rss>`
	return io.NopCloser(strings.NewReader(rss))
}

func TestParseSingleFeedWithFlags_OldParserWhenDisabled(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockHTTP := new(MockHTTPClient)
	mockLogger := new(MockLogger)
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockHTTP,
		Logger:     mockLogger,
	}
	
	service := NewFeedService(deps)
	
	// Create context with feature flags disabled
	flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
		featureflags.NewFeedParser: false,
	})
	ctx := featureflags.WithManager(context.Background(), flags)
	
	// Setup expectations
	feedURL := "http://example.com/feed.rss"
	mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, ErrCacheMiss)
	mockHTTP.On("Get", mock.Anything, feedURL).Return(&mockResponse{
		statusCode: 200,
		body:       createMockRSSReader(),
		headers:    map[string]string{},
	}, nil)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Debug", "Using legacy feed parser", mock.Anything).Once()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	
	// Execute
	feed, err := service.ParseSingleFeedWithFlags(ctx, feedURL)
	
	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, feed)
	assert.NotContains(t, feed.Description, "[v2]") // Should not have v2 marker
	mockLogger.AssertCalled(t, "Debug", "Using legacy feed parser", mock.Anything)
}

func TestParseSingleFeedWithFlags_NewParserWhenEnabled(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockHTTP := new(MockHTTPClient)
	mockLogger := new(MockLogger)
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockHTTP,
		Logger:     mockLogger,
	}
	
	service := NewFeedService(deps)
	
	// Create context with feature flags enabled
	flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
		featureflags.NewFeedParser: true,
	})
	ctx := featureflags.WithManager(context.Background(), flags)
	
	// Setup expectations
	feedURL := "http://example.com/feed.rss"
	mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, ErrCacheMiss)
	mockHTTP.On("Get", mock.Anything, feedURL).Return(&mockResponse{
		statusCode: 200,
		body:       createMockRSSReader(),
		headers:    map[string]string{},
	}, nil)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Debug", "Using new feed parser", mock.Anything).Once()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	
	// Execute
	feed, err := service.ParseSingleFeedWithFlags(ctx, feedURL)
	
	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, feed)
	assert.Contains(t, feed.Description, "[v2]") // Should have v2 marker
	mockLogger.AssertCalled(t, "Debug", "Using new feed parser", mock.Anything)
}

func TestParseFeedsWithFlags_CacheDisabled(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockHTTP := new(MockHTTPClient)
	mockLogger := new(MockLogger)
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockHTTP,
		Logger:     mockLogger,
	}
	
	service := NewFeedService(deps)
	
	// Create context with cache disabled
	flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
		featureflags.CacheEnabled: false,
	})
	ctx := featureflags.WithManager(context.Background(), flags)
	
	// Setup expectations
	urls := []string{
		"http://example.com/feed1.rss",
		"http://example.com/feed2.rss",
	}
	
	// Should not interact with cache at all
	mockCache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	
	// Should fetch directly
	for _, url := range urls {
		mockHTTP.On("Get", mock.Anything, url).Return(&mockResponse{
			statusCode: 200,
			body:       createMockRSSReader(),
			headers:    map[string]string{},
		}, nil)
	}
	
	mockLogger.On("Info", "Cache disabled by feature flag", mock.Anything).Once()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()
	
	// Execute
	feeds, err := service.ParseFeedsWithFlags(ctx, urls)
	
	// Verify
	assert.NoError(t, err)
	assert.Len(t, feeds, 2)
	mockLogger.AssertCalled(t, "Info", "Cache disabled by feature flag", mock.Anything)
}

func TestParseFeedsWithFlags_CacheEnabled(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockHTTP := new(MockHTTPClient)
	mockLogger := new(MockLogger)
	
	deps := interfaces.Dependencies{
		Cache:      mockCache,
		HTTPClient: mockHTTP,
		Logger:     mockLogger,
	}
	
	service := NewFeedService(deps)
	
	// Create context with cache enabled
	flags := featureflags.NewStaticManager(map[featureflags.FeatureFlag]bool{
		featureflags.CacheEnabled: true,
	})
	ctx := featureflags.WithManager(context.Background(), flags)
	
	// Setup expectations
	urls := []string{"http://example.com/feed1.rss"}
	
	// Should check cache
	mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, ErrCacheMiss)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	mockHTTP.On("Get", mock.Anything, urls[0]).Return(&mockResponse{
		statusCode: 200,
		body:       createMockRSSReader(),
		headers:    map[string]string{},
	}, nil)
	
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	
	// Execute
	feeds, err := service.ParseFeedsWithFlags(ctx, urls)
	
	// Verify
	assert.NoError(t, err)
	assert.Len(t, feeds, 1)
	
	// Should have used cache
	mockCache.AssertCalled(t, "Get", mock.Anything, mock.Anything)
	mockLogger.AssertNotCalled(t, "Info", "Cache disabled by feature flag", mock.Anything)
}