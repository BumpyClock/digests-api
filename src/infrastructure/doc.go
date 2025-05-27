// Package infrastructure provides concrete implementations of the interfaces
// defined in the core package. These implementations handle external concerns
// such as caching, HTTP communication, and logging.
//
// The infrastructure package is organized by technical concern:
//
// - cache/memory: In-memory cache implementation using sync.Map
// - cache/redis: Redis-based cache implementation
// - http/standard: Standard library HTTP client with retry logic
// - logger/standard: Simple structured logger implementation
//
// # Design Philosophy
//
// Infrastructure components are designed to be:
// - Pluggable: Easy to swap implementations
// - Configurable: Accept configuration objects
// - Testable: Include both unit and integration tests
// - Production-ready: Include retries, timeouts, and error handling
//
// # Cache Implementations
//
// Memory Cache Example:
//
//	cache := memory.NewMemoryCache()
//	err := cache.Set(ctx, "key", []byte("value"), 1*time.Hour)
//	value, err := cache.Get(ctx, "key")
//
// Redis Cache Example:
//
//	config := &redis.Config{
//	    Address:  "localhost:6379",
//	    Password: "",
//	    DB:       0,
//	}
//	cache, err := redis.NewRedisCache(config)
//
// # HTTP Client
//
// The HTTP client includes automatic retry logic for transient failures:
//
//	client := standard.NewStandardHTTPClient(30 * time.Second)
//	resp, err := client.Get(ctx, "https://example.com")
//	if err != nil {
//	    // Handle error
//	}
//	defer resp.Body().Close()
//
// # Logger
//
// The logger supports structured logging with fields:
//
//	logger := standard.NewStandardLogger()
//	logger.Info("Processing request", map[string]interface{}{
//	    "user_id": "123",
//	    "action":  "parse_feed",
//	})
//
package infrastructure