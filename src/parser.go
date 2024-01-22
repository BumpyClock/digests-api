package main

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/cascax/colorthief-go"
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

// FeedResponseItem represents an enriched structure for an individual feed item.
type FeedResponseItem struct {
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	Link            string              `json:"link"`
	Author          string              `json:"author"`
	Published       string              `json:"published"`
	Content         string              `json:"content,omitempty"`
	Content_Encoded string              `json:"content_encoded,omitempty"`
	Categories      string              `json:"categories,omitempty"`
	Enclosures      []*gofeed.Enclosure `json:"enclosures,omitempty"`
	Thumbnail       string              `json:"thumbnail,omitempty"`
	ThumbnailColor  RGBColor            `json:"thumbnailColor,omitempty"`
}

// FeedResponse represents the structure for the overall feed, including metadata and items.
type FeedResponse struct {
	SiteTitle   string             `json:"siteTitle"`
	FeedTitle   string             `json:"feedTitle"`
	Description string             `json:"description"`
	Link        string             `json:"link"`
	Updated     string             `json:"updated,omitempty"`
	Published   string             `json:"published,omitempty"`
	Author      *gofeed.Person     `json:"author,omitempty"`
	Language    string             `json:"language,omitempty"`
	Image       *gofeed.Image      `json:"image,omitempty"`
	Categories  string             `json:"categories,omitempty"`
	Items       []FeedResponseItem `json:"items"`
}

func extractColorFromThumbnail_prominentColor(url string) (r, g, b uint8) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, 0
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, 0, 0
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return 0, 0, 0
	}

	// Convert image.Image to *image.NRGBA
	bounds := img.Bounds()
	imgNRGBA := image.NewNRGBA(bounds)
	draw.Draw(imgNRGBA, bounds, img, bounds.Min, draw.Src)

	// Get the most prominent color
	colors, err := prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, prominentcolor.GetDefaultMasks())
	if err != nil || len(colors) == 0 {
		return 0, 0, 0
	}

	// Return the RGB components of the most prominent color
	return uint8(colors[0].Color.R), uint8(colors[0].Color.G), uint8(colors[0].Color.B)
}

func extractColorFromThumbnail_colorThief(url string) (r, g, b uint8) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, 0
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return 0, 0, 0
	}

	// Use colorthief-go to get the dominant color as a color.Color
	dominantColor, err := colorthief.GetColor(img)
	if err != nil {
		return 0, 0, 0
	}

	// Convert color.Color to color.RGBA
	rgba := color.RGBAModel.Convert(dominantColor).(color.RGBA)

	return rgba.R, rgba.G, rgba.B
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
				return // Handle the error as needed
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

						if thumbnail == "" {
							thumbnail, err = thumbnailFinder.fetchImageFromSource(item.Link) // Fetch from webpage if not found in content or description
							if err != nil {
								thumbnail = ""
							}
						}
					}
					thumbnailColor := RGBColor{0, 0, 0}
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

			response := FeedResponse{
				SiteTitle:   feed.Title,
				FeedTitle:   feed.Title,
				Description: feed.Description,
				Link:        feed.Link,
				Updated:     feed.Updated,
				Published:   feed.Published,
				Author:      feed.Author,
				Language:    feed.Language,
				Image:       feed.Image,
				Categories:  strings.Join(feed.Categories, ", "),
				Items:       feedItems,
			}

			responses <- response
		}(url)
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	var feedResponses []FeedResponse
	for response := range responses {
		feedResponses = append(feedResponses, response)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feedResponses)
}
