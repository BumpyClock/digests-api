// Package digestsCache provides caching implementations.
package digestsCache

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// GoCache is an in-memory cache implementation using the 'patrickmn/go-cache' library.
type GoCache struct {
	cache *cache.Cache
}

// ErrCacheMiss is the error returned when a key is not found in the cache.
var ErrCacheMiss = errors.New("cache: key not found")

/**
 * @function NewGoCache
 * @description Creates a new instance of GoCache.
 * @param {time.Duration} defaultExpiration The default expiration time for items in the cache.
 * @param {time.Duration} cleanupInterval The interval at which expired items are purged from the cache.
 * @returns {*GoCache} A pointer to the newly created GoCache.
 */
func NewGoCache(defaultExpiration, cleanupInterval time.Duration) *GoCache {
	c := cache.New(defaultExpiration, cleanupInterval)
	return &GoCache{cache: c}
}

/**
 * @function Set
 * @description Stores a value in the cache with the given key and expiration time.
 * @param {string} prefix A prefix to prepend to the key.
 * @param {string} key The key to store the value under.
 * @param {interface{}} value The value to store.
 * @param {time.Duration} expiration The expiration time for the key-value pair.
 * @returns {error} An error if the value could not be marshaled to JSON.
 */
func (c *GoCache) Set(prefix string, key string, value interface{}, expiration time.Duration) error {
	fullKey := prefix + ":" + key
	valBytes, err := json.Marshal(value)
	if err != nil {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to marshal value for key")
		return err
	}
	c.cache.Set(fullKey, valBytes, expiration)
	return nil
}

/**
 * @function Get
 * @description Retrieves a value from the cache by key.
 * @param {string} prefix A prefix to prepend to the key.
 * @param {string} key The key to retrieve the value for.
 * @param {interface{}} dest A pointer to the variable to store the retrieved value in.
 * @returns {error} An error if the key is not found or the value could not be unmarshaled.
 */
func (c *GoCache) Get(prefix string, key string, dest interface{}) error {
	fullKey := prefix + ":" + key
	val, found := c.cache.Get(fullKey)
	if !found {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Info("Key not found in cache")
		return ErrCacheMiss
	}

	valBytes, ok := val.([]byte)
	if !ok {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to assert type of cached value")
		return ErrCacheMiss // Or define a more appropriate error
	}

	err := json.Unmarshal(valBytes, dest)
	if err != nil {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to unmarshal cached value")
		return err
	}

	return nil
}

/**
 * @function GetSubscribedListsFromCache
 * @description Retrieves all subscribed lists from the cache that match the given prefix.
 * @param {string} prefix The prefix to filter keys by.
 * @returns {[]string, error} A slice of feed URLs and an error if any occurred.
 */
func (c *GoCache) GetSubscribedListsFromCache(prefix string) ([]string, error) {
	var urls []string
	for k, v := range c.cache.Items() {
		if strings.HasPrefix(k, prefix+":") {
			var feedItem FeedItem
			err := json.Unmarshal(v.Object.([]byte), &feedItem)
			if err != nil {
				log.WithFields(logrus.Fields{
					"key":   k,
					"error": err,
				}).Error("Failed to unmarshal value from cache")
				continue
			}

			if feedItem.FeedUrl != "" {
				urls = append(urls, feedItem.FeedUrl)
			}
		}
	}
	return urls, nil
}

/**
 * @function SetFeedItems
 * @description Sets the feed items for a given key, merging with existing items if any.
 * @param {string} prefix The prefix for the cache key.
 * @param {string} key The cache key.
 * @param {[]FeedItem} newItems The new feed items to add.
 * @param {time.Duration} expiration The expiration time for the cache entry.
 * @returns {error} An error if any occurred during the operation.
 */
func (c *GoCache) SetFeedItems(prefix string, key string, newItems []FeedItem, expiration time.Duration) error {
	var existingItems []FeedItem
	err := c.Get(prefix, key, &existingItems)
	if err != nil && err != ErrCacheMiss {
		return err
	}

	itemMap := make(map[string]FeedItem)
	for _, item := range existingItems {
		itemMap[item.GUID] = item
	}
	for _, newItem := range newItems {
		itemMap[newItem.GUID] = newItem
	}

	uniqueItems := make([]FeedItem, 0, len(itemMap))
	for _, item := range itemMap {
		uniqueItems = append(uniqueItems, item)
	}

	return c.Set(prefix, key, uniqueItems, expiration)
}

/**
 * @function Count
 * @description Returns the number of items in the cache.
 * @returns {int64, error} The number of items and an error (always nil in this implementation).
 */
func (c *GoCache) Count() (int64, error) {
	return int64(c.cache.ItemCount()), nil
}
