package main

import (
	"encoding/json"

	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/jinzhu/copier"
	"github.com/mmcdole/gofeed"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ParseRequest represents the expected incoming JSON payload structure.
type ParseRequest struct {
	URLs []string `json:"urls"`
}

type RGBColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// Add these structs to your existing code
type MediaContent struct {
	URL    string `xml:"url,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
}

type ExtendedItem struct {
	*gofeed.Item
	MediaContent []MediaContent `xml:"http://search.yahoo.com/mrss/ content"`
}

// FeedResponseItem represents an enriched structure for an individual feed item.
type FeedResponseItem struct {
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	Link            string              `json:"link"`
	Author          string              `json:"author"`
	Published       string              `json:"published"`
	Content         string              `json:"content"`
	Created         string              `json:"created"`
	Content_Encoded string              `json:"content_encoded"`
	Categories      string              `json:"categories"`
	Enclosures      []*gofeed.Enclosure `json:"enclosures"`
	Thumbnail       string              `json:"thumbnail"`
	ThumbnailColor  RGBColor            `json:"thumbnailColor"`
}

// FeedResponse represents the structure for the overall feed, including metadata and items.
type FeedResponse struct {
	Status        string             `json:"status"`
	SiteTitle     string             `json:"siteTitle"`
	FeedTitle     string             `json:"feedTitle"`
	FeedUrl       string             `json:"feedUrl"`
	Description   string             `json:"description"`
	Link          string             `json:"link"`
	LastUpdated   string             `json:"lastUpdated"`
	LastRefreshed string             `json:"lastRefreshed"`
	Published     string             `json:"published"`
	Author        *gofeed.Person     `json:"author"`
	Language      string             `json:"language"`
	Favicon       string             `json:"favicon"`
	Categories    string             `json:"categories"`
	Items         []FeedResponseItem `json:"items"`
}

type Feeds struct {
	Feeds []FeedResponse `json:"feeds"`
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

// parseHandler processes the POST request to parse specified feed URLs and return detailed feed information.
func parseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	responses := make(chan FeedResponse, len(req.URLs))
	var wg sync.WaitGroup

	for _, url := range req.URLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			parser := gofeed.NewParser()
			feed, err := parser.ParseURL(url)
			if err != nil {
				log.Printf("Failed to parse URL %s: %v", url, err)
				return
			}

			feedItems := make([]FeedResponseItem, 0, len(feed.Items))
			var itemWg sync.WaitGroup
			itemResponses := make(chan FeedResponseItem, len(feed.Items))

			for _, item := range feed.Items {
				itemWg.Add(1)
				go func(item *gofeed.Item) {
					defer itemWg.Done()

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

					if thumbnail == "" {
						if item.Content != "" {
							thumbnail = thumbnailFinder.extractThumbnailFromContent(item.Content) // Extract from content if not found
						} else if item.Description != "" {
							thumbnail = thumbnailFinder.extractThumbnailFromContent(item.Description) // Extract from description if content is empty
						}

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

						if thumbnail == "" {
							thumbnail, err = thumbnailFinder.fetchImageFromSource(item.Link) // Fetch from webpage if not found in content or description
							if err != nil {
								thumbnail = ""
							}
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

					feedItem := FeedResponseItem{
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

					itemResponses <- feedItem
				}(item)
			}

			go func() {
				itemWg.Wait()
				close(itemResponses)
			}()

			for itemResponse := range itemResponses {
				feedItems = append(feedItems, itemResponse)
			}

			favicon := ""
			if feed.Image != nil {
				favicon = feed.Image.URL
			} else {
				favicon = DiscoverFavicon(feed.Link)
			}

			response := FeedResponse{
				Status:        "ok",
				SiteTitle:     feed.Title,
				FeedTitle:     feed.Title,
				FeedUrl:       url,
				Description:   feed.Description,
				Link:          feed.Link,
				LastUpdated:   feed.Updated,
				LastRefreshed: time.Now().Format(time.RFC3339),
				Published:     feed.Published,
				Author:        feed.Author,
				Language:      feed.Language,
				Favicon:       favicon,
				Categories:    strings.Join(feed.Categories, ", "),
				Items:         feedItems,
			}

			responses <- response
		}(url)
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	// Collect all responses
	var allResponses []FeedResponse
	for response := range responses {
		allResponses = append(allResponses, response)
	}

	feeds := Feeds{Feeds: allResponses}

	// Encode all responses as a single JSON array
	if err := enc.Encode(feeds); err != nil {
		log.Printf("Failed to encode response: %v", err)
		return
	}
}
