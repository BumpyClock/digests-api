package main

import (
	"github.com/mmcdole/gofeed"
)

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
	ID              string              `json:"id"`
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

type ReaderViewResult struct {
	URL         string `json:"url"`
	Status      string `json:"status"`
	ReaderView  string `json:"content"`
	Error       error  `json:"error,omitempty"`
	Title       string `json:"title"`
	SiteName    string `json:"siteName"`
	Image       string `json:"image"`
	Favicon     string `json:"favicon"`
	TextContent string `json:"textContent"`
}

// CONSTANTS
const redis_address = "localhost:6379"
const redis_password = ""
const redis_db = 0

// const redis_feedsItems_key = "feedsItems"
// const redis_feedDetails_key = "feedDetails"
