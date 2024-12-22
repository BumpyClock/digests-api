package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

// parseHandler is an HTTP handler for the /parse endpoint,
// expecting a POST request with a JSON body of feed URLs to parse,
// plus optional pagination parameters (Page, ItemsPerPage).
func parseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Default values for pagination
	page := 1
	itemsPerPage := 50

	// Decode request into ParseRequest
	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If user provided a page > 0, use it; otherwise keep default
	if req.Page > 0 {
		page = req.Page
	}
	// If user provided itemsPerPage > 0, use it; otherwise keep default
	if req.ItemsPerPage > 0 {
		itemsPerPage = req.ItemsPerPage
	}

	responses := processURLs(req.URLs, page, itemsPerPage)
	sendResponse(w, responses)
}

// validateURLsHandler handles the /validate endpoint
func validateURLsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req URLValidationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	statuses := make([]URLStatus, len(req.URLs))
	for i, url := range req.URLs {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			resp, err := http.Get(url)
			if resp != nil {
				defer resp.Body.Close()
			}
			status := "ok"
			if err != nil || resp.StatusCode != http.StatusOK {
				status = "error"
				if err == nil {
					status = http.StatusText(resp.StatusCode)
				}
			}
			statuses[i] = URLStatus{URL: url, Status: status}
		}(i, url)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(statuses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func discoverHandler(w http.ResponseWriter, r *http.Request) {
	var urls Urls
	err := json.NewDecoder(r.Body).Decode(&urls)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	results := make([]FeedResult, len(urls.Urls))
	for i, url := range urls.Urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			feedLink, err := discoverRssFeedUrl(url)
			result := feedResultPool.Get().(*FeedResult)
			defer feedResultPool.Put(result)
			if err != nil {
				result.URL = url
				result.Status = "error"
				result.Error = err.Error()
				result.FeedLink = ""
			} else {
				result.URL = url
				result.Status = "ok"
				result.FeedLink = feedLink
			}
			results[i] = *result
		}(i, url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]FeedResult{"feeds": results})
}

// URLValidationRequest represents the request format for URL validation
type URLValidationRequest struct {
	URLs []string `json:"urls"`
}

