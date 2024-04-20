package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"golang.org/x/net/html"

	"github.com/EdlinOrg/prominentcolor"
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

type jsonLinkResponseImage struct {
	Images []string `json:"images"`
}

func (tf *ThumbnailFinder) fetchImageFromSource(pageURL string) (string, error) {
	// Prepare the API URL with the required parameters.
	apiURL := fmt.Sprintf("https://link2json.azurewebsites.net/extract?url=%s", pageURL)

	// Send the GET request.
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf(`[Thumbnail Discovery] Error sending GET request: %s`, err)
	} else {
		defer resp.Body.Close()

		// Decode the response.
		var response jsonLinkResponseImage
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Printf(`[Thumbnail Discovery] Error decoding response: %s`, err)
		} else if len(response.Images) > 0 {
			// Get the thumbnail from the response.
			// log.Printf(`[Thumbnail Discovery] Found thumbnail for URL %s: %s`, pageURL, response.Images[0])
			return response.Images[0], nil
		}
	}

	// Fallback to previous method if no thumbnail is found from the API.
	resp, err = http.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("error fetching page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error loading HTTP response body: %w", err)
	}

	// Look for Open Graph image meta tag
	meta := doc.Find("meta[property='og:image']")
	if meta.Length() > 0 {
		content, _ := meta.Attr("content")
		return content, nil
	}

	// Fallback to previous method if no Open Graph image meta tag is found
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

func extractColorFromThumbnail_prominentColor(url string) (r, g, b uint8) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic while processing URL %s: %v", url, r)
			r, g, b = 128, 128, 128
		}
	}()

	if url == "" {
		return 128, 128, 128 // RGB values for gray
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return 128, 128, 128
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 128, 128, 128
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return 128, 128, 128
	}

	resizedImage := imaging.Resize(img, 100, 0, imaging.Lanczos)

	bounds := resizedImage.Bounds()
	imgNRGBA := image.NewNRGBA(bounds)
	draw.Draw(imgNRGBA, bounds, resizedImage, bounds.Min, draw.Src)

	if imgNRGBA == nil {
		log.Printf("imgNRGBA is nil for URL %s", url)
	}

	colors, err := prominentcolor.KmeansWithAll(prominentcolor.ArgumentDefault, imgNRGBA, prominentcolor.DefaultK, 1, prominentcolor.GetDefaultMasks())
	if err != nil || len(colors) == 0 {
		return 128, 128, 128
	}

	if len(colors) > 0 {
		return uint8(colors[0].Color.R), uint8(colors[0].Color.G), uint8(colors[0].Color.B)
	}

	return 128, 128, 128
}

func DiscoverFavicon(pageURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		log.Printf("Error creating request for favicon discovery: %v", err)
		return ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching page for favicon: %v", err)
		return ""
	}
	defer resp.Body.Close()

	// Parse the HTML document
	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Printf("Error parsing HTML for favicon: %v", err)
		return ""
	}

	var favicon string
	var findNodes func(*html.Node)
	findNodes = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			relAttr := ""
			hrefAttr := ""
			for _, attr := range n.Attr {
				if attr.Key == "rel" {
					relAttr = attr.Val
				} else if attr.Key == "href" {
					hrefAttr = attr.Val
				}
			}

			relValues := strings.Fields(relAttr)
			for _, relValue := range relValues {
				if relValue == "icon" || relValue == "shortcut" || relValue == "apple-touch-icon" {
					if hrefAttr != "" {
						favicon = hrefAttr
						return // Found the favicon, no need to continue searching
					}
				}
			}
		}

		for c := n.FirstChild; c != nil && favicon == ""; c = c.NextSibling {
			findNodes(c) // Recursive search
		}
	}

	findNodes(doc)

	// Resolve relative favicon URL
	if favicon != "" && !strings.HasPrefix(favicon, "http") {
		if parsedFaviconURL, err := url.Parse(favicon); err == nil {
			if baseUrl, err := url.Parse(pageURL); err == nil {
				favicon = baseUrl.ResolveReference(parsedFaviconURL).String()
			}
		}
	}

	return favicon
}
