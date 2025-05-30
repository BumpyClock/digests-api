// ABOUTME: Reader handler for the Huma API  
// ABOUTME: Provides HTTP endpoints for extracting clean article content from web pages

package handlers

import (
	"context"
	"net/http"

	"digests-app-api/api/dto/requests"
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/danielgtaylor/huma/v2"
)

// ReaderHandler handles reader view extraction requests
type ReaderHandler struct {
	readerService interfaces.ReaderService
}

// NewReaderHandler creates a new reader handler
func NewReaderHandler(readerService interfaces.ReaderService) *ReaderHandler {
	return &ReaderHandler{
		readerService: readerService,
	}
}

// RegisterRoutes registers all reader-related routes
func (h *ReaderHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getReaderView",
		Method:      http.MethodPost,
		Path:        "/getreaderview",
		Summary:     "Extract reader view from URLs",
		Description: "Extracts clean article content from web pages, removing ads and clutter",
		Tags:        []string{"Reader"},
	}, h.GetReaderView)
}

// GetReaderViewInput defines the input for the GetReaderView operation
type GetReaderViewInput struct {
	Body requests.ReaderViewRequest
}

// GetReaderViewOutput defines the output for the GetReaderView operation
type GetReaderViewOutput struct {
	Body []domain.ReaderView
}

// GetReaderView handles reader view extraction
func (h *ReaderHandler) GetReaderView(ctx context.Context, input *GetReaderViewInput) (*GetReaderViewOutput, error) {
	if len(input.Body.URLs) == 0 {
		return nil, huma.Error400BadRequest("No URLs provided")
	}

	// Extract reader views
	views := h.readerService.ExtractReaderViews(ctx, input.Body.URLs)

	return &GetReaderViewOutput{
		Body: views,
	}, nil
}