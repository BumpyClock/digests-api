package digestsCache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nitishm/go-rejson/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var ctx = context.Background()
var log = logrus.New()

type RedisCache struct {
	client  *redis.Client
	handler *rejson.Handler
}

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

func (cache *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	_, err := cache.handler.JSONSet(key, ".", value)
	if err != nil {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to set key in Redis")
		return err
	}

	err = cache.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		log.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to set expiration for key in Redis")
	}

	return err
}

func (cache *RedisCache) Get(key string, dest interface{}) error {
	val, err := cache.handler.JSONGet(key, ".")
	if err != nil {
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

// FeedItem represents the structure of a feed item.
// Adjust fields according to your actual feed item structure.
type FeedItem struct {
	GUID string `json:"guid"`
	// Include other fields as necessary.
}

func (cache *RedisCache) SetFeedItems(key string, newItems []FeedItem, expiration time.Duration) error {
	// Fetch existing items from cache
	var existingItems []FeedItem
	err := cache.Get(key, &existingItems)
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
	return cache.Set(key, uniqueItems, expiration)
}

func (cache *RedisCache) Count() (int64, error) {
	return cache.client.DBSize(ctx).Result()
}
