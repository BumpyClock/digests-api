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
	"net/url"
	"strings"
	"sync"
	"time"
	
	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/EdlinOrg/prominentcolor"
	_ "golang.org/x/image/webp" // WebP support
)

const (
	defaultColorValue = 128
	httpTimeout      = 10 * time.Second
	userAgent        = "Mozilla/5.0 (compatible; FeedParser/1.0)"
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
		s.deps.Logger.Debug("Failed to extract color from thumbnail", map[string]interface{}{
			"url":   imageURL,
			"error": err.Error(),
		})
		color = s.defaultColor()
	}
	
	// Ensure color is not nil
	if color == nil {
		color = s.defaultColor()
	}

	// Cache the result
	if s.deps.Cache != nil && color != nil {
		cacheKey := fmt.Sprintf("thumbnailColor:%s", imageURL)
		cacheData := fmt.Sprintf("%d,%d,%d", color.R, color.G, color.B)
		_ = s.deps.Cache.Set(ctx, cacheKey, []byte(cacheData), 24*time.Hour)
	}

	return color, nil
}

// extractColorFromURL downloads and extracts color from image
func (s *ThumbnailColorService) extractColorFromURL(ctx context.Context, imageURL string) (color *domain.RGBColor, err error) {
	// Add panic recovery like the original code
	defer func() {
		if rec := recover(); rec != nil {
			s.deps.Logger.Debug("Recovered from panic in color extraction", map[string]interface{}{
				"url":   imageURL,
				"panic": fmt.Sprintf("%v", rec),
			})
			// Return default color on panic
			color = s.defaultColor()
			err = fmt.Errorf("panic recovered: %v", rec)
		}
	}()

	// Validate URL
	parsedURL, parseErr := url.Parse(imageURL)
	if parseErr != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid image URL: %s", imageURL)
	}
	
	// Skip SVG files as they can't be decoded as raster images
	if strings.HasSuffix(strings.ToLower(imageURL), ".svg") {
		return nil, fmt.Errorf("SVG images are not supported")
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent from original code
	req.Header.Set("User-Agent", userAgent)

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
	if bounds.Empty() {
		return nil, fmt.Errorf("image has empty bounds")
	}
	
	imgNRGBA := image.NewNRGBA(bounds)
	if imgNRGBA == nil {
		return nil, fmt.Errorf("failed to create NRGBA image")
	}
	
	draw.Draw(imgNRGBA, bounds, img, bounds.Min, draw.Src)

	// Try to extract color with masks first
	var colors []prominentcolor.ColorItem
	colors, err = prominentcolor.KmeansWithAll(
		prominentcolor.ArgumentDefault,
		imgNRGBA,
		prominentcolor.DefaultK,
		1,
		prominentcolor.GetDefaultMasks(),
	)
	
	// If no colors found or error, try without masks
	if err != nil || len(colors) == 0 {
		s.deps.Logger.Debug("Retrying color extraction without masks", map[string]interface{}{
			"url": imageURL,
			"error": err,
		})
		
		colors, err = prominentcolor.KmeansWithAll(
			prominentcolor.ArgumentDefault,
			imgNRGBA,
			prominentcolor.DefaultK,
			1,
			nil,
		)
		
		// If still no colors, return error
		if err != nil || len(colors) == 0 {
			return nil, fmt.Errorf("no colors extracted from image")
		}
	}

	// Safety check before accessing colors[0]
	if len(colors) == 0 {
		return nil, fmt.Errorf("color array is empty")
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

// GetCachedColor retrieves a color from cache without computing it
func (s *ThumbnailColorService) GetCachedColor(ctx context.Context, imageURL string) (*domain.RGBColor, error) {
	if imageURL == "" {
		return nil, fmt.Errorf("empty image URL")
	}

	// Check cache only
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

	return nil, fmt.Errorf("color not found in cache")
}

// ExtractColorBatch extracts colors for multiple URLs concurrently
func (s *ThumbnailColorService) ExtractColorBatch(ctx context.Context, imageURLs []string) map[string]*domain.RGBColor {
	results := make(map[string]*domain.RGBColor)
	resultsMutex := sync.Mutex{}
	
	// Log batch processing
	s.deps.Logger.Debug("Starting batch color extraction", map[string]interface{}{
		"count": len(imageURLs),
	})
	
	// Use a wait group and limited concurrency
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Reasonable concurrency for background processing

	// Process URLs concurrently
	for _, url := range imageURLs {
		wg.Add(1)
		go func(imageURL string) {
			defer wg.Done()
			
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				
				// Extract color - will use cache if available
				color, err := s.ExtractColor(ctx, imageURL)
				if err != nil {
					// Log error but don't store default - let it compute next time
					s.deps.Logger.Debug("Failed to extract color in batch", map[string]interface{}{
						"url":   imageURL,
						"error": err.Error(),
					})
					return
				}
				
				resultsMutex.Lock()
				results[imageURL] = color
				resultsMutex.Unlock()
				
			case <-ctx.Done():
				// Context cancelled
				return
			}
		}(url)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	
	s.deps.Logger.Debug("Completed batch color extraction", map[string]interface{}{
		"requested": len(imageURLs),
		"extracted": len(results),
	})

	return results
}