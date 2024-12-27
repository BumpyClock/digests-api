// Package digestsCache provides caching implementations.
package digestsCache

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/nitishm/go-rejson/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var ctx = context.Background()

// log is a logger instance for this package.
var log = logrus.New()

// RedisCache is a cache implementation using Redis as the backend.
type RedisCache struct {
	client  *redis.Client
	handler *rejson.Handler
}

// FeedItem represents an item in a feed, used for caching purposes.
type FeedItem struct {
	GUID    string `json:"guid"`
	FeedUrl string `json:"feedUrl"`
	// Include other fields as necessary.
}

/**
 * @function NewRedisCache
 * @description Creates a new RedisCache instance.
 * @param {string} addr The address of the Redis server.
 * @param {string} password The password for the Redis server.
 * @param {int} db The Redis database number to use.
 * @returns {(*RedisCache, error)} A pointer to the new RedisCache and an error if the connection failed.
 */
func NewRedisCache(addr string, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.WithFields(logrus.Fields{
			"address":  addr,
			"database": db,
		}).Error("Failed to connect to Redis")
		return nil, err
	}

	handler := rejson.NewReJSONHandler()
	handler.SetGoRedisClient(client)

	return &RedisCache{client: client, handler: handler}, nil
}

/**
 * @function Set
 * @description Stores a value in Redis with the given key and expiration time.
 * @param {string} prefix The prefix for the key.
 * @param {string} key The key to store the value under.
 * @param {interface{}} value The value to store.
 * @param {time.Duration} expiration The expiration time for the key-value pair.
 * @returns {error} An error if the operation failed.
 */
func (cache *RedisCache) Set(prefix string, key string, value interface{}, expiration time.Duration) error {
	_, err := cache.handler.JSONSet(prefix+":"+key, ".", value)
	if err != nil {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to set key in Redis")
		return err
	}

	if expiration != 0 {
		err = cache.client.Expire(ctx, prefix+":"+key, expiration).Err()
		if err != nil {
			log.WithFields(logrus.Fields{
				"key": key,
			}).Error("Failed to set expiration for key in Redis")
		}
	}

	return err
}

/**
 * @function Get
 * @description Retrieves a value from Redis by key.
 * @param {string} prefix The prefix for the key.
 * @param {string} key The key to retrieve the value for.
 * @param {interface{}} dest A pointer to the variable to store the retrieved value in.
 * @returns {error} An error if the key is not found or the value could not be unmarshaled.
 */
func (cache *RedisCache) Get(prefix string, key string, dest interface{}) error {
	val, err := cache.handler.JSONGet(prefix+":"+key, ".")
	if err != nil {
		if err == redis.Nil {
			log.WithFields(logrus.Fields{
				"key": key,
			}).Info("Key not found in Redis")
			return ErrCacheMiss // Use the common ErrCacheMiss
		}
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to get key from Redis")
		return err
	}

	// Convert val to []byte, then to string
	valStr := string(val.([]byte))

	err = json.Unmarshal([]byte(valStr), dest)
	if err != nil {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to unmarshal value for key from Redis")
	}

	return err
}

/**
 * @function GetSubscribedListsFromCache
 * @description Retrieves all subscribed lists from Redis that match the given prefix.
 * @param {string} prefix The prefix to filter keys by.
 * @returns {([]string, error)} A slice of feed URLs and an error if any occurred.
 */
func (cache *RedisCache) GetSubscribedListsFromCache(prefix string) ([]string, error) {
	ctx := context.Background()                               // Create a new context
	keys, err := cache.client.Keys(ctx, prefix+":*").Result() // Pass the context to the Keys method
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to get keys from Redis")
		return nil, err
	}

	var urls []string
	for _, key := range keys {
		var feedItem FeedItem
		actualKey := strings.TrimPrefix(key, prefix+":") // Remove the prefix from the key
		err := cache.Get(prefix, actualKey, &feedItem)
		if err != nil {
			log.WithFields(logrus.Fields{
				"key":   actualKey,
				"error": err,
			}).Error("Failed to get value from Redis")
			continue
		}

		if feedItem.FeedUrl != "" {
			urls = append(urls, feedItem.FeedUrl)
		}
	}

	return urls, nil
}

/**
 * @function SetFeedItems
 * @description Sets the feed items for a given key, merging with existing items if any.
 *              Deduplication is performed based on the GUID of the feed items.
 * @param {string} prefix The prefix for the cache key.
 * @param {string} key The cache key.
 * @param {[]FeedItem} newItems The new feed items to add.
 * @param {time.Duration} expiration The expiration time for the cache entry.
 * @returns {error} An error if any occurred during the operation.
 */
func (cache *RedisCache) SetFeedItems(prefix string, key string, newItems []FeedItem, expiration time.Duration) error {
	// Fetch existing items from cache
	var existingItems []FeedItem
	err := cache.Get(prefix, key, &existingItems)
	if err != nil && err != redis.Nil {
		return err
	}

	// Deduplication based on GUID
	itemMap := make(map[string]FeedItem)
	for _, item := range existingItems {
		itemMap[item.GUID] = item
	}
	for _, newItem := range newItems {
		itemMap[newItem.GUID] = newItem // This will replace existing items with the same GUID or add new ones
	}

	// Convert map back to slice
	uniqueItems := make([]FeedItem, 0, len(itemMap))
	for _, item := range itemMap {
		uniqueItems = append(uniqueItems, item)
	}

	// Cache the deduplicated slice of items
	return cache.Set(prefix, key, uniqueItems, expiration)
}

/**
 * @function Count
 * @description Returns the number of keys in the Redis database.
 * @returns {(int64, error)} The number of keys and an error if the operation failed.
 */
func (cache *RedisCache) Count() (int64, error) {
	return cache.client.DBSize(ctx).Result()
}
