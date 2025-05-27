package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAPI(t *testing.T) {
	api, router := NewAPI()
	
	if api == nil {
		t.Error("NewAPI returned nil API")
	}
	if router == nil {
		t.Error("NewAPI returned nil router")
	}
}

func TestNewAPI_HasCorrectTitle(t *testing.T) {
	api, _ := NewAPI()
	
	info := api.OpenAPI().Info
	expectedTitle := "Digests API"
	
	if info.Title != expectedTitle {
		t.Errorf("API title = %s, want %s", info.Title, expectedTitle)
	}
}

func TestNewAPI_HasCorrectVersion(t *testing.T) {
	api, _ := NewAPI()
	
	info := api.OpenAPI().Info
	expectedVersion := "1.0.0"
	
	if info.Version != expectedVersion {
		t.Errorf("API version = %s, want %s", info.Version, expectedVersion)
	}
}

func TestAPI_UsesChiRouter(t *testing.T) {
	_, router := NewAPI()
	
	if router == nil {
		t.Fatal("Router is nil")
	}
	
	// The router should implement http.Handler
	handler := http.Handler(router)
	if handler == nil {
		t.Error("Router does not implement http.Handler")
	}
}

func TestAPI_OpenAPIEndpoint(t *testing.T) {
	_, router := NewAPI()
	
	// Create a test request
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	resp := w.Result()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("OpenAPI endpoint status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/vnd.oai.openapi+json" {
		t.Errorf("OpenAPI content-type = %s, want application/vnd.oai.openapi+json", contentType)
	}
}

func TestAPI_DocsEndpoint(t *testing.T) {
	_, router := NewAPI()
	
	// Create a test request
	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	resp := w.Result()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Docs endpoint status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Docs content-type = %s, want text/html", contentType)
	}
}