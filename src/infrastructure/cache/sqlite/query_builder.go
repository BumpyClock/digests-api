// ABOUTME: Safe SQL query builder for SQLite cache operations
// ABOUTME: Enforces parameterization and prevents SQL injection attacks

package sqlite

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Logger interface - minimal interface to avoid circular dependencies
type Logger interface {
	Warn(msg string, fields map[string]interface{})
}

// QueryBuilder provides a safe way to build SQL queries with automatic parameterization
type QueryBuilder struct {
	query  string
	params []interface{}
}

// Table and column name validation - only alphanumeric, underscore allowed
var (
	safeNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	// Maximum lengths to prevent DoS
	maxKeyLength   = 255
	maxValueLength = 1024 * 1024 // 1MB
)

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		params: make([]interface{}, 0),
	}
}

// validateName validates table/column names to prevent SQL injection
func (qb *QueryBuilder) validateName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("invalid name: %s (only alphanumeric and underscore allowed)", name)
	}
	
	// Prevent extremely long names
	if len(name) > 64 {
		return fmt.Errorf("name too long: %s (max 64 characters)", name)
	}
	
	return nil
}

// Select builds a SELECT query
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	// Validate column names
	for _, col := range columns {
		if err := qb.validateName(col); err != nil {
			// In production, we might want to handle this differently
			// For now, we'll use * for invalid column names
			qb.query = "SELECT * "
			return qb
		}
	}
	
	if len(columns) == 0 {
		qb.query = "SELECT * "
	} else {
		qb.query = "SELECT " + strings.Join(columns, ", ") + " "
	}
	
	return qb
}

// From adds FROM clause
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	if err := qb.validateName(table); err != nil {
		// This should not happen in our cache implementation
		// as table name is hardcoded
		return qb
	}
	
	qb.query += "FROM " + table + " "
	return qb
}

// Where adds WHERE clause with parameterized conditions
func (qb *QueryBuilder) Where(column string, operator string, value interface{}) *QueryBuilder {
	if err := qb.validateName(column); err != nil {
		return qb
	}
	
	// Validate operator
	allowedOperators := map[string]bool{
		"=":  true,
		"!=": true,
		">":  true,
		"<":  true,
		">=": true,
		"<=": true,
	}
	
	if !allowedOperators[operator] {
		operator = "=" // Default to equals for safety
	}
	
	if strings.Contains(qb.query, "WHERE") {
		qb.query += "AND "
	} else {
		qb.query += "WHERE "
	}
	
	qb.query += column + " " + operator + " ? "
	qb.params = append(qb.params, value)
	
	return qb
}

// Insert builds an INSERT query
func (qb *QueryBuilder) Insert(table string) *QueryBuilder {
	if err := qb.validateName(table); err != nil {
		return qb
	}
	
	qb.query = "INSERT INTO " + table + " "
	return qb
}

// InsertOrReplace builds an INSERT OR REPLACE query
func (qb *QueryBuilder) InsertOrReplace(table string) *QueryBuilder {
	if err := qb.validateName(table); err != nil {
		return qb
	}
	
	qb.query = "INSERT OR REPLACE INTO " + table + " "
	return qb
}

// Values adds VALUES clause
func (qb *QueryBuilder) Values(columns []string, values []interface{}) *QueryBuilder {
	if len(columns) != len(values) {
		return qb // Invalid input
	}
	
	// Validate column names
	validColumns := make([]string, 0, len(columns))
	validValues := make([]interface{}, 0, len(values))
	
	for i, col := range columns {
		if err := qb.validateName(col); err == nil {
			validColumns = append(validColumns, col)
			validValues = append(validValues, values[i])
		}
	}
	
	if len(validColumns) == 0 {
		return qb
	}
	
	placeholders := make([]string, len(validColumns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	
	qb.query += "(" + strings.Join(validColumns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	qb.params = append(qb.params, validValues...)
	
	return qb
}

// Delete builds a DELETE query
func (qb *QueryBuilder) Delete(table string) *QueryBuilder {
	if err := qb.validateName(table); err != nil {
		return qb
	}
	
	qb.query = "DELETE FROM " + table + " "
	return qb
}

// Build returns the built query and parameters
func (qb *QueryBuilder) Build() (string, []interface{}) {
	return strings.TrimSpace(qb.query), qb.params
}

// ValidateKey validates cache key to prevent injection and other issues
func ValidateKey(key string, logger Logger) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	
	if len(key) > maxKeyLength {
		return fmt.Errorf("key too long: max %d characters", maxKeyLength)
	}
	
	// Check for null bytes which can cause issues
	if strings.Contains(key, "\x00") {
		return errors.New("key cannot contain null bytes")
	}
	
	// While SQLite handles these safely with parameterization,
	// we can add warnings for suspicious patterns
	suspiciousPatterns := []string{
		"--",
		"/*",
		"*/",
		";",
		"'",
		"\"",
		"\\",
		"\n",
		"\r",
		"\t",
	}
	
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(key, pattern) {
			// Log warning but don't reject - parameterization handles it
			if logger != nil {
				logger.Warn("Suspicious pattern detected in cache key", map[string]interface{}{
					"pattern": pattern,
					"key_length": len(key),
					"key_preview": truncateKey(key),
				})
			}
		}
	}
	
	return nil
}

// truncateKey returns a safe preview of the key for logging
func truncateKey(key string) string {
	const maxPreview = 50
	if len(key) <= maxPreview {
		return key
	}
	return key[:maxPreview] + "..."
}

// ValidateValue validates cache value
func ValidateValue(value []byte) error {
	if len(value) == 0 {
		return errors.New("value cannot be empty")
	}
	
	if len(value) > maxValueLength {
		return fmt.Errorf("value too large: max %d bytes", maxValueLength)
	}
	
	return nil
}

// CacheQueryBuilder provides pre-built queries for cache operations
type CacheQueryBuilder struct{}

// NewCacheQueryBuilder creates a cache-specific query builder
func NewCacheQueryBuilder() *CacheQueryBuilder {
	return &CacheQueryBuilder{}
}

// GetQuery builds a parameterized GET query
func (cq *CacheQueryBuilder) GetQuery() (string, int) {
	qb := NewQueryBuilder()
	qb.Select("value", "expiry").
		From("cache").
		Where("key", "=", nil).
		Where("expiry", ">", nil)
	
	query, _ := qb.Build()
	return query, 2 // Returns query and number of expected parameters
}

// SetQuery builds a parameterized SET query
func (cq *CacheQueryBuilder) SetQuery() (string, int) {
	qb := NewQueryBuilder()
	qb.InsertOrReplace("cache").
		Values([]string{"key", "value", "expiry"}, []interface{}{nil, nil, nil})
	
	query, _ := qb.Build()
	return query, 3
}

// DeleteQuery builds a parameterized DELETE query
func (cq *CacheQueryBuilder) DeleteQuery() (string, int) {
	qb := NewQueryBuilder()
	qb.Delete("cache").Where("key", "=", nil)
	
	query, _ := qb.Build()
	return query, 1
}

// CleanupQuery builds a parameterized cleanup query
func (cq *CacheQueryBuilder) CleanupQuery() (string, int) {
	qb := NewQueryBuilder()
	qb.Delete("cache").Where("expiry", "<=", nil)
	
	query, _ := qb.Build()
	return query, 1
}