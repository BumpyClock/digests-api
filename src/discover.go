package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Urls struct {
	Urls []string `json:"urls"`
}

func discoverHandler(w http.ResponseWriter, r *http.Request) {
	var urls Urls
	err := json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
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
			if err != nil {
				results[i] = FeedResult{
					URL:      url,
					Status:   "error",
					Error:    err.Error(),
					FeedLink: "",
				}
				return
			}
			results[i] = FeedResult{
				URL:      url,
				Status:   "ok",
				FeedLink: feedLink,
			}
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]FeedResult{"feeds": results})
}

type FeedResult struct {
	URL      string `json:"url"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	FeedLink string `json:"feedLink"`
}

func discoverRssFeedUrl(urlStr string) (string, error) {
	if isGitHubRepo(urlStr) {
		return generateGitHubRssUrl(urlStr), nil
	}

	if isRedditLink(urlStr) {
		return generateRedditRssUrl(urlStr), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	rssLink, exists := doc.Find(`link[type="application/rss+xml"]`).Attr("href")
	if !exists {
		return "", errors.New("RSS feed not found")
	}

	rssLink, err = ensureAbsoluteUrl(urlStr, rssLink)
	if err != nil {
		return "", err
	}

	return rssLink, nil
}

func isGitHubRepo(url string) bool {
	return strings.Contains(url, "github.com")
}

func generateGitHubRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/commits/master.atom"
}

func isRedditLink(url string) bool {
	return strings.Contains(url, "reddit.com")
}

func generateRedditRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/.rss"
}

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
