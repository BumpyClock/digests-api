package sqlite

import (
	"strings"
	"testing"
)

func TestQueryBuilder_Select(t *testing.T) {
	tests := []struct {
		name     string
		columns  []string
		expected string
	}{
		{
			name:     "Select all",
			columns:  []string{},
			expected: "SELECT *",
		},
		{
			name:     "Select specific columns",
			columns:  []string{"value", "expiry"},
			expected: "SELECT value, expiry",
		},
		{
			name:     "Invalid column names",
			columns:  []string{"value; DROP TABLE cache;", "expiry"},
			expected: "SELECT *", // Falls back to * for safety
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder()
			qb.Select(tt.columns...)
			query, _ := qb.Build()
			
			if !strings.HasPrefix(query, tt.expected) {
				t.Errorf("Expected query to start with %q, got %q", tt.expected, query)
			}
		})
	}
}

func TestQueryBuilder_Where(t *testing.T) {
	tests := []struct {
		name     string
		column   string
		operator string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid where clause",
			column:   "key",
			operator: "=",
			expected: "WHERE key = ?",
		},
		{
			name:     "Invalid operator defaults to =",
			column:   "key",
			operator: "LIKE",
			expected: "WHERE key = ?",
		},
		{
			name:     "SQL injection in column name",
			column:   "key; DROP TABLE cache;",
			operator: "=",
			expected: "", // Should not add WHERE clause
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder()
			qb.Where(tt.column, tt.operator, "test")
			query, params := qb.Build()
			
			if tt.wantErr {
				if strings.Contains(query, "WHERE") {
					t.Errorf("Expected no WHERE clause for invalid input, got %q", query)
				}
				return
			}
			
			if !strings.Contains(query, tt.expected) {
				t.Errorf("Expected query to contain %q, got %q", tt.expected, query)
			}
			
			if len(params) != 1 {
				t.Errorf("Expected 1 parameter, got %d", len(params))
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid key",
			key:     "cache:user:123",
			wantErr: false,
		},
		{
			name:    "Empty key",
			key:     "",
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "Key too long",
			key:     strings.Repeat("a", 256),
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "Key with null byte",
			key:     "key\x00null",
			wantErr: true,
			errMsg:  "null bytes",
		},
		{
			name:    "Key with SQL injection attempt",
			key:     "key'; DROP TABLE cache; --",
			wantErr: false, // We allow it but parameterization handles it
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateValue(t *testing.T) {
	tests := []struct {
		name    string
		value   []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid value",
			value:   []byte("test data"),
			wantErr: false,
		},
		{
			name:    "Empty value",
			value:   []byte{},
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "Nil value",
			value:   nil,
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "Value too large",
			value:   make([]byte, 1024*1024+1),
			wantErr: true,
			errMsg:  "too large",
		},
		{
			name:    "Binary data",
			value:   []byte{0x00, 0x01, 0x02, 0x03},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestCacheQueryBuilder(t *testing.T) {
	cqb := NewCacheQueryBuilder()
	
	t.Run("GetQuery", func(t *testing.T) {
		query, params := cqb.GetQuery()
		expected := "SELECT value, expiry FROM cache WHERE key = ? AND expiry > ?"
		
		if query != expected {
			t.Errorf("Expected %q, got %q", expected, query)
		}
		
		if params != 2 {
			t.Errorf("Expected 2 parameters, got %d", params)
		}
	})
	
	t.Run("SetQuery", func(t *testing.T) {
		query, params := cqb.SetQuery()
		expected := "INSERT OR REPLACE INTO cache (key, value, expiry) VALUES (?, ?, ?)"
		
		if query != expected {
			t.Errorf("Expected %q, got %q", expected, query)
		}
		
		if params != 3 {
			t.Errorf("Expected 3 parameters, got %d", params)
		}
	})
	
	t.Run("DeleteQuery", func(t *testing.T) {
		query, params := cqb.DeleteQuery()
		expected := "DELETE FROM cache WHERE key = ?"
		
		if query != expected {
			t.Errorf("Expected %q, got %q", expected, query)
		}
		
		if params != 1 {
			t.Errorf("Expected 1 parameter, got %d", params)
		}
	})
	
	t.Run("CleanupQuery", func(t *testing.T) {
		query, params := cqb.CleanupQuery()
		expected := "DELETE FROM cache WHERE expiry <= ?"
		
		if query != expected {
			t.Errorf("Expected %q, got %q", expected, query)
		}
		
		if params != 1 {
			t.Errorf("Expected 1 parameter, got %d", params)
		}
	})
}

func TestQueryBuilder_NameValidation(t *testing.T) {
	qb := NewQueryBuilder()
	
	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"Valid name", "cache_table", false},
		{"Name with numbers", "table123", false},
		{"Starting with underscore", "_table", false},
		{"Empty name", "", true},
		{"Name with spaces", "table name", true},
		{"Name with special chars", "table-name", true},
		{"SQL injection attempt", "table; DROP TABLE users;", true},
		{"Name too long", strings.Repeat("a", 65), true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qb.validateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}