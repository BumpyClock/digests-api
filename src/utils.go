// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// validateURLsHandler handles the /validate endpoint
func validateURLsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		zap.L().Warn("[validateURLsHandler] Invalid method", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req URLValidationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		zap.L().Error("[validateURLsHandler] Error decoding request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	statuses := make([]URLStatus, len(req.URLs))
	for i, url := range req.URLs {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			resp, err := httpClient.Get(url)
			if resp != nil {
				defer resp.Body.Close()
			}
			status := "ok"
			if err != nil || resp.StatusCode != http.StatusOK {
				status = "error"
				if err == nil {
					status = http.StatusText(resp.StatusCode)
				}
				zap.L().Warn("[validateURLsHandler] URL validation failed", zap.String("url", url), zap.String("status", status), zap.Error(err))
			} else {
				zap.L().Debug("[validateURLsHandler] URL validation successful", zap.String("url", url), zap.String("status", status))
			}
			statuses[i] = URLStatus{URL: url, Status: status}
		}(i, url)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(statuses)
	if err != nil {
		zap.L().Error("[validateURLsHandler] Error encoding response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
