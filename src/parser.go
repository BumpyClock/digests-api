package main

import (
	"encoding/json"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"crypto/sha256"
	"encoding/hex"
	"net/http"
	URL "net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	readability "github.com/go-shiori/go-readability"
	"github.com/jinzhu/copier"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func createHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

func parseHTMLContent(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent // Return the original content if parsing fails
	}

	var f func(*html.Node)
	var textContent strings.Builder
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			textContent.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return textContent.String()
}

// func isGUID(s string) bool {
// 	// Check if s is a URL
// 	_, err := URL.ParseRequestURI(s)
// 	if err == nil {
// 		// s is a URL, so it's not a GUID
// 		return false
// 	}

// 	// If it's not a URL, we assume it's a GUID
// 	return true
// }

func getBaseDomain(url string) string {
	parsedURL, err := URL.Parse(url)
	if err != nil {
		return ""
	}

	return parsedURL.Scheme + "://" + parsedURL.Host
}

// parseHandler processes the POST request to parse specified feed URLs and return detailed feed information.
func parseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	responses := processURLs(req.URLs)

	sendResponse(w, responses)
}

func decodeRequest(r *http.Request) (ParseRequest, error) {
	var req ParseRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func processURLs(urls []string) []FeedResponse {
	responses := make(chan FeedResponse, len(urls))
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			response := processURL(url)
			responses <- response
		}(url)
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	return collectResponses(responses)
}

func isCacheStale(lastRefreshed string) bool {
	layout := "2006-01-02T15:04:05Z" // updated time format
	parsedTime, err := time.Parse(layout, lastRefreshed)
	if err != nil {
		log.Printf("Failed to parse LastRefreshed: %v", err)
		return false
	}

	if time.Since(parsedTime) > time.Duration(refresh_timer)*time.Minute {
		log.Println("[Cache Stale] Cache is stale")
		return true
	} else {
		return false
	}
}

func processURL(url string) FeedResponse {
	// make sure the url starts with https:// or http:// if it starts with http:// then convert to https://
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	} else if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "https://", 1)
	}

	cacheKey := createHash(url)

	var cachedFeed FeedResponse
	err := cache.Get(feed_prefix, cacheKey, &cachedFeed)
	if err == nil && cachedFeed.SiteTitle != "" {
		log.WithFields(logrus.Fields{
			"url": url,
		}).Info("[Cache Hit] Using cached feed details")
		// cachedFeed.LastRefreshed is older than 15 minutes the cache is stale and we should refresh it
		if isCacheStale(cachedFeed.LastRefreshed) {
			log.WithFields(logrus.Fields{
				"url": url,
			}).Info("[Cache Stale] Cache is stale")
		} else {
			log.WithFields(logrus.Fields{
				"url": url,
			}).Info("[Cache Hit] Cache is fresh")
			return cachedFeed
		}
	} else {
		log.WithFields(logrus.Fields{
			"url": url,
		}).Info("[Cache Miss] Cache miss")
	}

	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(url)
	if err != nil {
		log.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Error("Failed to parse URL")
		return FeedResponse{}
	}

	feedItems := processFeedItems(feed.Items)
	favicon := getFavicon(feed)
	baseDomain := getBaseDomain(feed.Link)
	addURLToList(url)

	response := createFeedResponse(feed, url, baseDomain, favicon, feedItems)

	// Cache the new feed details and items
	if err := cache.Set(feed_prefix, cacheKey, response, 24*time.Hour); err != nil {
		log.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Error("Failed to cache feed details")
	}

	return response
}

func processFeedItems(items []*gofeed.Item) []FeedResponseItem {

	itemResponses := make(chan FeedResponseItem, len(items))
	var itemWg sync.WaitGroup

	for _, item := range items {
		itemWg.Add(1)
		go func(item *gofeed.Item) {
			defer itemWg.Done()
			itemResponse := processFeedItem(item)
			itemResponses <- itemResponse
		}(item)
	}

	go func() {
		itemWg.Wait()
		close(itemResponses)
	}()

	return collectItemResponses(itemResponses)
}

