package handlers

import (
	"fmt"
	"testing"

	"digests-app-api/core/errors"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
)

func TestToHumaError(t *testing.T) {
	tests := []struct {
		name           string
		input          error
		expectedStatus int
		expectedInMsg  string
	}{
		{
			name:           "nil error returns nil",
			input:          nil,
			expectedStatus: 0,
			expectedInMsg:  "",
		},
		{
			name:           "NotFoundError returns 404",
			input:          &errors.NotFoundError{Resource: "feed"},
			expectedStatus: 404,
			expectedInMsg:  "feed not found",
		},
		{
			name:           "ValidationError returns 400",
			input:          &errors.ValidationError{Field: "url", Message: "invalid format"},
			expectedStatus: 400,
			expectedInMsg:  "url: invalid format",
		},
		{
			name:           "ExternalAPIError with 500 returns 503",
			input:          &errors.ExternalAPIError{StatusCode: 500, Message: "server error"},
			expectedStatus: 503,
			expectedInMsg:  "External service error",
		},
		{
			name:           "ExternalAPIError with 503 returns 503",
			input:          &errors.ExternalAPIError{StatusCode: 503, Message: "service unavailable"},
			expectedStatus: 503,
			expectedInMsg:  "External service error",
		},
		{
			name:           "ExternalAPIError with 429 returns 429",
			input:          &errors.ExternalAPIError{StatusCode: 429, Message: "rate limited"},
			expectedStatus: 429,
			expectedInMsg:  "Rate limited by external service",
		},
		{
			name:           "ExternalAPIError with 400 returns 400",
			input:          &errors.ExternalAPIError{StatusCode: 400, Message: "bad request"},
			expectedStatus: 400,
			expectedInMsg:  "External service request error",
		},
		{
			name:           "ExternalAPIError with 404 returns 400",
			input:          &errors.ExternalAPIError{StatusCode: 404, Message: "not found"},
			expectedStatus: 400,
			expectedInMsg:  "External service request error",
		},
		{
			name:           "ExternalAPIError with unexpected status returns 500",
			input:          &errors.ExternalAPIError{StatusCode: 200, Message: "ok but error"},
			expectedStatus: 500,
			expectedInMsg:  "Unexpected external service response",
		},
		{
			name:           "wrapped NotFoundError returns 404",
			input:          fmt.Errorf("wrapped: %w", &errors.NotFoundError{Resource: "share"}),
			expectedStatus: 404,
			expectedInMsg:  "share not found",
		},
		{
			name:           "wrapped ValidationError returns 400",
			input:          fmt.Errorf("context: %w", &errors.ValidationError{Field: "id", Message: "required"}),
			expectedStatus: 400,
			expectedInMsg:  "id: required",
		},
		{
			name:           "unknown error returns 500",
			input:          fmt.Errorf("some unknown error"),
			expectedStatus: 500,
			expectedInMsg:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toHumaError(tt.input)

			if tt.input == nil {
				assert.Nil(t, result)
				return
			}

			humaErr, ok := result.(*huma.ErrorModel)
			assert.True(t, ok, "Expected huma.ErrorModel")
			assert.Equal(t, tt.expectedStatus, humaErr.Status)
			assert.Contains(t, humaErr.Title, tt.expectedInMsg)
		})
	}
}

func TestToHumaError_ExternalAPIError_TypeAssertion(t *testing.T) {
	// Test that external API error without proper type assertion still works
	var err error = &errors.ExternalAPIError{StatusCode: 500, Message: "test"}
	
	result := toHumaError(err)
	
	humaErr, ok := result.(*huma.ErrorModel)
	assert.True(t, ok)
	assert.Equal(t, 503, humaErr.Status)
}