// ABOUTME: Service layer implementation for reader view extraction
// ABOUTME: Handles article content extraction using go-readability

package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"

	md "github.com/JohannesKaufmann/html-to-markdown"
	readability "github.com/go-shiori/go-readability"
)

type Service struct {
	cache  interfaces.Cache
	logger interfaces.Logger
}

func NewService(cache interfaces.Cache, logger interfaces.Logger) *Service {
	return &Service{
		cache:  cache,
		logger: logger,
	}
}

// ExtractReaderViews extracts clean article content from multiple URLs
func (s *Service) ExtractReaderViews(ctx context.Context, urls []string) []domain.ReaderView {
	results := make([]domain.ReaderView, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(index int, url string) {
			defer wg.Done()
			
			// Check cache first
			if s.cache != nil {
				cacheKey := fmt.Sprintf("reader:%s", url)
				if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
					var cachedView domain.ReaderView
					if err := json.Unmarshal(data, &cachedView); err == nil {
						results[index] = cachedView
						return
					}
				}
			}

			// Extract reader view
			view := s.extractSingleView(url)
			results[index] = view

			// Cache successful results
			if s.cache != nil && view.Status == "ok" {
				cacheKey := fmt.Sprintf("reader:%s", url)
				if data, err := json.Marshal(view); err == nil {
					_ = s.cache.Set(ctx, cacheKey, data, 1*time.Hour)
				}
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

func (s *Service) extractSingleView(url string) domain.ReaderView {
	result := domain.ReaderView{
		URL:    url,
		Status: "ok",
	}

	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		s.logger.Error("Failed to parse reader view", map[string]interface{}{
			"url":   url,
			"error": err.Error(),
		})
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	result.Title = article.Title
	result.Content = article.Content
	result.TextContent = article.TextContent
	result.SiteName = article.SiteName
	result.Image = article.Image
	result.Favicon = article.Favicon

	// Convert HTML content to Markdown
	if result.Content != "" {
		converter := md.NewConverter("", true, nil)
		markdown, err := converter.ConvertString(result.Content)
		if err != nil {
			s.logger.Debug("Failed to convert HTML to markdown", map[string]interface{}{
				"url":   url,
				"error": err.Error(),
			})
			// Don't fail the entire request if markdown conversion fails
			result.Markdown = ""
		} else {
			// Build markdown with metadata
			result.Markdown = buildMarkdownWithMetadata(result.Title, article.Byline, "", result.SiteName, markdown)
		}
	}

	return result
}

// buildMarkdownWithMetadata creates a well-formatted markdown document with metadata
func buildMarkdownWithMetadata(title, author, publishedTime, siteName, content string) string {
	var markdown strings.Builder
	
	// Add title as H1
	if title != "" {
		markdown.WriteString("# ")
		markdown.WriteString(title)
		markdown.WriteString("\n\n")
	}
	
	// Add metadata section
	var metadataItems []string
	
	if author != "" {
		metadataItems = append(metadataItems, fmt.Sprintf("**Author:** %s", author))
	}
	
	if publishedTime != "" {
		// Try to parse and format the time nicely
		if parsedTime, err := time.Parse(time.RFC3339, publishedTime); err == nil {
			formattedTime := parsedTime.Format("January 2, 2006 at 3:04 PM")
			metadataItems = append(metadataItems, fmt.Sprintf("**Published:** %s", formattedTime))
		} else {
			// If parsing fails, use the raw string
			metadataItems = append(metadataItems, fmt.Sprintf("**Published:** %s", publishedTime))
		}
	}
	
	if siteName != "" {
		metadataItems = append(metadataItems, fmt.Sprintf("**Source:** %s", siteName))
	}
	
	if len(metadataItems) > 0 {
		markdown.WriteString(strings.Join(metadataItems, " | "))
		markdown.WriteString("\n\n---\n\n")
	}
	
	// Add the cleaned content
	markdown.WriteString(cleanMarkdown(content))
	
	return markdown.String()
}

// cleanMarkdown removes excessive newlines and cleans up markdown formatting
func cleanMarkdown(markdown string) string {
	// First handle Windows-style line endings
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")
	
	// Replace escaped newlines with actual newlines
	markdown = strings.ReplaceAll(markdown, "\\n\\n", "\n\n")
	markdown = strings.ReplaceAll(markdown, "\\n", "\n")
	
	// Replace multiple consecutive newlines with double newlines
	re := regexp.MustCompile(`\n{3,}`)
	markdown = re.ReplaceAllString(markdown, "\n\n")
	
	// Clean up spaces before/after newlines
	markdown = regexp.MustCompile(`[ \t]+\n`).ReplaceAllString(markdown, "\n")
	markdown = regexp.MustCompile(`\n[ \t]+`).ReplaceAllString(markdown, "\n")
	
	// Ensure proper spacing around headers
	markdown = regexp.MustCompile(`\n(#{1,6} )`).ReplaceAllString(markdown, "\n\n$1")
	markdown = regexp.MustCompile(`(#{1,6} [^\n]+)\n([^\n])`).ReplaceAllString(markdown, "$1\n\n$2")
	
	// Trim leading and trailing whitespace
	markdown = strings.TrimSpace(markdown)
	
	return markdown
}