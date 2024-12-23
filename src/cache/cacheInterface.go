// Package digestsCache provides caching implementations.
package digestsCache

import "time"

// Cache is the interface that defines the methods for a cache implementation.
type Cache interface {
	Set(prefix string, key string, value interface{}, expiration time.Duration) error
	Get(prefix string, key string, dest interface{}) error
	GetSubscribedListsFromCache(prefix string) ([]string, error)
	SetFeedItems(prefix string, key string, newItems []FeedItem, expiration time.Duration) error
	Count() (int64, error)
}
