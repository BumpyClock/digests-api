// ABOUTME: In-memory cache implementation using sync.Map for thread-safe operations
// ABOUTME: Provides a simple cache with TTL support and automatic cleanup

package memory

import (
	"context"
	"errors"
	"sync"
	"time"
)

// item represents a cached item with expiration
type item struct {
	value      []byte
	expiration time.Time
	noExpire   bool
}

// MemoryCache implements the Cache interface using in-memory storage
type MemoryCache struct {
	items sync.Map
	mu    sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get the item
	value, ok := c.items.Load(key)
	if !ok {
		return nil, errors.New("key not found")
	}

	item := value.(*item)

	// Check if expired
	if !item.noExpire && time.Now().After(item.expiration) {
		// Remove expired item
		c.items.Delete(key)
		// Trigger cleanup of other expired items
		go c.cleanup()
		return nil, errors.New("key not found")
	}

	// Return a copy of the value
	result := make([]byte, len(item.value))
	copy(result, item.value)
	return result, nil
}

// Set stores a value in the cache with the given TTL
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	// Create the item
	newItem := &item{
		value:    valueCopy,
		noExpire: ttl == 0,
	}

	if ttl > 0 {
		newItem.expiration = time.Now().Add(ttl)
	}

	// Store the item
	c.items.Store(key, newItem)

	return nil
}

// Delete removes a key from the cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.items.Delete(key)
	return nil
}

// cleanup removes expired items from the cache
func (c *MemoryCache) cleanup() {
	now := time.Now()
	c.items.Range(func(key, value interface{}) bool {
		item := value.(*item)
		if !item.noExpire && now.After(item.expiration) {
			c.items.Delete(key)
		}
		return true
	})
}