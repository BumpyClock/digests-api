// utils.go

package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// validateURLsHandler handles the /validate endpoint
func validateURLsHandler(c *gin.Context) {
	var req URLValidationRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	c.JSON(http.StatusOK, statuses)
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
