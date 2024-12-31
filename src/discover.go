// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

var (
	feedResultPool = &sync.Pool{
		New: func() interface{} {
			return &FeedResult{}
		},
	}
)

/**
 * @function discoverHandler
 * @description Handles HTTP requests to the /discover endpoint.
 *              It expects a POST request with a JSON body containing an array of URLs.
 *              It attempts to discover the RSS feed URL for each provided URL and returns the results as JSON.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 * @dependencies discoverRssFeedUrl, generateGitHubRssUrl, generateRedditRssUrl, ensureAbsoluteUrl, httpClient, log
 */
func discoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		zap.L().Warn("[discoverHandler] Invalid method", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var urls Urls
	err := json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		zap.L().Error("[discoverHandler] Error decoding request body", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	results := make([]FeedResult, len(urls.Urls))
	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			feedLink, err := discoverRssFeedUrl(url)
			result := feedResultPool.Get().(*FeedResult)
			defer feedResultPool.Put(result)
			if err != nil {
				result.URL = url
				result.Status = "error"
				result.Error = err.Error()
				result.FeedLink = ""
			} else {
				result.URL = url
				result.Status = "ok"
				result.FeedLink = feedLink
			}
			results[i] = *result
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]FeedResult{"feeds": results})
}

/**
 * @function discoverRssFeedUrl
 * @description Attempts to discover the RSS feed URL for a given URL.
 *              It first checks if the URL is a GitHub or Reddit URL and generates the feed URL accordingly.
 *              Otherwise, it sends an HTTP GET request to the URL, parses the HTML response,
 *              and searches for link elements with type="application/rss+xml".
 * @param {string} urlStr The URL to discover the RSS feed for.
 * @returns {string, error} The RSS feed URL if found, or an error if not found or an error occurred.
 * @dependencies generateGitHubRssUrl, generateRedditRssUrl, ensureAbsoluteUrl, httpClient, goquery.NewDocumentFromReader, log
 */
func discoverRssFeedUrl(urlStr string) (string, error) {
	if strings.HasPrefix(urlStr, "https://github.com") {
		return generateGitHubRssUrl(urlStr), nil
	}

	if strings.HasPrefix(urlStr, "https://www.reddit.com") {
		return generateRedditRssUrl(urlStr), nil
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		zap.L().Error("[discoverRssFeedUrl] Error creating request", zap.String("url", urlStr), zap.Error(err))
		return "", err
	}

	// Set a custom User-Agent to avoid being blocked by some websites
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	resp, err := httpClient.Do(req)
	if err != nil {
		zap.L().Error("[discoverRssFeedUrl] Error sending request", zap.String("url", urlStr), zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zap.L().Error("[discoverRssFeedUrl] Request failed", zap.String("url", urlStr), zap.Int("statusCode", resp.StatusCode))
		return "", errors.New("request failed")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		zap.L().Error("[discoverRssFeedUrl] Error parsing HTML", zap.String("url", urlStr), zap.Error(err))
		return "", err
	}

	rssLink, exists := doc.Find(`link[type="application/rss+xml"]`).Attr("href")
	if !exists {
		zap.L().Warn("[discoverRssFeedUrl] RSS feed not found in HTML", zap.String("url", urlStr))
		return "", errors.New("RSS feed not found")
	}

	rssLink, err = ensureAbsoluteUrl(urlStr, rssLink)
	if err != nil {
		zap.L().Error("[discoverRssFeedUrl] Error ensuring absolute URL", zap.String("url", urlStr), zap.String("rssLink", rssLink), zap.Error(err))
		return "", err
	}

	return rssLink, nil
}

/**
 * @function generateGitHubRssUrl
 * @description Generates a GitHub RSS feed URL from a GitHub repository URL.
 * @param {string} url The GitHub repository URL.
 * @returns {string} The corresponding GitHub RSS feed URL.
 */
func generateGitHubRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/commits/master.atom"
}

/**
 * @function generateRedditRssUrl
 * @description Generates a Reddit RSS feed URL from a Reddit URL.
 * @param {string} url The Reddit URL.
 * @returns {string} The corresponding Reddit RSS feed URL.
 */
func generateRedditRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/.rss"
}

/**
 * @function ensureAbsoluteUrl
 * @description Ensures that a given URL is absolute. If the URL is relative, it resolves it against the base URL.
 * @param {string} baseUrl The base URL to use for resolving relative URLs.
 * @param {string} relativeOrAbsoluteUrl The URL to ensure is absolute.
 * @returns {string, error} The absolute URL, or an error if the URL is invalid.
 * @dependencies url.Parse, url.Parse
 */
func ensureAbsoluteUrl(baseUrl, relativeOrAbsoluteUrl string) (string, error) {
	u, err := url.Parse(relativeOrAbsoluteUrl)
	if err != nil || !u.IsAbs() {
		u, err = url.Parse(baseUrl)
		if err != nil {
			return "", err
		}
		rel, err := url.Parse(relativeOrAbsoluteUrl)
		if err != nil {
			return "", err
		}
		return u.ResolveReference(rel).String(), nil
	}
	return relativeOrAbsoluteUrl, nil
}
