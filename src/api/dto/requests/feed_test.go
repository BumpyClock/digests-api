package requests

import (
	"testing"
)

func TestParseFeedsRequest_Structure(t *testing.T) {
	// Test that the struct has the expected fields with correct tags
	req := ParseFeedsRequest{
		URLs:         []string{"https://example.com/feed.xml"},
		Page:         2,
		ItemsPerPage: 25,
	}
	
	if len(req.URLs) != 1 {
		t.Errorf("URLs length = %d, want 1", len(req.URLs))
	}
	
	if req.Page != 2 {
		t.Errorf("Page = %d, want 2", req.Page)
	}
	
	if req.ItemsPerPage != 25 {
		t.Errorf("ItemsPerPage = %d, want 25", req.ItemsPerPage)
	}
}

func TestParseFeedsRequest_Defaults(t *testing.T) {
	req := ParseFeedsRequest{
		URLs: []string{"https://example.com/feed.xml"},
	}
	
	// Apply defaults
	req.ApplyDefaults()
	
	if req.Page != 1 {
		t.Errorf("Default Page = %d, want 1", req.Page)
	}
	
	if req.ItemsPerPage != 50 {
		t.Errorf("Default ItemsPerPage = %d, want 50", req.ItemsPerPage)
	}
}

func TestParseFeedsRequest_DoesNotOverrideSetValues(t *testing.T) {
	req := ParseFeedsRequest{
		URLs:         []string{"https://example.com/feed.xml"},
		Page:         3,
		ItemsPerPage: 25,
	}
	
	// Apply defaults
	req.ApplyDefaults()
	
	if req.Page != 3 {
		t.Errorf("Page = %d, want 3 (should not override)", req.Page)
	}
	
	if req.ItemsPerPage != 25 {
		t.Errorf("ItemsPerPage = %d, want 25 (should not override)", req.ItemsPerPage)
	}
}