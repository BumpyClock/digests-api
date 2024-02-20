package main

import (
	"github.com/gin-gonic/gin"

	"net/http"
	"sync"
	"time"

	readability "github.com/go-shiori/go-readability"
)

func getReaderViewResult(url string) ReaderViewResult {
	readerView, err := getReaderView(url)
	if err != nil {
		log.Errorf("Error getting reader view for %s: %v", url, err)
		return ReaderViewResult{
			URL:    url,
			Status: "error",
			Error:  err,
		}
	} else {
		return ReaderViewResult{
			URL:         url,
			Status:      "ok",
			ReaderView:  readerView.Content,
			Title:       readerView.Title,
			SiteName:    readerView.SiteName,
			Image:       readerView.Image,
			Favicon:     readerView.Favicon,
			TextContent: readerView.TextContent,
		}
	}
}

func getReaderViewHandler(c *gin.Context) {
	var urls Urls
	if err := c.BindJSON(&urls); err != nil {
		log.Printf("Error decoding request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	sem := make(chan bool, numCores)

	var wg sync.WaitGroup
	results := make([]ReaderViewResult, len(urls.Urls))
	for i, url := range urls.Urls {
		wg.Add(1)
		sem <- true // Will block if there is no empty slot.
		go func(i int, url string) {
			defer wg.Done()
			defer func() { <-sem }() // Release the slot.
			cacheKey := createHash(url)
			var result ReaderViewResult
			if err := cache.Get(readerView_prefix, cacheKey, &result); err != nil {
				log.Println("[ReaderView]Cache miss for", url)
				result = getReaderViewResult(url)
				if len(result.TextContent) < 100 || result.TextContent == "Please enable JS and disable any ad blocker" {
					result.TextContent = "<div id=\"readability-page-1\" class=\"page\"><p id=\"cmsg\">Error getting reader view, site is likely requires a subscription. Please open the link in a new tab.</p>\n</div><div><a href=\"" + result.URL + "\" target=\"_blank\" rel=\"noopener noreferrer\">Open link in a new tab</a></div>"
					result.ReaderView = result.TextContent
				}
				if err := cache.Set(readerView_prefix, cacheKey, result, 1*time.Hour); err != nil {
					log.Printf("[ReaderView]Failed to cache reader view for %s: %v", url, err)
				}
			} else {
				log.Println("[ReaderView]Cache hit for", url)
			}
			results[i] = result
		}(i, url)
	}
	wg.Wait()

	c.JSON(http.StatusOK, results)
}

func getReaderView(url string) (readability.Article, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		log.Errorf("failed to parse %s, %v\n", url, err)
		return readability.Article{}, err
	}

	return article, nil
}
