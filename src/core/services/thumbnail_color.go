// ABOUTME: Thumbnail color extraction service for extracting prominent colors from images
// ABOUTME: Uses K-means clustering to find the most prominent color in thumbnails

package services

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"
	
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/EdlinOrg/prominentcolor"
)

const (
	defaultColorValue = 128
	httpTimeout      = 10 * time.Second
)

// ThumbnailColorService handles color extraction from images
type ThumbnailColorService struct {
	deps       interfaces.Dependencies
	httpClient *http.Client
}

// NewThumbnailColorService creates a new thumbnail color service
func NewThumbnailColorService(deps interfaces.Dependencies) *ThumbnailColorService {
	return &ThumbnailColorService{
		deps: deps,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// ExtractColor extracts the prominent color from an image URL
func (s *ThumbnailColorService) ExtractColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	if imageURL == "" {
		return s.defaultColor(), nil
	}

	// Check cache first
	if s.deps.Cache != nil {
		cacheKey := fmt.Sprintf("thumbnailColor:%s", imageURL)
		if data, err := s.deps.Cache.Get(ctx, cacheKey); err == nil && data != nil {
			var color domain.RGBColor
			// Simple parsing - assumes format "R,G,B"
			if _, err := fmt.Sscanf(string(data), "%d,%d,%d", &color.R, &color.G, &color.B); err == nil {
				return &color, nil
			}
		}
	}

	// Extract color
	color, err := s.extractColorFromURL(ctx, imageURL)
	if err != nil {
		s.deps.Logger.Error("Failed to extract color from thumbnail", map[string]interface{}{
			"url":   imageURL,
			"error": err.Error(),
		})
		color = s.defaultColor()
	}

	// Cache the result
	if s.deps.Cache != nil {
		cacheKey := fmt.Sprintf("thumbnailColor:%s", imageURL)
		cacheData := fmt.Sprintf("%d,%d,%d", color.R, color.G, color.B)
		_ = s.deps.Cache.Set(ctx, cacheKey, []byte(cacheData), 24*time.Hour)
	}

	return color, nil
}

// extractColorFromURL downloads and extracts color from image
func (s *ThumbnailColorService) extractColorFromURL(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FeedParser/1.0)")

	// Download image
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode image
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Convert to NRGBA for processing
	bounds := img.Bounds()
	imgNRGBA := image.NewNRGBA(bounds)
	draw.Draw(imgNRGBA, bounds, img, bounds.Min, draw.Src)

	// Extract prominent color using K-means
	colors, err := prominentcolor.KmeansWithAll(
		prominentcolor.ArgumentDefault,
		imgNRGBA,
		prominentcolor.DefaultK,
		1,
		prominentcolor.GetDefaultMasks(),
	)
	
	if err != nil || len(colors) == 0 {
		// Try without background mask
		colors, err = prominentcolor.KmeansWithAll(
			prominentcolor.ArgumentDefault,
			imgNRGBA,
			prominentcolor.DefaultK,
			1,
			nil,
		)
		if err != nil || len(colors) == 0 {
			return nil, fmt.Errorf("failed to extract color: %w", err)
		}
	}

	// Return the most prominent color
	return &domain.RGBColor{
		R: uint8(colors[0].Color.R),
		G: uint8(colors[0].Color.G),
		B: uint8(colors[0].Color.B),
	}, nil
}

// defaultColor returns the default gray color
func (s *ThumbnailColorService) defaultColor() *domain.RGBColor {
	return &domain.RGBColor{
		R: defaultColorValue,
		G: defaultColorValue,
		B: defaultColorValue,
	}
}

// ExtractColorBatch extracts colors for multiple URLs concurrently
func (s *ThumbnailColorService) ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor {
	results := make(map[string]*domain.RGBColor)
	
	// Use a channel to limit concurrency
	semaphore := make(chan struct{}, 5)
	resultChan := make(chan struct {
		url   string
		color *domain.RGBColor
	}, len(imageURLs))

	// Process URLs concurrently
	for _, url := range imageURLs {
		go func(imageURL string) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			color, _ := s.ExtractColor(ctx, imageURL)
			resultChan <- struct {
				url   string
				color *domain.RGBColor
			}{url: imageURL, color: color}
		}(url)
	}

	// Collect results
	for i := 0; i < len(imageURLs); i++ {
		result := <-resultChan
		results[result.url] = result.color
	}

	return results
}