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

type GoCache struct {
	cache *cache.Cache
}

var ErrCacheMiss = errors.New("cache: key not found")

func NewGoCache(defaultExpiration, cleanupInterval time.Duration) *GoCache {
	c := cache.New(defaultExpiration, cleanupInterval)
	return &GoCache{cache: c}
}

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

func (c *GoCache) Count() (int64, error) {
	return int64(c.cache.ItemCount()), nil
}
