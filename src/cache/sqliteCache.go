// Package digestsCache provides caching implementations.
package digestsCache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sirupsen/logrus"
)

// SQLiteCache implements the Cache interface using a local SQLite database.
type SQLiteCache struct {
	db *sql.DB
}

var ErrSQLCacheMiss = errors.New("sqlite cache: key not found or expired")

// NewSQLiteCache creates a new SQLiteCache instance and initializes the database schema.
func NewSQLiteCache(dbPath string) (*SQLiteCache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logrus.WithField("dbPath", dbPath).Error("Failed to open SQLite database")
		return nil, err
	}

	// Enable WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		logrus.WithError(err).Error("Failed to enable WAL mode")
	}

	// Create the cache table if it doesn't exist
	schema := `
	CREATE TABLE IF NOT EXISTS cache (
		id TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		expiration INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_expiration ON cache (expiration);
	`
	if _, err := db.Exec(schema); err != nil {
		logrus.WithField("error", err).Error("Failed to create cache table in SQLite")
		return nil, err
	}

	return &SQLiteCache{db: db}, nil
}

// Set inserts or updates a key in the SQLite database with an expiration time.
// prefix:key is used as the primary key.
func (c *SQLiteCache) Set(prefix string, key string, value interface{}, expiration time.Duration) error {
	fullKey := prefix + ":" + key

	valBytes, err := json.Marshal(value)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"key": key,
		}).Error("Failed to marshal value for key (SQLite)")
		return err
	}

	expirationUnix := time.Now().Add(expiration).Unix()
	if expiration == 0 {
		// If expiration is 0, set a large expiration in the distant future
		expirationUnix = time.Now().AddDate(10, 0, 0).Unix()
	}

	// Use a transaction for better performance
	tx, err := c.db.Begin()
	if err != nil {
		logrus.WithError(err).Error("Failed to begin transaction")
		return err
	}
	defer tx.Rollback() // Rollback is safe to call even if the transaction was committed

	stmt := `
	INSERT INTO cache(id, data, expiration)
	VALUES(?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		data=excluded.data,
		expiration=excluded.expiration;
	`

	_, err = tx.Exec(stmt, fullKey, valBytes, expirationUnix)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"key":   fullKey,
			"error": err,
		}).Error("Failed to set key in SQLite cache")
		return err
	}

	if err := tx.Commit(); err != nil {
		logrus.WithError(err).Error("Failed to commit transaction")
		return err
	}

	return nil
}

// Get retrieves a key from the SQLite database and unmarshals it into dest.
// If the key is expired or missing, it returns an error.
func (c *SQLiteCache) Get(prefix string, key string, dest interface{}) error {
	fullKey := prefix + ":" + key

	stmt := `
	SELECT data, expiration FROM cache
	WHERE id = ? AND expiration > ?
	LIMIT 1;
	`

	row := c.db.QueryRow(stmt, fullKey, time.Now().Unix())

	var (
		data       []byte
		expiration int64
	)
	err := row.Scan(&data, &expiration)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"key": fullKey,
			}).Debug("Key not found in SQLite cache")
			return ErrCacheMiss
		}
		logrus.WithFields(logrus.Fields{
			"key":   fullKey,
			"error": err,
		}).Error("Failed to read key from SQLite cache")
		return err
	}

	// Unmarshal data into dest
	if err := json.Unmarshal(data, dest); err != nil {
		logrus.WithFields(logrus.Fields{
			"key":   fullKey,
			"error": err,
		}).Error("Failed to unmarshal value from SQLite cache")
		return err
	}

	return nil
}

// GetSubscribedListsFromCache scans the cache table for records that start with prefix:
// and attempts to unmarshal them into a FeedItem to extract the FeedUrl.
func (c *SQLiteCache) GetSubscribedListsFromCache(prefix string) ([]string, error) {
	var urls []string

	stmt := `
	SELECT id, data FROM cache
	WHERE id LIKE ? AND expiration > ?;
	`
	rows, err := c.db.Query(stmt, prefix+":%", time.Now().Unix())
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to query SQLite cache for subscribed lists")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			fullKey string
			data    []byte
		)
		if err := rows.Scan(&fullKey, &data); err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("Failed to scan SQLite cache row")
			continue
		}

		var feedItem FeedItem
		if err := json.Unmarshal(data, &feedItem); err != nil {
			logrus.WithFields(logrus.Fields{
				"key":   fullKey,
				"error": err,
			}).Error("Failed to unmarshal value from SQLite cache")
			continue
		}

		if feedItem.FeedUrl != "" {
			urls = append(urls, feedItem.FeedUrl)
		}
	}

	if err := rows.Err(); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error iterating SQLite cache rows")
		return nil, err
	}

	return urls, nil
}

// SetFeedItems fetches existing feed items from the cache, deduplicates them with newItems,
// then updates the cache with the merged slice.
func (c *SQLiteCache) SetFeedItems(prefix string, key string, newItems []FeedItem, expiration time.Duration) error {
	var existingItems []FeedItem
	err := c.Get(prefix, key, &existingItems)
	if err != nil && !errors.Is(err, ErrCacheMiss) && !errors.Is(err, ErrSQLCacheMiss) {
		return err
	}

	// Deduplicate
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

// Count returns the total number of items in the cache (including expired items).
func (c *SQLiteCache) Count() (int64, error) {
	stmt := `SELECT COUNT(*) FROM cache;`
	var count int64
	err := c.db.QueryRow(stmt).Scan(&count)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to count items in SQLite cache")
		return 0, err
	}
	return count, nil
}
