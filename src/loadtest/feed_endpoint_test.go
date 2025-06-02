// ABOUTME: Load tests for the /feeds endpoint
// ABOUTME: Tests performance under high concurrent load

package loadtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digests-app-api/api"
	"digests-app-api/api/dto/requests"
	"digests-app-api/api/handlers"
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
)

// MockFeedService for load testing
type mockFeedService struct {
	delay time.Duration
}

func (m *mockFeedService) ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	feeds := make([]*domain.Feed, len(urls))
	for i, url := range urls {
		feeds[i] = &domain.Feed{
			ID:          fmt.Sprintf("feed-%d", i),
			Title:       fmt.Sprintf("Feed from %s", url),
			Description: "Test feed",
			URL:         url,
			Items: []domain.FeedItem{
				{
					ID:          "item-1",
					Title:       "Test Item",
					Description: "Test Description",
					Link:        "http://example.com/item",
					Published:   time.Now(),
				},
			},
			LastUpdated: time.Now(),
		}
	}
	return feeds, nil
}

func (m *mockFeedService) ParseSingleFeed(ctx context.Context, url string) (*domain.Feed, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	return &domain.Feed{
		ID:          "feed-1",
		Title:       fmt.Sprintf("Feed from %s", url),
		Description: "Test feed",
		URL:         url,
		Items:       []domain.FeedItem{},
		LastUpdated: time.Now(),
	}, nil
}

func (m *mockFeedService) ParseFeedsWithConfig(ctx context.Context, urls []string, config interface{}) ([]*domain.Feed, error) {
	return m.ParseFeeds(ctx, urls)
}

// mockEnrichmentService is a minimal stub of ContentEnrichmentService
// used for load testing with fast, no-op responses.
type mockEnrichmentService struct{}

func (m *mockEnrichmentService) ExtractMetadata(ctx context.Context, url string) (*interfaces.MetadataResult, error) {
	return nil, nil
}

func (m *mockEnrichmentService) ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*interfaces.MetadataResult {
	return make(map[string]*interfaces.MetadataResult)
}

func (m *mockEnrichmentService) ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	return nil, nil
}

func (m *mockEnrichmentService) ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor {
	return make(map[string]*domain.RGBColor)
}

func (m *mockEnrichmentService) GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	return nil, nil
}

// LoadTestMetrics tracks performance metrics
type LoadTestMetrics struct {
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedReqs      int64
	TotalDuration   time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	RequestsPerSec  float64
}

func TestFeedsEndpoint_100ConcurrentRequests(t *testing.T) {
	// Setup
	apiInstance, router := api.NewAPI()
	feedService := &mockFeedService{delay: 10 * time.Millisecond}
	enrichmentService := &mockEnrichmentService{}
	handler := handlers.NewFeedHandler(feedService, enrichmentService)
	handler.RegisterRoutes(apiInstance)
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test configuration
	concurrency := 100
	requestsPerWorker := 10
	totalRequests := concurrency * requestsPerWorker
	
	// Metrics collection
	var (
		successCount int64
		failCount    int64
		latencies    []time.Duration
		mu           sync.Mutex
	)
	
	// Create wait group
	var wg sync.WaitGroup
	wg.Add(concurrency)
	
	// Start time
	startTime := time.Now()
	
	// Launch concurrent workers
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			
			client := &http.Client{
				Timeout: 30 * time.Second,
			}
			
			for j := 0; j < requestsPerWorker; j++ {
				// Prepare request
				reqBody := requests.ParseFeedsRequest{
					URLs: []string{
						fmt.Sprintf("http://example.com/feed%d.rss", j),
						fmt.Sprintf("http://example.com/feed%d.atom", j),
					},
				}
				
				body, _ := json.Marshal(reqBody)
				
				// Make request
				reqStart := time.Now()
				resp, err := client.Post(
					server.URL+"/feeds",
					"application/json",
					bytes.NewReader(body),
				)
				latency := time.Since(reqStart)
				
				// Record metrics
				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
				
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					continue
				}
				
				// Read response body
				io.ReadAll(resp.Body)
				resp.Body.Close()
				
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failCount, 1)
				}
			}
		}(i)
	}
	
	// Wait for all workers to complete
	wg.Wait()
	totalDuration := time.Since(startTime)
	
	// Calculate metrics
	metrics := calculateMetrics(latencies, totalDuration, totalRequests)
	metrics.SuccessfulReqs = successCount
	metrics.FailedReqs = failCount
	
	// Print results
	t.Logf("Load Test Results - 100 Concurrent Requests")
	t.Logf("==========================================")
	t.Logf("Total Requests: %d", metrics.TotalRequests)
	t.Logf("Successful: %d", metrics.SuccessfulReqs)
	t.Logf("Failed: %d", metrics.FailedReqs)
	t.Logf("Total Duration: %v", metrics.TotalDuration)
	t.Logf("Requests/sec: %.2f", metrics.RequestsPerSec)
	t.Logf("Min Latency: %v", metrics.MinLatency)
	t.Logf("Avg Latency: %v", metrics.AvgLatency)
	t.Logf("P95 Latency: %v", metrics.P95Latency)
	t.Logf("P99 Latency: %v", metrics.P99Latency)
	t.Logf("Max Latency: %v", metrics.MaxLatency)
	
	// Assertions
	if metrics.FailedReqs > 0 {
		t.Errorf("Had %d failed requests", metrics.FailedReqs)
	}
	
	if metrics.P95Latency > 1*time.Second {
		t.Errorf("P95 latency too high: %v", metrics.P95Latency)
	}
}

