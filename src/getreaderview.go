package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	readability "github.com/go-shiori/go-readability"
)

func getReaderViewResult(url string) ReaderViewResult {
	readerView, err := getReaderView(url)
	if err != nil {
		log.Errorf("[ReaderView] Error retrieving content for %s: %v", url, err)
		return ReaderViewResult{
			URL:    url,
			Status: "error",
			Error:  err,
		}
	}
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

func getReaderViewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warnf("[ReaderView] Invalid method: %s", r.Method)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var urls Urls
	if err := json.NewDecoder(r.Body).Decode(&urls); err != nil {
		log.Errorf("[ReaderView] Error decoding request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	results := make([]ReaderViewResult, len(urls.Urls))

	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()

			cacheKey := createHash(url)
			var result ReaderViewResult

			if err := cache.Get(readerView_prefix, cacheKey, &result); err != nil {
				log.Infof("[ReaderView] Cache miss for %s", url)
				result = getReaderViewResult(url)
				if len(result.TextContent) < 100 {
					result.TextContent = `<div id="readability-page-1" class="page"><p id="cmsg">Error getting reader view or site requires subscription. Please open the link in a new tab.</p></div>`
					result.ReaderView = result.TextContent
				}
				if err := cache.Set(readerView_prefix, cacheKey, result, 24*time.Hour); err != nil {
					log.Errorf("[ReaderView] Failed to cache reader view for %s: %v", url, err)
				}
			} else {
				log.Infof("[ReaderView] Cache hit for %s", url)
			}
			results[i] = result
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Errorf("[ReaderView] Failed to encode JSON: %v", err)
	}
}

func getReaderView(url string) (readability.Article, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		log.Errorf("[ReaderView] Failed to parse %s: %v", url, err)
		return readability.Article{}, err
	}
	return article, nil
}