func processFeedItem(item *gofeed.Item) FeedResponseItem {
	author := ""
	if item.Author != nil {
		author = item.Author.Name
	}

	categories := strings.Join(item.Categories, ", ")

	thumbnail := ""
	if len(item.Enclosures) > 0 {
		thumbnail = item.Enclosures[0].URL // Use the first enclosure as the thumbnail
	}

	thumbnailFinder := NewThumbnailFinder() // Initialize the ThumbnailFinder

	// Extract thumbnail from content or description if not found
	if thumbnail == "" {
		if item.Content != "" {
			thumbnail = thumbnailFinder.extractThumbnailFromContent(item.Content)
		} else if item.Description != "" {
			thumbnail = thumbnailFinder.extractThumbnailFromContent(item.Description)
		}
	}

	// Extract thumbnail from media content if not found
	if thumbnail == "" {
		extItem := &ExtendedItem{}
		_ = copier.Copy(extItem, item) // Use the copier package to copy the data

		if media, ok := extItem.Extensions["media"]; ok {
			if content, ok := media["content"]; ok && len(content) > 0 {
				if url, ok := content[0].Attrs["url"]; ok {
					extItem.MediaContent = append(extItem.MediaContent, MediaContent{URL: url})
					thumbnail = url // Use the first media content as the thumbnail
				}
			}
		}
	}

	// Extract thumbnail from article if not found
	if thumbnail == "" {
		cacheKey := createHash(item.Link)
		tempReaderViewResult := getReaderViewResult(item.Link)
		thumbnail = tempReaderViewResult.Image
		if err := cache.Set(readerView_prefix, cacheKey, tempReaderViewResult, 24*time.Hour); err != nil {
			log.Printf("Failed to cache feed details for %s: %v", item.Link, err)
		}

	}

	thumbnailColor := RGBColor{128, 128, 128}
	if thumbnail != "" {
		r, g, b := extractColorFromThumbnail_prominentColor(thumbnail)
		thumbnailColor = RGBColor{r, g, b}
	}

	parsedContent := parseHTMLContent(item.Content)

	description := item.Description
	if description == "" {
		description = parsedContent
	}

	item.GUID = createHash(item.Link)

	return FeedResponseItem{
		ID:              item.GUID,
		Title:           item.Title,
		Description:     description,
		Link:            item.Link,
		Author:          author,
		Published:       item.Published,
		Created:         item.Published,
		Content:         parsedContent,
		Content_Encoded: item.Content,
		Categories:      categories,
		Enclosures:      item.Enclosures,
		Thumbnail:       thumbnail,
		ThumbnailColor:  thumbnailColor,
	}
}

func getFavicon(feed *gofeed.Feed) string {
	favicon := ""
	if feed.Image != nil {
		favicon = feed.Image.URL
	} else {
		baseDomain := getBaseDomain(feed.Link)

		article, err := readability.FromURL(baseDomain, 10*time.Second)
		if err == nil {
			favicon = article.Favicon
		} else {
			log.Printf(`[Favicon Discovery] Error getting favicon from readability: %s`, err)
		}
	}

	if favicon == "" {
		favicon = DiscoverFavicon(feed.Link)
	}

	return favicon
}

func createFeedResponse(feed *gofeed.Feed, url, baseDomain, favicon string, feedItems []FeedResponseItem) FeedResponse {
	return FeedResponse{
		Status:        "ok",
		GUID:          createHash(url),
		SiteTitle:     feed.Title,
		FeedTitle:     feed.Title,
		FeedUrl:       url,
		Description:   feed.Description,
		Link:          baseDomain,
		LastUpdated:   feed.Updated,
		LastRefreshed: time.Now().Format(time.RFC3339),
		Published:     feed.Published,
		Author:        feed.Author,
		Language:      feed.Language,
		Favicon:       favicon,
		Categories:    strings.Join(feed.Categories, ", "),
		Items:         feedItems,
	}
}

func collectResponses(responses chan FeedResponse) []FeedResponse {
	var allResponses []FeedResponse
	for response := range responses {
		allResponses = append(allResponses, response)
	}
	return allResponses
}

func collectItemResponses(itemResponses chan FeedResponseItem) []FeedResponseItem {
	var feedItems []FeedResponseItem
	for itemResponse := range itemResponses {
		feedItems = append(feedItems, itemResponse)
	}
	return feedItems
}

func sendResponse(w http.ResponseWriter, responses []FeedResponse) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	feeds := Feeds{Feeds: responses}

	if err := enc.Encode(feeds); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// Feed refresh logic

func refreshFeeds() {
	urls := getAllCachedURLs() // This function should return all URLs from the cache

	for _, url := range urls {
		log.Printf("Refreshing feed for URL: %s", url)
		_ = processURL(url) // Refresh the feed and ignore the result
	}
}

func addURLToList(url string) {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	urlList = append(urlList, url)
}

// func removeURLFromList(url string) {
// 	urlListMutex.Lock()
// 	defer urlListMutex.Unlock()

// 	for i, u := range urlList {
// 		if u == url {
// 			urlList = append(urlList[:i], urlList[i+1:]...)
// 			break
// 		}
// 	}
// }

func getAllCachedURLs() []string {
	urlListMutex.Lock()
	defer urlListMutex.Unlock()

	if len(urlList) == 0 {
		startTime := time.Now() // Record the start time

		var err error
		urlList, err = cache.GetSubscribedListsFromCache(feed_prefix)
		if err != nil {
			// Handle the error
			log.Println("Failed to get subscribed lists from cache:", err)
			return nil
		}

		duration := time.Since(startTime) // Calculate the duration

		// Log the urlList and the duration
		log.Infof("Loaded urlList from cache: %s", urlList)
		log.Infof("Time taken to load urlList: %s", duration)
	}

	return append([]string(nil), urlList...)
}
