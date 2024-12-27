// Package digestsCache provides caching implementations.
package digestsCache

import "time"

// Cache is the interface that defines the methods for a cache implementation.
// Any struct that implements these methods can be considered a cache.
type Cache interface {
	// Set stores a key-value pair in the cache with an optional expiration time.
	// The 'prefix' and 'key' are combined to form the complete key.
	// The 'value' is the data to be stored, which can be of any type.
	// The 'expiration' is the duration after which the cached item should be considered invalid.
	Set(prefix string, key string, value interface{}, expiration time.Duration) error

	// Get retrieves the value associated with a key from the cache.
	// The 'prefix' and 'key' are combined to form the complete key.
	// The 'dest' parameter is a pointer to the variable where the retrieved value should be stored.
	// Returns an error if the key is not found or if there is an issue unmarshalling the value.
	Get(prefix string, key string, dest interface{}) error

	// GetSubscribedListsFromCache retrieves a list of subscribed feed URLs from the cache.
	// The 'prefix' is used to filter the keys in the cache.
	// Returns a slice of strings representing the feed URLs and an error if any occurred.
	GetSubscribedListsFromCache(prefix string) ([]string, error)

	// SetFeedItems sets or updates a list of FeedItems in the cache, associated with a given key.
	// It fetches existing items, merges them with the new items (deduplicating by GUID),
	// and then caches the merged list.
	// The 'prefix' and 'key' are combined to form the complete key.
	// The 'expiration' is the duration after which the cached items should be considered invalid.
	SetFeedItems(prefix string, key string, newItems []FeedItem, expiration time.Duration) error

	// Count returns the total number of items in the cache.
	Count() (int64, error)
}
