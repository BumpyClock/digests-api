// ABOUTME: Advanced example showing custom configuration and advanced features
// ABOUTME: Demonstrates dependency injection, custom implementations, and background processing

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	
	"github.com/BumpyClock/digests-api/digests-lib"
	"digests-app-api/core/workers"
)

func main() {
	// Example 1: Create client with custom configuration
	fmt.Println("=== Custom Configuration ===")
	
	client, err := digests.NewClient(
		// Use SQLite cache instead of memory
		digests.WithCacheOption(digests.CacheOption{
			Type:     digests.CacheTypeSQLite,
			FilePath: "./feeds_cache.db",
		}),
		
		// Custom HTTP client configuration
		digests.WithHTTPClientConfig(digests.HTTPClientConfig{
			Timeout:             45 * time.Second,
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 20,
			UserAgent:           "MyFeedReader/2.0",
		}),
		
		// Custom logger with prefix
		digests.WithLoggerWithPrefix("[MyApp]"),
		
		// Enable background processing for enrichment
		digests.WithBackgroundProcessing(true),
		
		// Custom worker configuration
		digests.WithWorkerConfig(workers.WorkerConfig{
			MaxWorkers: 20,
			QueueSize:  500,
		}),
		
		// Custom color cache TTL
		digests.WithColorCacheTTL(14 * 24 * time.Hour), // 14 days
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()
	
	// Example 2: Context with timeout
	fmt.Println("\n=== Context with Timeout ===")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	feeds, err := client.ParseFeeds(ctx, []string{
		"https://blog.golang.org/feed.atom",
		"https://kubernetes.io/feed.xml",
	})
	if err != nil {
		log.Printf("Error with timeout: %v\n", err)
	} else {
		for _, feed := range feeds {
			fmt.Printf("- %s\n", feed.Title)
		}
	}
	
	// Example 3: Selective enrichment
	fmt.Println("\n=== Selective Enrichment ===")
	
	// Only extract metadata, skip color extraction
	metadataFeeds, err := client.ParseFeeds(
		context.Background(),
		[]string{"https://techcrunch.com/feed/"},
		digests.WithEnrichment(true, false), // metadata=true, colors=false
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		for _, feed := range metadataFeeds {
			fmt.Printf("Feed: %s\n", feed.Title)
			for i, item := range feed.Items {
				if i >= 3 {
					break
				}
				fmt.Printf("  - %s\n", item.Title)
				if item.Thumbnail != "" {
					fmt.Printf("    Thumbnail: %s\n", item.Thumbnail)
					if item.ThumbnailColor != nil {
						fmt.Printf("    Color: RGB(%d,%d,%d)\n", 
							item.ThumbnailColor.R,
							item.ThumbnailColor.G,
							item.ThumbnailColor.B)
					}
				}
			}
		}
	}
	
	// Example 4: Podcast feed handling
	fmt.Println("\n=== Podcast Feed ===")
	podcastFeed, err := client.ParseFeed(
		context.Background(),
		"https://feeds.simplecast.com/54nAGcIl", // The Changelog podcast
	)
	if err != nil {
		log.Printf("Error parsing podcast: %v\n", err)
	} else {
		fmt.Printf("Podcast: %s\n", podcastFeed.Title)
		fmt.Printf("Type: %s\n", podcastFeed.FeedType)
		for i, episode := range podcastFeed.Items {
			if i >= 3 {
				break
			}
			fmt.Printf("\nEpisode %d:\n", i+1)
			fmt.Printf("  Title: %s\n", episode.Title)
			fmt.Printf("  Duration: %s seconds\n", episode.Duration)
			if episode.AudioURL != "" {
				fmt.Printf("  Audio: %s\n", episode.AudioURL)
			}
			if episode.Season > 0 {
				fmt.Printf("  Season: %d, Episode: %d\n", episode.Season, episode.Episode)
			}
		}
	}
	
	// Example 5: Batch processing with error handling
	fmt.Println("\n=== Batch Processing ===")
	
	feedURLs := []string{
		"https://xkcd.com/rss.xml",
		"https://invalid-feed-url.com/feed",
		"https://www.reddit.com/r/golang/.rss",
		"https://another-invalid-url.net/rss",
	}
	
	// Parse all feeds, even if some fail
	allFeeds, err := client.ParseFeeds(context.Background(), feedURLs)
	if err != nil {
		// This will only happen if ALL feeds fail
		log.Printf("Complete failure: %v\n", err)
	} else {
		successCount := 0
		for _, feed := range allFeeds {
			if feed != nil {
				successCount++
				fmt.Printf("✓ %s\n", feed.Title)
			}
		}
		fmt.Printf("\nSuccessfully parsed %d/%d feeds\n", successCount, len(feedURLs))
	}
	
	// Example 6: Share functionality (requires share storage setup)
	fmt.Println("\n=== Share Functionality ===")
	
	// Note: This will fail without proper share storage configuration
	// In a real application, you would configure share storage first
	shareURLs := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/article3",
	}
	
	share, err := client.CreateShare(context.Background(), shareURLs)
	if err != nil {
		fmt.Printf("Share creation failed (expected without storage): %v\n", err)
	} else {
		fmt.Printf("Created share: %s\n", share.ID)
		fmt.Printf("URLs: %d\n", len(share.URLs))
	}
	
	// Example 7: Quiet mode for production
	fmt.Println("\n=== Quiet Mode ===")
	
	quietClient, err := digests.NewClient(
		digests.WithQuietMode(), // Suppress all log output
		digests.WithDefaultDependencies(),
	)
	if err != nil {
		log.Fatal("Failed to create quiet client:", err)
	}
	defer quietClient.Close()
	
	// This won't produce any log output
	_, _ = quietClient.ParseFeed(context.Background(), "https://news.ycombinator.com/rss")
	fmt.Println("Quiet parsing completed (no logs)")
	
	// Clean up
	os.Remove("./feeds_cache.db")
	
	fmt.Println("\nDone!")
}