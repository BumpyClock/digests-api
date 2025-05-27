// Package interfaces defines the core interfaces used throughout the application.
// These interfaces allow for dependency injection and make the code testable.
package interfaces

import (
	"context"
	"time"
)

// Cache defines the interface for cache operations.
// Implementations can be Redis, in-memory, or any other caching solution.
//
// Example usage:
//
//	cache := someCache // implements Cache interface
//	
//	// Store a value
//	err := cache.Set(ctx, "user:123", userData, 1*time.Hour)
//	
//	// Retrieve a value
//	data, err := cache.Get(ctx, "user:123")
//	if err != nil {
//		// handle error or cache miss
//	}
//	
//	// Delete a value
//	err = cache.Delete(ctx, "user:123")
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns the cached data as []byte or an error if the key doesn't exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with the given key and TTL.
	// If ttl is 0, the value should be stored indefinitely.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from the cache by key.
	// Returns nil if the key doesn't exist.
	Delete(ctx context.Context, key string) error
}