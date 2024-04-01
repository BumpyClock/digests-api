package main

import (
	"encoding/json"
	"net/http"
	"time"

	"math/rand"
)

//createShareHandler handles the /create endpoint

type createShareRequest struct {
	Urls []string `json:"urls"`
}

type fetchShareRequest struct {
	Key string `json:"key"`
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
