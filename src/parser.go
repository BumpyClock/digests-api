package main

import (
	"encoding/json"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	URL "net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	readability "github.com/go-shiori/go-readability"
	"github.com/jinzhu/copier"
	"github.com/mmcdole/gofeed"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

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

func isGUID(s string) bool {
	// Check if s is a URL
	_, err := url.ParseRequestURI(s)
	if err == nil {
		// s is a URL, so it's not a GUID
		return false
	}

	// If it's not a URL, we assume it's a GUID
	return true
}

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

func processURL(url string) FeedResponse {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(url)
	if err != nil {
		log.Printf("Failed to parse URL %s: %v", url, err)
		return FeedResponse{}
	}

	feedItems := processFeedItems(feed.Items)
	favicon := getFavicon(feed)
	baseDomain := getBaseDomain(feed.Link)

	return createFeedResponse(feed, url, baseDomain, favicon, feedItems)
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
		article, err := readability.FromURL(item.Link, 30*time.Second)
		if err == nil {
			thumbnail = article.Image
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

	//create a Hash item.GUID if item.GUID is a url to assign it to item.ID
	// Check if item.GUID is already a GUID
	if !isGUID(item.GUID) {
		// item.GUID is not a GUID, check if it's a URL
		_, err := url.ParseRequestURI(item.GUID)
		if err == nil {
			// item.GUID is a URL, create a hash
			hash := sha256.Sum256([]byte(item.GUID))
			item.GUID = hex.EncodeToString(hash[:])
		} else {
			log.Printf("Error parsing GUID: %s %s", err, item.GUID)
		}
	}

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
