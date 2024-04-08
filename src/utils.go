// utils.go

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

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

// URLValidationRequest represents the request format for URL validation
type URLValidationRequest struct {
	URLs []string `json:"urls"`
}

// URLStatus represents the status of a single URL validation
type URLStatus struct {
	URL    string `json:"url"`
	Status string `json:"status"`
}

func parseTime(timeStr string) (time.Time, error) {
	// Try parsing with RFC1123 format
	t, err := time.Parse(time.RFC1123, timeStr)
	if err == nil {
		return t, nil
	}

	// If that fails, try parsing with RFC3339 format
	t, err = time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// If that fails, try parsing with "02 Jan 06 3:04 PM" format
	t, err = time.Parse("02 Jan 06 3:04 PM", timeStr)
	if err == nil {
		return t, nil
	}

	log.Printf("Failed to parse time: %v", err)

	// If all attempts fail, return the error
	return time.Time{}, fmt.Errorf("Failed to parse time: %v", err)
}
