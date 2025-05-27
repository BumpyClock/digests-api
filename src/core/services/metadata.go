// ABOUTME: Metadata extraction service for extracting thumbnails and metadata from web pages
// ABOUTME: Uses colly to scrape Open Graph tags and other metadata from article URLs

package services

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	
	"digests-app-api/core/interfaces"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

const (
	collyUserAgent = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
)


// MetadataService handles metadata extraction from URLs
type MetadataService struct {
	deps interfaces.Dependencies
}

// NewMetadataService creates a new metadata service
func NewMetadataService(deps interfaces.Dependencies) *MetadataService {
	return &MetadataService{
		deps: deps,
	}
}

// ExtractMetadata extracts metadata from a single URL
func (s *MetadataService) ExtractMetadata(ctx context.Context, targetURL string) (*interfaces.MetadataResult, error) {
	// Check cache first
	if s.deps.Cache != nil {
		cacheKey := "metadata:" + targetURL
		if data, err := s.deps.Cache.Get(ctx, cacheKey); err == nil && data != nil {
			var result interfaces.MetadataResult
			if err := json.Unmarshal(data, &result); err == nil {
				return &result, nil
			}
		}
	}

	// Extract metadata
	result := s.extractFromURL(targetURL)

	// Cache the result
	if s.deps.Cache != nil && result != nil {
		cacheKey := "metadata:" + targetURL
		if data, err := json.Marshal(result); err == nil {
			_ = s.deps.Cache.Set(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return result, nil
}

// ExtractMetadataBatch extracts metadata for multiple URLs concurrently
func (s *MetadataService) ExtractMetadataBatch(ctx context.Context, urls []string) map[string]*interfaces.MetadataResult {
	results := make(map[string]*interfaces.MetadataResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency
	semaphore := make(chan struct{}, 10)

	for _, url := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if result, err := s.ExtractMetadata(ctx, targetURL); err == nil && result != nil {
				mu.Lock()
				results[targetURL] = result
				mu.Unlock()
			}
		}(url)
	}

	wg.Wait()
	return results
}

// extractFromURL performs the actual metadata extraction
func (s *MetadataService) extractFromURL(targetURL string) *interfaces.MetadataResult {
	// Basic validation
	if targetURL == "" || targetURL == "http://" || targetURL == "://" || targetURL == "about:blank" {
		return nil
	}

	c := colly.NewCollector(
		colly.UserAgent(collyUserAgent),
		colly.MaxBodySize(5*1024*1024), // 5MB limit
		colly.Async(false),
		colly.AllowURLRevisit(),
	)

	// Set timeout
	c.SetRequestTimeout(10 * time.Second)
	

	result := &interfaces.MetadataResult{
		Images: []string{},
	}

	// Extract Open Graph tags
	c.OnHTML("meta", func(e *colly.HTMLElement) {
		property := e.Attr("property")
		content := e.Attr("content")
		name := e.Attr("name")
		
		if content == "" {
			return
		}

		// Theme color
		if name == "theme-color" {
			result.ThemeColor = content
		}

		// Twitter card image
		if name == "twitter:image" && result.Thumbnail == "" {
			result.Thumbnail = content
		}

		// Open Graph tags
		switch property {
		case "og:title":
			if result.Title == "" {
				result.Title = content
			}
		case "og:description":
			if result.Description == "" {
				result.Description = content
			}
		case "og:image":
			result.Images = append(result.Images, content)
			if result.Thumbnail == "" {
				result.Thumbnail = content
			}
		}
	})

	// Fallback to regular meta tags
	c.OnHTML("head", func(e *colly.HTMLElement) {
		// Title fallback
		if result.Title == "" {
			if title := e.DOM.Find("title").First().Text(); title != "" {
				result.Title = strings.TrimSpace(title)
			}
		}

		// Description fallback
		if result.Description == "" {
			e.DOM.Find("meta[name='description']").Each(func(_ int, s *goquery.Selection) {
				if content, exists := s.Attr("content"); exists && content != "" {
					result.Description = content
				}
			})
		}

		// Favicon
		e.DOM.Find("link[rel]").Each(func(_ int, s *goquery.Selection) {
			rel := s.AttrOr("rel", "")
			href := s.AttrOr("href", "")
			relValues := strings.Fields(rel)
			for _, rv := range relValues {
				if rv == "icon" || rv == "shortcut" || rv == "apple-touch-icon" {
					if href != "" && result.Favicon == "" {
						result.Favicon = e.Request.AbsoluteURL(href)
					}
				}
			}
		})
	})

	// Extract JSON-LD for additional metadata
	c.OnHTML("script[type='application/ld+json']", func(e *colly.HTMLElement) {
		var ldData map[string]interface{}
		if err := json.Unmarshal([]byte(e.Text), &ldData); err == nil {
			// Try to extract image from JSON-LD
			if result.Thumbnail == "" {
				if img, ok := ldData["image"].(string); ok {
					result.Thumbnail = img
				} else if imgObj, ok := ldData["image"].(map[string]interface{}); ok {
					if url, ok := imgObj["url"].(string); ok {
						result.Thumbnail = url
					}
				}
			}
		}
	})

	// Domain detection
	c.OnRequest(func(r *colly.Request) {
		if parsedURL, err := url.Parse(r.URL.String()); err == nil {
			result.Domain = parsedURL.Host
		}
	})

	// Add error handling
	c.OnError(func(r *colly.Response, err error) {
		s.deps.Logger.Debug("Error visiting URL for metadata", map[string]interface{}{
			"url":   targetURL,
			"error": err.Error(),
			"status": r.StatusCode,
		})
	})

	// Visit the page
	if err := c.Visit(targetURL); err != nil {
		s.deps.Logger.Debug("Failed to visit URL for metadata extraction", map[string]interface{}{
			"url":   targetURL,
			"error": err.Error(),
		})
		// Return empty result instead of nil to prevent issues
		return result
	}

	// If no thumbnail found in meta tags, try to find the first significant image
	if result.Thumbnail == "" && len(result.Images) == 0 {
		c.OnHTML("img", func(e *colly.HTMLElement) {
			src := e.Attr("src")
			if src != "" && isSignificantImage(e) {
				absURL := e.Request.AbsoluteURL(src)
				result.Images = append(result.Images, absURL)
				if result.Thumbnail == "" {
					result.Thumbnail = absURL
				}
			}
		})
		_ = c.Visit(targetURL)
	}

	return result
}

// isSignificantImage checks if an image is likely to be content (not logo/icon)
func isSignificantImage(e *colly.HTMLElement) bool {
	width := e.Attr("width")
	height := e.Attr("height")
	
	// Check dimensions if available
	if width != "" && height != "" {
		w, _ := strconv.Atoi(width)
		h, _ := strconv.Atoi(height)
		if w < 200 || h < 200 {
			return false
		}
	}
	
	// Check class/id for common patterns
	class := strings.ToLower(e.Attr("class"))
	id := strings.ToLower(e.Attr("id"))
	alt := strings.ToLower(e.Attr("alt"))
	
	// Skip logos, icons, avatars
	skipPatterns := []string{"logo", "icon", "avatar", "profile", "user", "author"}
	for _, pattern := range skipPatterns {
		if strings.Contains(class, pattern) || strings.Contains(id, pattern) || strings.Contains(alt, pattern) {
			return false
		}
	}
	
	return true
}