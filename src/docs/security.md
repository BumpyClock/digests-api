# Security Guidelines

## SQL Injection Prevention

### Overview
The Digests API uses SQLite for persistent caching. To prevent SQL injection attacks, we implement multiple layers of protection.

### Protection Layers

#### 1. Parameterized Queries (Primary Defense)
All SQL queries use parameter placeholders (`?`) instead of string concatenation:

```go
// ✅ SAFE: Parameterized query
query := "SELECT value FROM cache WHERE key = ?"
db.QueryRow(query, userInput)

// ❌ UNSAFE: String concatenation (NEVER do this)
query := "SELECT value FROM cache WHERE key = '" + userInput + "'"
```

#### 2. Query Builder Pattern
We use a query builder (`query_builder.go`) that:
- Enforces parameterization
- Validates table and column names
- Prevents string concatenation in queries

```go
// Query builder automatically parameterizes
qb := NewCacheQueryBuilder()
query, _ := qb.GetQuery()
db.QueryRow(query, key, timestamp)
```

#### 3. Input Validation
All inputs are validated before use:

```go
// Key validation
func ValidateKey(key string) error {
    if key == "" {
        return errors.New("key cannot be empty")
    }
    if len(key) > maxKeyLength {
        return errors.New("key too long")
    }
    if strings.Contains(key, "\x00") {
        return errors.New("key cannot contain null bytes")
    }
    return nil
}
```

#### 4. Size Limits
To prevent DoS attacks:
- Maximum key length: 255 characters
- Maximum value size: 1MB

### SQL Query Patterns

#### Safe Query Examples
```go
// SELECT with parameters
query := "SELECT value, expiry FROM cache WHERE key = ? AND expiry > ?"
db.QueryRow(query, key, time.Now().Unix())

// INSERT with parameters
query := "INSERT OR REPLACE INTO cache (key, value, expiry) VALUES (?, ?, ?)"
db.Exec(query, key, value, expiry)

// DELETE with parameters
query := "DELETE FROM cache WHERE key = ?"
db.Exec(query, key)
```

#### Table/Column Name Validation
Table and column names cannot be parameterized in SQL. We validate them using regex:

```go
// Only alphanumeric and underscore allowed
safeNamePattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
```

### Testing SQL Injection

The `client_security_test.go` file contains comprehensive SQL injection tests:

```go
// Test various injection attempts
injectionKeys := []string{
    "key'; DROP TABLE cache; --",
    "key' OR '1'='1",
    "key\" OR \"1\"=\"1",
    // ... many more
}
```

### Best Practices

1. **Always use parameterized queries** - Never concatenate user input into SQL
2. **Validate all inputs** - Check for empty values, size limits, and suspicious patterns
3. **Use the query builder** - It enforces safe patterns automatically
4. **Log suspicious activity** - Monitor for injection attempts
5. **Keep SQLite updated** - Security patches are important
6. **Principle of least privilege** - Cache operations don't need admin rights

### Code Review Checklist

When reviewing code that touches the database:

- [ ] All queries use `?` placeholders for user input
- [ ] No string concatenation in SQL queries
- [ ] Input validation is performed
- [ ] Size limits are enforced
- [ ] Query builder is used where applicable
- [ ] Error messages don't expose SQL structure

### Monitoring

Look for these patterns in logs as potential injection attempts:
- Keys containing SQL keywords (DROP, SELECT, UNION, etc.)
- Keys with special characters (', ", --, /*, etc.)
- Unusually long keys or values
- Repeated failed operations from same source

### Incident Response

If SQL injection is suspected:

1. **Immediate**: Check if tables still exist
   ```sql
   SELECT name FROM sqlite_master WHERE type='table';
   ```

2. **Investigate**: Review logs for suspicious patterns
   ```bash
   grep -E "(DROP|UNION|SELECT.*FROM)" /var/log/digests-api.log
   ```

3. **Mitigate**: If compromised, restore from backup
4. **Patch**: Update code to prevent future attacks
5. **Report**: Document incident and lessons learned