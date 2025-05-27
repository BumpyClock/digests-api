// ABOUTME: Basic example showing simple feed parsing with the Digests library
// ABOUTME: Demonstrates minimal configuration and common use cases

package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/BumpyClock/digests-api/digests-lib"
)

func main() {
	// Example 1: Create a client with default configuration
	client, err := digests.NewClient()
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()
	
	// Example 2: Parse a single feed
	fmt.Println("=== Parsing Single Feed ===")
	feed, err := client.ParseFeed(context.Background(), "https://news.ycombinator.com/rss")
	if err != nil {
		log.Printf("Error parsing feed: %v\n", err)
	} else {
		fmt.Printf("Feed: %s\n", feed.Title)
		fmt.Printf("Type: %s\n", feed.FeedType)
		fmt.Printf("Items: %d\n", len(feed.Items))
		if len(feed.Items) > 0 {
			fmt.Printf("Latest: %s\n", feed.Items[0].Title)
		}
	}
	
	// Example 3: Parse multiple feeds concurrently
	fmt.Println("\n=== Parsing Multiple Feeds ===")
	feedURLs := []string{
		"https://news.ycombinator.com/rss",
		"https://feeds.arstechnica.com/arstechnica/index",
		"https://www.theverge.com/rss/index.xml",
	}
	
	feeds, err := client.ParseFeeds(context.Background(), feedURLs)
	if err != nil {
		log.Printf("Error parsing feeds: %v\n", err)
	} else {
		for _, feed := range feeds {
			fmt.Printf("- %s (%d items)\n", feed.Title, len(feed.Items))
		}
	}
	
	// Example 4: Parse with pagination
	fmt.Println("\n=== Parsing with Pagination ===")
	paginatedFeeds, err := client.ParseFeeds(
		context.Background(),
		[]string{"https://news.ycombinator.com/rss"},
		digests.WithPagination(1, 5), // Page 1, 5 items per page
	)
	if err != nil {
		log.Printf("Error parsing with pagination: %v\n", err)
	} else {
		for _, feed := range paginatedFeeds {
			fmt.Printf("Feed: %s\n", feed.Title)
			for i, item := range feed.Items {
				fmt.Printf("  %d. %s\n", i+1, item.Title)
			}
		}
	}
	
	// Example 5: Parse without enrichment for faster performance
	fmt.Println("\n=== Parsing without Enrichment ===")
	start := time.Now()
	fastFeeds, err := client.ParseFeeds(
		context.Background(),
		[]string{"https://news.ycombinator.com/rss"},
		digests.WithoutEnrichment(),
	)
	elapsed := time.Since(start)
	
	if err != nil {
		log.Printf("Error parsing without enrichment: %v\n", err)
	} else {
		fmt.Printf("Parsed in %v\n", elapsed)
		for _, feed := range fastFeeds {
			fmt.Printf("- %s (%d items)\n", feed.Title, len(feed.Items))
		}
	}
	
	// Example 6: Search for RSS feeds
	fmt.Println("\n=== Searching for Feeds ===")
	results, err := client.Search(context.Background(), "technology news")
	if err != nil {
		log.Printf("Error searching: %v\n", err)
	} else {
		fmt.Printf("Found %d results\n", len(results))
		for i, result := range results {
			if i >= 3 {
				break // Show only first 3
			}
			fmt.Printf("- %s\n", result.Title)
			fmt.Printf("  URL: %s\n", result.FeedURL)
		}
	}
	
	// Example 7: Error handling
	fmt.Println("\n=== Error Handling ===")
	_, err = client.ParseFeed(context.Background(), "https://invalid-url-that-does-not-exist.com/feed")
	if err != nil {
		if digests.IsNetworkError(err) {
			fmt.Println("Network error occurred:", err)
		} else if digests.IsParsingError(err) {
			fmt.Println("Parsing error occurred:", err)
		} else {
			fmt.Println("Other error occurred:", err)
		}
	}
	
	fmt.Println("\nDone!")
}