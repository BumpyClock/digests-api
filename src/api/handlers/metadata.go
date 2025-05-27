// ABOUTME: Metadata handler for extracting Open Graph and meta tags from web pages
// ABOUTME: Provides structured metadata extraction including images, videos, and site info

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	
	"github.com/PuerkitoBio/goquery"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gocolly/colly"
)

// MetadataHandler handles metadata extraction
type MetadataHandler struct {
	// We'll use colly for metadata extraction like the old implementation
}

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler() *MetadataHandler {
	return &MetadataHandler{}
}

// RegisterRoutes registers metadata routes
func (h *MetadataHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "extractMetadata", 
		Method:      http.MethodPost,
		Path:        "/metadata",
		Summary:     "Extract metadata from web pages",
		Description: "Extracts Open Graph tags, JSON-LD data, and other metadata from provided URLs",
		Tags:        []string{"Metadata"},
	}, h.ExtractMetadata)
}

// MetadataInput defines the input for metadata extraction
type MetadataInput struct {
	Body struct {
		URLs []string `json:"urls" doc:"List of URLs to extract metadata from"`
	}
}

// WebMedia represents media information
type WebMedia struct {
	URL         string   `json:"url"`
	Alt         string   `json:"alt,omitempty"`
	Type        string   `json:"type,omitempty"`
	Width       int      `json:"width,omitempty"`
	Height      int      `json:"height,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SecureURL   string   `json:"secure_url,omitempty"`
	Duration    int      `json:"duration,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty"`
}

// MetadataItem represents extracted metadata
type MetadataItem struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Images      []WebMedia  `json:"images"`
	Type        string      `json:"type"`
	Sitename    string      `json:"sitename"`
	Favicon     string      `json:"favicon"`
	Duration    int         `json:"duration"`
	Domain      string      `json:"domain"`
	URL         string      `json:"url"`
	Videos      []WebMedia  `json:"videos"`
	Locale      string      `json:"locale,omitempty"`
	Determiner  string      `json:"determiner,omitempty"`
	Raw         interface{} `json:"raw,omitempty"`
	ThemeColor  string      `json:"themeColor,omitempty"`
}

// MetadataOutput defines the output for metadata extraction
type MetadataOutput struct {
	Body struct {
		Metadata []MetadataItem `json:"metadata" doc:"Extracted metadata for each URL"`
	}
}

// ExtractMetadata handles the POST /metadata endpoint
func (h *MetadataHandler) ExtractMetadata(ctx context.Context, input *MetadataInput) (*MetadataOutput, error) {
	if len(input.Body.URLs) == 0 {
		return nil, huma.Error400BadRequest("No URLs provided")
	}

	// Process URLs concurrently
	var wg sync.WaitGroup
	results := make([]MetadataItem, len(input.Body.URLs))
	
	for i, url := range input.Body.URLs {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			metadata := h.extractMetadataFromURL(targetURL)
			results[idx] = metadata
		}(i, url)
	}
	
	wg.Wait()

	output := &MetadataOutput{}
	output.Body.Metadata = results
	return output, nil
}

// extractMetadataFromURL extracts metadata from a single URL
func (h *MetadataHandler) extractMetadataFromURL(targetURL string) MetadataItem {
	// Basic validation
	if targetURL == "" || targetURL == "http://" || targetURL == "://" || targetURL == "about:blank" {
		return MetadataItem{URL: targetURL}
	}

	// Use colly like the old implementation
	c := colly.NewCollector(
		colly.UserAgent("facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"),
	)

	metadata := MetadataItem{
		URL:    targetURL,
		Images: []WebMedia{},
		Videos: []WebMedia{},
	}

	// Extract Open Graph tags
	c.OnHTML("meta", func(e *colly.HTMLElement) {
		property := e.Attr("property")
		content := e.Attr("content")
		name := e.Attr("name")
		
		if (property == "" && name == "") || content == "" {
			return
		}

		// Theme color
		if name == "theme-color" {
			metadata.ThemeColor = content
		}

		// Open Graph tags
		parts := strings.Split(property, ":")
		if len(parts) < 2 || parts[0] != "og" {
			return
		}

		switch property {
		case "og:title":
			metadata.Title = content
		case "og:description":
			metadata.Description = content
		case "og:site_name":
			metadata.Sitename = content
		case "og:url":
			metadata.URL = content
		case "og:type":
			metadata.Type = content
		case "og:locale":
			metadata.Locale = content
		case "og:determiner":
			metadata.Determiner = content
			
		// Images
		case "og:image":
			metadata.Images = append(metadata.Images, WebMedia{URL: content})
		case "og:image:width", "og:image:height", "og:image:alt", "og:image:type", "og:image:secure_url":
			if len(metadata.Images) > 0 {
				idx := len(metadata.Images) - 1
				switch property {
				case "og:image:width":
					if w, err := strconv.Atoi(content); err == nil {
						metadata.Images[idx].Width = w
					}
				case "og:image:height":
					if h, err := strconv.Atoi(content); err == nil {
						metadata.Images[idx].Height = h
					}
				case "og:image:alt":
					metadata.Images[idx].Alt = content
				case "og:image:type":
					metadata.Images[idx].Type = content
				case "og:image:secure_url":
					metadata.Images[idx].SecureURL = content
				}
			}
			
		// Videos
		case "og:video:url":
			metadata.Videos = append(metadata.Videos, WebMedia{URL: content})
		case "og:video:width", "og:video:height", "og:video:type", "og:video:secure_url", "og:video:duration", "og:video:release_date", "og:video:tag":
			if len(metadata.Videos) > 0 {
				idx := len(metadata.Videos) - 1
				switch property {
				case "og:video:width":
					if w, err := strconv.Atoi(content); err == nil {
						metadata.Videos[idx].Width = w
					}
				case "og:video:height":
					if h, err := strconv.Atoi(content); err == nil {
						metadata.Videos[idx].Height = h
					}
				case "og:video:type":
					metadata.Videos[idx].Type = content
				case "og:video:secure_url":
					metadata.Videos[idx].SecureURL = content
				case "og:video:duration":
					if d, err := strconv.Atoi(content); err == nil {
						metadata.Videos[idx].Duration = d
					}
				case "og:video:release_date":
					metadata.Videos[idx].ReleaseDate = content
				case "og:video:tag":
					tags := strings.Split(content, ",")
					for _, t := range tags {
						metadata.Videos[idx].Tags = append(metadata.Videos[idx].Tags, strings.TrimSpace(t))
					}
				}
			}
		}
	})

	// Extract JSON-LD
	c.OnHTML("script[type='application/ld+json']", func(e *colly.HTMLElement) {
		rawJSON := e.Text
		var ldData interface{}
		if err := json.Unmarshal([]byte(rawJSON), &ldData); err == nil {
			metadata.Raw = ldData
		}
	})

	// Domain detection
	c.OnRequest(func(r *colly.Request) {
		if metadata.Domain == "" {
			if parsedURL, err := url.Parse(r.URL.String()); err == nil {
				metadata.Domain = parsedURL.Host
			}
		}
	})

	// Favicon discovery
	c.OnHTML("head", func(e *colly.HTMLElement) {
		if metadata.Favicon == "" {
			e.DOM.Find("link[rel]").Each(func(_ int, s *goquery.Selection) {
				rel := s.AttrOr("rel", "")
				href := s.AttrOr("href", "")
				relValues := strings.Fields(rel)
				for _, rv := range relValues {
					if rv == "icon" || rv == "shortcut" || rv == "apple-touch-icon" {
						if href != "" {
							metadata.Favicon = e.Request.AbsoluteURL(href)
							return
						}
					}
				}
			})
		}
	})

	// Visit the page
	_ = c.Visit(targetURL)

	return metadata
}