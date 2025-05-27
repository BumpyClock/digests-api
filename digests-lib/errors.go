// ABOUTME: Error types and handling for the Digests library
// ABOUTME: Provides structured errors with context for library operations

package digests

import (
	"fmt"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation ErrorType = "validation"
	
	// ErrorTypeNotFound indicates a resource was not found
	ErrorTypeNotFound ErrorType = "not_found"
	
	// ErrorTypeNetwork indicates a network error
	ErrorTypeNetwork ErrorType = "network"
	
	// ErrorTypeParsing indicates a parsing error
	ErrorTypeParsing ErrorType = "parsing"
	
	// ErrorTypeInternal indicates an internal error
	ErrorTypeInternal ErrorType = "internal"
	
	// ErrorTypeConfiguration indicates a configuration error
	ErrorTypeConfiguration ErrorType = "configuration"
)

// Error represents a structured error from the library
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new error with the given type and message
func NewError(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// WithCause adds a cause to the error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithContext adds context to the error
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Common errors
var (
	// ErrClientClosed is returned when operations are attempted on a closed client
	ErrClientClosed = NewError(ErrorTypeInternal, "client is closed")
	
	// ErrNoCache is returned when cache operations are attempted without a cache
	ErrNoCache = NewError(ErrorTypeConfiguration, "no cache configured")
	
	// ErrNoStorage is returned when share operations are attempted without storage
	ErrNoStorage = NewError(ErrorTypeConfiguration, "no share storage configured")
	
	// ErrInvalidPagination is returned when pagination parameters are invalid
	ErrInvalidPagination = NewError(ErrorTypeValidation, "invalid pagination parameters")
)

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Type == ErrorTypeValidation
	}
	return false
}

// IsNotFoundError checks if an error is a not found error
func IsNotFoundError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Type == ErrorTypeNotFound
	}
	return false
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Type == ErrorTypeNetwork
	}
	return false
}

// IsParsingError checks if an error is a parsing error
func IsParsingError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Type == ErrorTypeParsing
	}
	return false
}