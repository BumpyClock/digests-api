// ABOUTME: Discover handler for finding RSS feed URLs from regular website URLs
// ABOUTME: Supports automatic RSS discovery from HTML pages and special handling for GitHub/Reddit

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	
	"digests-app-api/core/interfaces"
	"github.com/PuerkitoBio/goquery"
	"github.com/danielgtaylor/huma/v2"
)

// DiscoverHandler handles RSS feed discovery
type DiscoverHandler struct {
	httpClient interfaces.HTTPClient
}

// NewDiscoverHandler creates a new discover handler
func NewDiscoverHandler(httpClient interfaces.HTTPClient) *DiscoverHandler {
	return &DiscoverHandler{
		httpClient: httpClient,
	}
}

// RegisterRoutes registers discover routes
func (h *DiscoverHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "discoverFeeds",
		Method:      http.MethodPost,
		Path:        "/discover",
		Summary:     "Discover RSS feeds from websites",
		Description: "Attempts to discover RSS/Atom feed URLs from provided website URLs",
		Tags:        []string{"Discovery"},
	}, h.DiscoverFeeds)
}

// DiscoverFeedsInput defines the input for feed discovery
type DiscoverFeedsInput struct {
	Body struct {
		URLs []string `json:"urls" doc:"List of website URLs to discover feeds from"`
	}
}

// FeedDiscoveryResult represents a single discovery result
type FeedDiscoveryResult struct {
	URL      string `json:"url" doc:"Original URL that was checked"`
	Status   string `json:"status" doc:"Discovery status: 'ok' or 'error'"`
	FeedLink string `json:"feedLink,omitempty" doc:"Discovered RSS feed URL"`
	Error    string `json:"error,omitempty" doc:"Error message if discovery failed"`
}

// DiscoverFeedsOutput defines the output for feed discovery
type DiscoverFeedsOutput struct {
	Body struct {
		Feeds []FeedDiscoveryResult `json:"feeds" doc:"Discovery results for each URL"`
	}
}

// DiscoverFeeds handles the POST /discover endpoint
func (h *DiscoverHandler) DiscoverFeeds(ctx context.Context, input *DiscoverFeedsInput) (*DiscoverFeedsOutput, error) {
	if len(input.Body.URLs) == 0 {
		return nil, huma.Error400BadRequest("No URLs provided")
	}

	// Process URLs concurrently
	var wg sync.WaitGroup
	results := make([]FeedDiscoveryResult, len(input.Body.URLs))
	
	for i, url := range input.Body.URLs {
		wg.Add(1)
		go func(idx int, siteURL string) {
			defer wg.Done()
			
			feedURL, err := h.discoverFeedURL(ctx, siteURL)
			if err != nil {
				results[idx] = FeedDiscoveryResult{
					URL:    siteURL,
					Status: "error",
					Error:  err.Error(),
				}
			} else {
				results[idx] = FeedDiscoveryResult{
					URL:      siteURL,
					Status:   "ok",
					FeedLink: feedURL,
				}
			}
		}(i, url)
	}
	
	wg.Wait()

	output := &DiscoverFeedsOutput{}
	output.Body.Feeds = results
	return output, nil
}

// discoverFeedURL attempts to discover RSS feed URL from a website
func (h *DiscoverHandler) discoverFeedURL(ctx context.Context, siteURL string) (string, error) {
	// Handle special cases
	if strings.HasPrefix(siteURL, "https://github.com") {
		return h.generateGitHubFeedURL(siteURL), nil
	}
	
	if strings.HasPrefix(siteURL, "https://www.reddit.com") || strings.HasPrefix(siteURL, "https://reddit.com") {
		return h.generateRedditFeedURL(siteURL), nil
	}

	// Fetch the page
	resp, err := h.httpClient.Get(ctx, siteURL)
	if err != nil {
		return "", err
	}
	defer resp.Body().Close()

	if resp.StatusCode() != http.StatusOK {
		return "", errors.New("failed to fetch page")
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body())
	if err != nil {
		return "", err
	}

	// Look for RSS feed links
	var feedURL string
	doc.Find(`link[type="application/rss+xml"], link[type="application/atom+xml"]`).Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && feedURL == "" {
			feedURL = href
		}
	})

	if feedURL == "" {
		return "", errors.New("no RSS feed found")
	}

	// Ensure absolute URL
	feedURL, err = h.ensureAbsoluteURL(siteURL, feedURL)
	if err != nil {
		return "", err
	}

	return feedURL, nil
}

// generateGitHubFeedURL generates RSS feed URL for GitHub repositories
func (h *DiscoverHandler) generateGitHubFeedURL(githubURL string) string {
	// GitHub provides Atom feeds for repository commits
	return strings.TrimRight(githubURL, "/") + "/commits/master.atom"
}

// generateRedditFeedURL generates RSS feed URL for Reddit
func (h *DiscoverHandler) generateRedditFeedURL(redditURL string) string {
	// Reddit provides RSS feeds by appending .rss
	return strings.TrimRight(redditURL, "/") + "/.rss"
}

// ensureAbsoluteURL converts relative URLs to absolute ones
func (h *DiscoverHandler) ensureAbsoluteURL(baseURL, relativeOrAbsoluteURL string) (string, error) {
	u, err := url.Parse(relativeOrAbsoluteURL)
	if err != nil {
		return "", err
	}
	
	if u.IsAbs() {
		return relativeOrAbsoluteURL, nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(u).String(), nil
}