// URLStatus represents the status of a single URL validation
type URLStatus struct {
	URL    string `json:"url"`
	Status string `json:"status"`
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

func createShareHandler(w http.ResponseWriter, r *http.Request) {
	var req createShareRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error("Error decoding request body: ", err)
		return
	}

	if req.Urls == nil || len(req.Urls) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		response := map[string]string{
			"status": "error",
			"error":  "No URLs provided",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Info("Create request received for URLs: ", req.Urls)

	// Generate a random key of maximum 6 characters
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomKey := make([]rune, 6)
	for i := range randomKey {
		randomKey[i] = chars[rand.Intn(len(chars))]
	}

	cacheKey := "share:" + string(randomKey)

	// Save the URLs to the cache
	err = cache.Set(cacheKey, "urls", req.Urls, 0) // setting exp 0 to keep it forever
	if err != nil {
		http.Error(w, "Failed to save shared URLs", http.StatusInternalServerError)
		return
	}

	// Respond with the link
	response := map[string]string{"status": "ok", "link": "https://www.digests.app/share/" + string(randomKey)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func shareHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Share request received")

	// Decode the request body
	var req fetchShareRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Error("Error decoding request body: ", err)
		return
	}

	// Get the key from the request body
	cacheKey := "share:" + req.Key

	// Get the URLs from the cache
	var urls []string
	err = cache.Get(cacheKey, "urls", &urls)
	if err != nil {
		http.Error(w, "Invalid share link", http.StatusBadRequest)
		log.Error("Error getting URLs from cache: ", err)
		return
	}

	// Respond with the URLs
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(urls)

	log.Info("Share request received for key: ", req.Key)
	log.Info("Returning URLs: ", urls)
}

// searchHandler handles the /search endpoint
func searchHandler(w http.ResponseWriter, r *http.Request) {
	// Get the 'url' query parameter
	searchType := r.URL.Query().Get("type")
	if searchType == "rss" {
		queryURL := r.URL.Query().Get("q")
		if queryURL == "" {
			http.Error(w, "No url provided", http.StatusBadRequest)
			response := map[string]string{
				"status": "error",
				"error":  "No url provided",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		} else {
			var searchResults []FeedSearchResponseItem = searchRSS(queryURL)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(searchResults)
			return

		}

	} else if searchType == "podcast" {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "No query provided", http.StatusBadRequest)
			response := map[string]string{
				"status": "error",
				"error":  "No query provided",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		} else {
			var searchResults []PodcastSearchResponseItem = searchPodcast(r, query)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(searchResults)
			return
		}

	} else {
		http.Error(w, "No or invalid type provided", http.StatusBadRequest)
		response := map[string]string{
			"status": "error",
			"error":  "No or invalid type provided",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

}

func streamAudioHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Received request to stream audio")

	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the request body
	var ttsReq TTSRequest
	err := json.NewDecoder(r.Body).Decode(&ttsReq)
	if err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Print("Request: ", ttsReq.Text)
	log.Print("Request URL: ", ttsReq.Url)

	// Check if text is provided
	if ttsReq.Text == "" {
		http.Error(w, "No text provided", http.StatusBadRequest)
		return
	} else if ttsReq.Url != "" {
		// Check if the URL is valid
		if !(strings.HasPrefix(ttsReq.Url, "http://") || strings.HasPrefix(ttsReq.Url, "https://")) {
			http.Error(w, "Invalid URL provided", http.StatusBadRequest)
			return
		}
	}

	cacheKey := ttsReq.Url

	var cachedAudio []byte
	// Check if the audio content is cached
	err = cache.Get(audio_prefix, cacheKey, &cachedAudio)
	if err == nil {
		log.Print("Audio content found in cache")
		// Set the headers and write the audio content to the response
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", fmt.Sprint(len(cachedAudio)))

		// Write the audio content to the response
		_, err = w.Write(cachedAudio)

		if err != nil {
			log.Printf("Failed to write audio content to response: %v", err)
		}
		return
	}

	// Initialize the TTS client once
	once.Do(initTTSClient)

	log.Print("Text to be synthesized: ", ttsReq.Text)
	const maxChunkSize = 1000

	// Split text into chunks of up to 1000 characters
	chunks := splitTextIntoChunks(ttsReq.Text, maxChunkSize)

	var audioContent bytes.Buffer

	for _, chunk := range chunks {
		req := texttospeechpb.SynthesizeSpeechRequest{
			// Set the text input to be synthesized.
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: chunk},
			},
			// Build the voice request, select the language code ("en-US") and the SSML
			// voice gender ("neutral").
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: "en-US",
				Name:         "en-US-Neural2-J",
			},
			// Select the type of audio file you want returned.
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding: *texttospeechpb.AudioEncoding_MP3.Enum(),
			},
		}
		// Perform the text-to-speech request
		resp, err := ttsClient.SynthesizeSpeech(context.Background(), &req)
		if err != nil {
			log.Printf("Failed to synthesize speech: %v", err)
			http.Error(w, "Failed to synthesize speech", http.StatusInternalServerError)
			return
		} else {
			log.Print("Speech synthesized successfully")
		}

		// Append the audio content to the buffer
		audioContent.Write(resp.AudioContent)
	}

	// Cache the audio content
	if err := cache.Set(audio_prefix, cacheKey, audioContent.Bytes(), 7*24*time.Hour); err != nil {
		log.Printf("Failed to cache audio content: %v", err)
	} else {
		log.Print("Audio content cached successfully for url: ", cacheKey)
	}

	// Set the headers and write the audio content to the response
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprint(audioContent.Len()))

	// Write the audio content to the response
	_, err = w.Write(audioContent.Bytes())
	if err != nil {
		log.Printf("Failed to write audio content to response: %v", err)
	}
}
