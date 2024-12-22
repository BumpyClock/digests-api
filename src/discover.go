package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type Urls struct {
	Urls []string `json:"urls"`
}

var (
	feedResultPool = &sync.Pool{
		New: func() interface{} {
			return &FeedResult{}
		},
	}
)

type FeedResult struct {
	URL      string `json:"url"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	FeedLink string `json:"feedLink"`
}

func discoverRssFeedUrl(urlStr string) (string, error) {
	if strings.HasPrefix(urlStr, "https://github.com") {
		return generateGitHubRssUrl(urlStr), nil
	}

	if strings.HasPrefix(urlStr, "https://www.reddit.com") {
		return generateRedditRssUrl(urlStr), nil
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
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

func generateGitHubRssUrl(url string) string {
	return strings.TrimRight(url, "/") + "/commits/master.atom"
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
