package feed

import (
	"testing"

	"digests-app-api/core/domain"
)

func TestPaginateItems_AllItemsWhenPerPageLarge(t *testing.T) {
	items := []domain.FeedItem{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}

	result := PaginateItems(items, 1, 10)

	if len(result) != 3 {
		t.Errorf("PaginateItems returned %d items, want 3", len(result))
	}
	if result[0].ID != "1" {
		t.Errorf("First item ID = %v, want 1", result[0].ID)
	}
}

func TestPaginateItems_FirstPage(t *testing.T) {
	// Create 20 items
	items := make([]domain.FeedItem, 20)
	for i := 0; i < 20; i++ {
		items[i] = domain.FeedItem{
			ID:    string(rune(i + 1)),
			Title: string(rune(i + 1)),
		}
	}

	result := PaginateItems(items, 1, 10)

	if len(result) != 10 {
		t.Errorf("PaginateItems returned %d items, want 10", len(result))
	}
	if result[0].ID != string(rune(1)) {
		t.Errorf("First item ID = %v, want 1", result[0].ID)
	}
	if result[9].ID != string(rune(10)) {
		t.Errorf("Last item ID = %v, want 10", result[9].ID)
	}
}

func TestPaginateItems_SecondPage(t *testing.T) {
	// Create 20 items
	items := make([]domain.FeedItem, 20)
	for i := 0; i < 20; i++ {
		items[i] = domain.FeedItem{
			ID:    string(rune(i + 1)),
			Title: string(rune(i + 1)),
		}
	}

	result := PaginateItems(items, 2, 10)

	if len(result) != 10 {
		t.Errorf("PaginateItems returned %d items, want 10", len(result))
	}
	if result[0].ID != string(rune(11)) {
		t.Errorf("First item ID = %v, want 11", result[0].ID)
	}
	if result[9].ID != string(rune(20)) {
		t.Errorf("Last item ID = %v, want 20", result[9].ID)
	}
}

func TestPaginateItems_PageBeyondItems(t *testing.T) {
	items := []domain.FeedItem{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}

	result := PaginateItems(items, 5, 10)

	if len(result) != 0 {
		t.Errorf("PaginateItems returned %d items, want 0 for page beyond items", len(result))
	}
}

func TestPaginateItems_InvalidPage(t *testing.T) {
	items := []domain.FeedItem{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}

	// Test page < 1
	result := PaginateItems(items, 0, 2)

	if len(result) != 2 {
		t.Errorf("PaginateItems returned %d items, want 2", len(result))
	}
	if result[0].ID != "1" {
		t.Errorf("First item ID = %v, want 1 (should use page=1)", result[0].ID)
	}

	// Test negative page
	result = PaginateItems(items, -1, 2)

	if len(result) != 2 {
		t.Errorf("PaginateItems returned %d items, want 2", len(result))
	}
	if result[0].ID != "1" {
		t.Errorf("First item ID = %v, want 1 (should use page=1)", result[0].ID)
	}
}

func TestPaginateItems_InvalidPerPage(t *testing.T) {
	items := make([]domain.FeedItem, 15)
	for i := 0; i < 15; i++ {
		items[i] = domain.FeedItem{
			ID:    string(rune(i + 1)),
			Title: string(rune(i + 1)),
		}
	}

	// Test perPage < 1
	result := PaginateItems(items, 1, 0)

	if len(result) != 10 {
		t.Errorf("PaginateItems returned %d items, want 10 (default)", len(result))
	}

	// Test negative perPage
	result = PaginateItems(items, 1, -5)

	if len(result) != 10 {
		t.Errorf("PaginateItems returned %d items, want 10 (default)", len(result))
	}
}

func TestPaginateItems_PartialLastPage(t *testing.T) {
	// Create 15 items
	items := make([]domain.FeedItem, 15)
	for i := 0; i < 15; i++ {
		items[i] = domain.FeedItem{
			ID:    string(rune(i + 1)),
			Title: string(rune(i + 1)),
		}
	}

	// Get second page with perPage=10
	result := PaginateItems(items, 2, 10)

	if len(result) != 5 {
		t.Errorf("PaginateItems returned %d items, want 5 (partial last page)", len(result))
	}
	if result[0].ID != string(rune(11)) {
		t.Errorf("First item ID = %v, want 11", result[0].ID)
	}
	if result[4].ID != string(rune(15)) {
		t.Errorf("Last item ID = %v, want 15", result[4].ID)
	}
}