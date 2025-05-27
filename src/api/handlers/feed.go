// ABOUTME: Feed handlers for the Huma API
// ABOUTME: Provides HTTP endpoints for feed parsing and management

package handlers

import (
	"context"
	"net/http"

	"digests-app-api/api/dto/mappers"
	"digests-app-api/api/dto/requests"
	"digests-app-api/api/dto/responses"
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/danielgtaylor/huma/v2"
)

// FeedService interface defines the methods needed from the feed service
type FeedService interface {
	ParseFeeds(ctx context.Context, urls []string) ([]*domain.Feed, error)
	ParseSingleFeed(ctx context.Context, url string) (*domain.Feed, error)
}

// ThumbnailColorService interface for color extraction
type ThumbnailColorService interface {
	ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
	ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor
	GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error)
}

// FeedHandler handles feed-related HTTP requests
type FeedHandler struct {
	feedService          FeedService
	thumbnailColorService ThumbnailColorService
	metadataService      interfaces.MetadataService
}

// NewFeedHandler creates a new feed handler
func NewFeedHandler(feedService FeedService, thumbnailColorService ThumbnailColorService, metadataService interfaces.MetadataService) *FeedHandler {
	return &FeedHandler{
		feedService:          feedService,
		thumbnailColorService: thumbnailColorService,
		metadataService:      metadataService,
	}
}

// RegisterRoutes registers all feed-related routes
func (h *FeedHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "parseFeeds",
		Method:      http.MethodPost,
		Path:        "/parse",
		Summary:     "Parse multiple RSS/Atom feeds",
		Description: "Fetches and parses multiple RSS/Atom feeds, returning structured data",
		Tags:        []string{"Feeds"},
	}, h.ParseFeeds)

	huma.Register(api, huma.Operation{
		OperationID: "parseSingleFeed",
		Method:      http.MethodGet,
		Path:        "/feed",
		Summary:     "Parse a single RSS/Atom feed",
		Description: "Fetches and parses a single RSS/Atom feed from the provided URL",
		Tags:        []string{"Feeds"},
	}, h.ParseSingleFeed)
}

// ParseFeedsInput defines the input for the ParseFeeds operation
type ParseFeedsInput struct {
	Body requests.ParseFeedsRequest `json:"body"`
}

// ParseFeedsOutput defines the output for the ParseFeeds operation
type ParseFeedsOutput struct {
	Body responses.ParseFeedsV1Response
}

// ParseFeeds handles the POST /feeds endpoint
func (h *FeedHandler) ParseFeeds(ctx context.Context, input *ParseFeedsInput) (*ParseFeedsOutput, error) {
	// Apply defaults
	input.Body.ApplyDefaults()

	// Call service
	feeds, err := h.feedService.ParseFeeds(ctx, input.Body.URLs)
	if err != nil {
		return nil, toHumaError(err)
	}

	// Extract article URLs for metadata extraction
	articleURLs := make([]string, 0)
	urlToItemMap := make(map[string]*domain.FeedItem)
	for _, feed := range feeds {
		for i := range feed.Items {
			item := &feed.Items[i]
			if item.Link != "" {
				articleURLs = append(articleURLs, item.Link)
				urlToItemMap[item.Link] = item
			}
		}
	}

	// Extract metadata for all articles
	var metadataResults map[string]*interfaces.MetadataResult
	if h.metadataService != nil && len(articleURLs) > 0 {
		metadataResults = h.metadataService.ExtractMetadataBatch(ctx, articleURLs)
	}

	// Update items with metadata thumbnails
	thumbnailURLs := make([]string, 0)
	for url, metadata := range metadataResults {
		if item, exists := urlToItemMap[url]; exists && metadata != nil && metadata.Thumbnail != "" {
			// Update the item's thumbnail with the one from metadata
			item.Thumbnail = metadata.Thumbnail
			thumbnailURLs = append(thumbnailURLs, metadata.Thumbnail)
		}
	}

	// Also collect any existing thumbnails from RSS feed
	for _, feed := range feeds {
		for i := range feed.Items {
			item := &feed.Items[i]
			if item.Thumbnail != "" {
				// Check if we haven't already added this thumbnail
				found := false
				for _, url := range thumbnailURLs {
					if url == item.Thumbnail {
						found = true
						break
					}
				}
				if !found {
					thumbnailURLs = append(thumbnailURLs, item.Thumbnail)
				}
			}
		}
	}

	// Check cache for already computed colors
	thumbnailColors := make(map[string]*domain.RGBColor)
	if h.thumbnailColorService != nil && len(thumbnailURLs) > 0 {
		// First, check which colors are already in cache
		for _, url := range thumbnailURLs {
			// Try to get from cache without computing
			if cached, err := h.thumbnailColorService.GetCachedColor(ctx, url); err == nil && cached != nil {
				thumbnailColors[url] = cached
			}
		}
		
		// Collect URLs that need color extraction
		var urlsToProcess []string
		for _, url := range thumbnailURLs {
			if _, exists := thumbnailColors[url]; !exists {
				urlsToProcess = append(urlsToProcess, url)
			}
		}
		
		// If we have URLs to process, do it in the background
		if len(urlsToProcess) > 0 {
			// Create a new context that won't be cancelled when request ends
			backgroundCtx := context.Background()
			
			// Process colors in background
			go func() {
				h.thumbnailColorService.ExtractColorBatch(backgroundCtx, urlsToProcess)
			}()
		}
	}

	// Convert directly to V1 format for compatibility with colors
	v1Response := responses.ConvertDomainFeedsToV1ResponseWithColors(feeds, thumbnailColors)

	return &ParseFeedsOutput{
		Body: v1Response,
	}, nil
}

// ParseSingleFeedInput defines the input for the ParseSingleFeed operation
type ParseSingleFeedInput struct {
	URL          string `query:"url" required:"true" format:"uri" doc:"Feed URL to parse"`
	Page         int    `query:"page,omitempty" minimum:"1" default:"1" doc:"Page number for items"`
	ItemsPerPage int    `query:"items_per_page,omitempty" minimum:"1" maximum:"100" default:"50" doc:"Number of items per page"`
}

// ParseSingleFeedOutput defines the output for the ParseSingleFeed operation
type ParseSingleFeedOutput struct {
	Body responses.FeedResponse
}

// ParseSingleFeed handles the GET /feed endpoint
func (h *FeedHandler) ParseSingleFeed(ctx context.Context, input *ParseSingleFeedInput) (*ParseSingleFeedOutput, error) {
	// Call service
	feed, err := h.feedService.ParseSingleFeed(ctx, input.URL)
	if err != nil {
		return nil, toHumaError(err)
	}

	// Convert to response DTO
	feedResponse := mappers.ToFeedResponse(feed)
	if feedResponse == nil {
		return nil, huma.Error404NotFound("Feed not found")
	}

	// Apply pagination to items
	if input.Page == 0 {
		input.Page = 1
	}
	if input.ItemsPerPage == 0 {
		input.ItemsPerPage = 50
	}

	start := (input.Page - 1) * input.ItemsPerPage
	end := start + input.ItemsPerPage

	if start < len(feedResponse.Items) {
		if end > len(feedResponse.Items) {
			end = len(feedResponse.Items)
		}
		feedResponse.Items = feedResponse.Items[start:end]
	} else {
		feedResponse.Items = []responses.FeedItemResponse{}
	}

	return &ParseSingleFeedOutput{
		Body: *feedResponse,
	}, nil
}