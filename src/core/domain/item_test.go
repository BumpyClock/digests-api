package domain

import "testing"

func TestFeedItem_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		item     FeedItem
		expected bool
	}{
		{
			name: "valid item with all required fields",
			item: FeedItem{
				Title: "Test Article",
				Link:  "https://example.com/article",
			},
			expected: true,
		},
		{
			name: "invalid item with empty title",
			item: FeedItem{
				Title: "",
				Link:  "https://example.com/article",
			},
			expected: false,
		},
		{
			name: "invalid item with empty link",
			item: FeedItem{
				Title: "Test Article",
				Link:  "",
			},
			expected: false,
		},
		{
			name: "invalid item with both empty",
			item: FeedItem{
				Title: "",
				Link:  "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}