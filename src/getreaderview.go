package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	readability "github.com/go-shiori/go-readability"
)

func getReaderViewHandler(w http.ResponseWriter, r *http.Request) {
	var urls Urls
	err := json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	results := make([]ReaderViewResult, len(urls.Urls))
	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			readerView, err := getReaderView(url)
			if err != nil {
				results[i] = ReaderViewResult{
					URL:    url,
					Status: "error",
					Error:  err,
				}
				return
			}
			results[i] = ReaderViewResult{
				URL:         url,
				Status:      "ok",
				ReaderView:  readerView.Content,
				Title:       readerView.Title,
				SiteName:    readerView.SiteName,
				Image:       readerView.Image,
				Favicon:     readerView.Favicon,
				TextContent: readerView.TextContent,
			}
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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

func getReaderView(url string) (readability.Article, error) {
	article := readability.Article{}
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		log.Fatalf("failed to parse %s, %v\n", url, err)
		return article, err
	}

	return article, nil
}