func TestFeedsEndpoint_1000RequestsPerSecond(t *testing.T) {
	// Setup
	apiInstance, router := api.NewAPI()
	feedService := &mockFeedService{delay: 5 * time.Millisecond}
	enrichmentService := &mockEnrichmentService{}
	handler := handlers.NewFeedHandler(feedService, enrichmentService)
	handler.RegisterRoutes(apiInstance)
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test configuration
	targetRPS := 1000
	duration := 5 * time.Second
	totalRequests := targetRPS * int(duration.Seconds())
	
	// Metrics
	var (
		successCount int64
		failCount    int64
		latencies    []time.Duration
		mu           sync.Mutex
	)
	
	// Create rate limiter
	ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
	defer ticker.Stop()
	
	// Context for cancellation
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	// Start time
	startTime := time.Now()
	
	// Request counter
	var requestCount int64
	
	// Launch request sender
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			go func(reqNum int64) {
				// Prepare request
				reqBody := requests.ParseFeedsRequest{
					URLs: []string{
						fmt.Sprintf("http://example.com/feed%d.rss", reqNum),
					},
				}
				
				body, _ := json.Marshal(reqBody)
				
				// Make request
				reqStart := time.Now()
				resp, err := client.Post(
					server.URL+"/feeds",
					"application/json",
					bytes.NewReader(body),
				)
				latency := time.Since(reqStart)
				
				// Record metrics
				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
				
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					return
				}
				
				io.ReadAll(resp.Body)
				resp.Body.Close()
				
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failCount, 1)
				}
			}(atomic.AddInt64(&requestCount, 1))
		}
	}
	
done:
	// Wait a bit for in-flight requests
	time.Sleep(1 * time.Second)
	
	totalDuration := time.Since(startTime)
	
	// Calculate metrics
	metrics := calculateMetrics(latencies, totalDuration, int(requestCount))
	metrics.SuccessfulReqs = successCount
	metrics.FailedReqs = failCount
	
	// Print results
	t.Logf("Load Test Results - 1000 Requests/Second")
	t.Logf("=======================================")
	t.Logf("Target RPS: %d", targetRPS)
	t.Logf("Actual RPS: %.2f", metrics.RequestsPerSec)
	t.Logf("Total Requests: %d", metrics.TotalRequests)
	t.Logf("Successful: %d", metrics.SuccessfulReqs)
	t.Logf("Failed: %d", metrics.FailedReqs)
	t.Logf("Success Rate: %.2f%%", float64(metrics.SuccessfulReqs)/float64(metrics.TotalRequests)*100)
	t.Logf("P95 Latency: %v", metrics.P95Latency)
	t.Logf("P99 Latency: %v", metrics.P99Latency)
	
	// Assertions
	successRate := float64(metrics.SuccessfulReqs) / float64(metrics.TotalRequests)
	if successRate < 0.95 {
		t.Errorf("Success rate too low: %.2f%%", successRate*100)
	}
}

// calculateMetrics computes performance metrics from latency data
func calculateMetrics(latencies []time.Duration, totalDuration time.Duration, totalRequests int) LoadTestMetrics {
	if len(latencies) == 0 {
		return LoadTestMetrics{}
	}
	
	// Sort latencies for percentile calculation
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	
	// Simple bubble sort (fine for test data)
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}
	
	// Calculate metrics
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	
	p95Index := int(float64(len(sortedLatencies)) * 0.95)
	p99Index := int(float64(len(sortedLatencies)) * 0.99)
	
	return LoadTestMetrics{
		TotalRequests:  int64(totalRequests),
		TotalDuration:  totalDuration,
		MinLatency:     sortedLatencies[0],
		MaxLatency:     sortedLatencies[len(sortedLatencies)-1],
		AvgLatency:     sum / time.Duration(len(latencies)),
		P95Latency:     sortedLatencies[p95Index],
		P99Latency:     sortedLatencies[p99Index],
		RequestsPerSec: float64(totalRequests) / totalDuration.Seconds(),
	}
}