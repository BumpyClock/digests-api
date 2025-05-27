// ABOUTME: Validation handler for checking if URLs are valid and accessible
// ABOUTME: Provides URL validation functionality with concurrent checking

package handlers

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"
	
	"digests-app-api/core/interfaces"
	"github.com/danielgtaylor/huma/v2"
)

// ValidateHandler handles URL validation
type ValidateHandler struct {
	httpClient interfaces.HTTPClient
}

// NewValidateHandler creates a new validation handler
func NewValidateHandler(httpClient interfaces.HTTPClient) *ValidateHandler {
	return &ValidateHandler{
		httpClient: httpClient,
	}
}

// RegisterRoutes registers validation routes
func (h *ValidateHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "validateURLs",
		Method:      http.MethodPost,
		Path:        "/validate",
		Summary:     "Validate URLs",
		Description: "Checks if provided URLs are valid and accessible",
		Tags:        []string{"Validation"},
	}, h.ValidateURLs)
}

// ValidateInput defines the input for URL validation
type ValidateInput struct {
	Body struct {
		URLs []string `json:"urls" doc:"List of URLs to validate"`
	}
}

// URLValidationResult represents validation result for a single URL
type URLValidationResult struct {
	URL    string `json:"url" doc:"The URL that was validated"`
	Status string `json:"status" doc:"Validation status: 'valid' or 'invalid'"`
}

// ValidateOutput defines the output for URL validation
type ValidateOutput struct {
	Body struct {
		Results []URLValidationResult `json:"results" doc:"Validation results for each URL"`
	}
}

// ValidateURLs handles the POST /validate endpoint
func (h *ValidateHandler) ValidateURLs(ctx context.Context, input *ValidateInput) (*ValidateOutput, error) {
	if len(input.Body.URLs) == 0 {
		return nil, huma.Error400BadRequest("No URLs provided")
	}

	// Process URLs concurrently
	var wg sync.WaitGroup
	results := make([]URLValidationResult, len(input.Body.URLs))
	
	for i, urlStr := range input.Body.URLs {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			
			status := "invalid"
			if h.isValidURL(ctx, targetURL) {
				status = "valid"
			}
			
			results[idx] = URLValidationResult{
				URL:    targetURL,
				Status: status,
			}
		}(i, urlStr)
	}
	
	wg.Wait()

	output := &ValidateOutput{}
	output.Body.Results = results
	return output, nil
}

// isValidURL checks if a URL is valid and accessible
func (h *ValidateHandler) isValidURL(ctx context.Context, urlStr string) bool {
	// Parse URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Check basic requirements
	if u.Scheme == "" || u.Host == "" {
		return false
	}

	// Only allow HTTP(S) schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Try to access the URL
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := h.httpClient.Get(ctx, urlStr)
	if err != nil {
		return false
	}
	defer resp.Body().Close()

	// Check if we got a successful response
	statusCode := resp.StatusCode()
	return statusCode >= 200 && statusCode < 400
}