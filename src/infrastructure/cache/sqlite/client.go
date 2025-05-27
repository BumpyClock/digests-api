// ABOUTME: SQLite-based cache implementation for persistent caching
// ABOUTME: Provides a file-based cache that survives application restarts

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	
	_ "github.com/mattn/go-sqlite3"
)

// Client implements the Cache interface using SQLite
type Client struct {
	db       *sql.DB
	filePath string
}

// NewSQLiteCache creates a new SQLite cache client
func NewSQLiteCache(filePath string) (*Client, error) {
	if filePath == "" {
		filePath = "cache.db"
	}
	
	// Open database connection
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}
	
	client := &Client{
		db:       db,
		filePath: filePath,
	}
	
	// Initialize schema
	if err := client.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	// Start cleanup routine
	go client.cleanupRoutine()
	
	return client, nil
}

// initSchema creates the cache table if it doesn't exist
func (c *Client) initSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS cache (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL,
			expiry INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_expiry ON cache(expiry);
	`
	
	_, err := c.db.Exec(query)
	return err
}

// Get retrieves a value from the cache
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}
	
	var value []byte
	var expiry int64
	
	query := "SELECT value, expiry FROM cache WHERE key = ? AND expiry > ?"
	err := c.db.QueryRowContext(ctx, query, key, time.Now().Unix()).Scan(&value, &expiry)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("key not found or expired")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get value: %w", err)
	}
	
	return value, nil
}

// Set stores a value in the cache with TTL
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	
	if len(value) == 0 {
		return errors.New("value cannot be empty")
	}
	
	expiry := time.Now().Add(ttl).Unix()
	
	query := `
		INSERT OR REPLACE INTO cache (key, value, expiry)
		VALUES (?, ?, ?)
	`
	
	_, err := c.db.ExecContext(ctx, query, key, value, expiry)
	if err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}
	
	return nil
}

// Delete removes a value from the cache
func (c *Client) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	
	query := "DELETE FROM cache WHERE key = ?"
	_, err := c.db.ExecContext(ctx, query, key)
	
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}
	
	return nil
}

// Clear removes all values from the cache
func (c *Client) Clear(ctx context.Context) error {
	query := "DELETE FROM cache"
	_, err := c.db.ExecContext(ctx, query)
	
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	
	return nil
}

// cleanupRoutine periodically removes expired entries
func (c *Client) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *Client) cleanup() {
	query := "DELETE FROM cache WHERE expiry <= ?"
	_, _ = c.db.Exec(query, time.Now().Unix())
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// Stats returns cache statistics
func (c *Client) Stats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count total entries
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM cache").Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["total_entries"] = count
	
	// Count expired entries
	var expired int
	err = c.db.QueryRow("SELECT COUNT(*) FROM cache WHERE expiry <= ?", time.Now().Unix()).Scan(&expired)
	if err != nil {
		return nil, err
	}
	stats["expired_entries"] = expired
	
	// Database file size
	var pageCount, pageSize int
	err = c.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err == nil {
		err = c.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
		if err == nil {
			stats["db_size_bytes"] = pageCount * pageSize
		}
	}
	
	stats["file_path"] = c.filePath
	
	return stats, nil
}