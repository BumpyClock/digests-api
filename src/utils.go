// Package main provides the main functionality for the web server.
package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

// validateURLsHandler handles the /validate endpoint
func validateURLsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.WithFields(logrus.Fields{
			"method": r.Method,
		}).Warn("[validateURLsHandler] Invalid method")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req URLValidationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[validateURLsHandler] Error decoding request body")
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
				log.WithFields(logrus.Fields{
					"url":    url,
					"status": status,
					"error":  err,
				}).Warn("[validateURLsHandler] URL validation failed")
			} else {
				log.WithFields(logrus.Fields{
					"url":    url,
					"status": status,
				}).Debug("[validateURLsHandler] URL validation successful")
			}
			statuses[i] = URLStatus{URL: url, Status: status}
		}(i, url)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(statuses)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[validateURLsHandler] Error encoding response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
