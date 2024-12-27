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

// ErrSQLCacheMiss is the error returned when a key is not found or expired in the SQLite cache.
var ErrSQLCacheMiss = errors.New("sqlite cache: key not found or expired")

/**
 * @function NewSQLiteCache
 * @description Creates a new SQLiteCache instance and initializes the database.
 * @param {string} dbPath The path to the SQLite database file.
 * @returns {(*SQLiteCache, error)} A pointer to the new SQLiteCache and an error if initialization failed.
 */
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

/**
 * @function Set
 * @description Inserts or updates a key-value pair in the SQLite database with an expiration time.
 * @param {string} prefix The prefix for the key.
 * @param {string} key The key to store the value under.
 * @param {interface{}} value The value to store.
 * @param {time.Duration} expiration The expiration time for the key-value pair.
 * @returns {error} An error if the operation failed.
 */
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

/**
 * @function Get
 * @description Retrieves a value from the SQLite database by key.
 * @param {string} prefix The prefix for the key.
 * @param {string} key The key to retrieve the value for.
 * @param {interface{}} dest A pointer to the variable to store the retrieved value in.
 * @returns {error} An error if the key is not found, expired, or the value could not be unmarshaled.
 */
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
			"key": fullKey,
		}).Error("Failed to unmarshal value from SQLite cache")
		return err
	}

	return nil
}

/**
 * @function GetSubscribedListsFromCache
 * @description Retrieves all subscribed lists from the SQLite database that match the given prefix.
 * @param {string} prefix The prefix to filter keys by.
 * @returns {([]string, error)} A slice of feed URLs and an error if any occurred.
 */
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

/**
 * @function Count
 * @description Returns the total number of items in the SQLite cache, including expired items.
 * @returns {(int64, error)} The number of items and an error if the operation failed.
 */
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
