// Package core contains the business logic for the Digests API.
// It is designed to be framework-agnostic and can be used independently
// of any web framework or infrastructure concerns.
//
// The core package is organized into several sub-packages:
//
// - domain: Contains pure domain models (Feed, FeedItem, Share, etc.)
// - feed: Feed parsing and processing service
// - search: Feed discovery and search service  
// - share: Feed sharing service
// - errors: Custom error types for better error handling
// - interfaces: Contracts for external dependencies (cache, HTTP, logger)
//
// # Design Principles
//
// The core package follows clean architecture principles:
// - No external framework dependencies
// - All external dependencies are injected via interfaces
// - Business logic is testable in isolation
// - Domain models are free from persistence concerns
//
// # Usage Example
//
//	import (
//	    "digests-app-api/core/feed"
//	    "digests-app-api/core/interfaces"
//	)
//	
//	// Create dependencies
//	deps := interfaces.Dependencies{
//	    Cache:      myCache,      // implements interfaces.Cache
//	    HTTPClient: myHTTPClient, // implements interfaces.HTTPClient
//	    Logger:     myLogger,     // implements interfaces.Logger
//	}
//	
//	// Create service
//	feedService := feed.NewFeedService(deps)
//	
//	// Parse feeds
//	feeds, err := feedService.ParseFeeds(ctx, []string{
//	    "https://example.com/feed.rss",
//	})
//
package core