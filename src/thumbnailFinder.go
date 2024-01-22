package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

type ThumbnailFinder struct {
	cache sync.Map
}

func NewThumbnailFinder() *ThumbnailFinder {
	return &ThumbnailFinder{}
}

func (tf *ThumbnailFinder) FindThumbnailForItem(item *gofeed.Item) string {
	if thumb, ok := tf.cache.Load(item.Link); ok {
		return thumb.(string)
	}

	thumbnail := tf.extractThumbnailFromEnclosures(item.Enclosures)
	if thumbnail != "" {
		return thumbnail
	}

	thumbnail = tf.extractThumbnailFromContent(item.Content)
	if thumbnail != "" {
		return thumbnail
	}

	if item.Link != "" {
		thumbnail, err := tf.fetchImageFromSource(item.Link)
		if err != nil {
			fmt.Println("Error fetching image:", err)
			return ""
		}
		tf.cache.Store(item.Link, thumbnail)
		return thumbnail
	}

	return ""
}

func (tf *ThumbnailFinder) extractThumbnailFromEnclosures(enclosures []*gofeed.Enclosure) string {
	for _, e := range enclosures {
		if strings.HasPrefix(e.Type, "image/") {
			return e.URL
		}
	}
	return ""
}

func (tf *ThumbnailFinder) extractThumbnailFromContent(content string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		fmt.Println("Error parsing content:", err)
		return ""
	}

	if src, exists := doc.Find("img").First().Attr("src"); exists {
		return src
	}
	return ""
}

func (tf *ThumbnailFinder) fetchImageFromSource(pageURL string) (string, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("error fetching page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error loading HTTP response body: %w", err)
	}

	src, exists := doc.Find("article img, .content img").First().Attr("src")
	if exists {
		if !strings.HasPrefix(src, "http") {
			parsedURL, err := url.Parse(pageURL)
			if err != nil {
				return "", err
			}
			return parsedURL.Scheme + "://" + parsedURL.Host + src, nil
		}
		return src, nil
	}
	return "", nil
}

// fetchImageFromSource fetches the given URL and attempts to find an image.
func fetchImageFromSource(pageURL string) (string, error) {
	// Custom logic for specific domains can be added here
	resp, err := http.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("error fetching page: %w", err)
	}
	defer resp.Body.Close()

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error loading HTTP response body: %w", err)
	}

	// Attempt to find an image
	src, exists := doc.Find("article img, .content img").First().Attr("src")
	if exists {
		// Handle relative URLs
		if !strings.HasPrefix(src, "http") {
			parsedURL, err := url.Parse(pageURL)
			if err != nil {
				return "", err
			}
			return parsedURL.Scheme + "://" + parsedURL.Host + src, nil
		}
		return src, nil
	}
	return "", nil
}
