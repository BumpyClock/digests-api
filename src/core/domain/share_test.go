package domain

import (
	"testing"
	"time"
)

func TestNewShare(t *testing.T) {
	tests := []struct {
		name    string
		urls    []string
		wantErr bool
	}{
		{
			name:    "valid share with single URL",
			urls:    []string{"https://example.com"},
			wantErr: false,
		},
		{
			name:    "valid share with multiple URLs",
			urls:    []string{"https://example.com", "https://test.com"},
			wantErr: false,
		},
		{
			name:    "invalid share with empty URLs",
			urls:    []string{},
			wantErr: true,
		},
		{
			name:    "invalid share with malformed URL",
			urls:    []string{"not a valid url"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share, err := NewShare(tt.urls)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewShare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if share == nil {
					t.Error("NewShare() returned nil share")
				} else if share.ID == "" {
					t.Error("NewShare() did not generate ID")
				} else if len(share.URLs) != len(tt.urls) {
					t.Errorf("NewShare() URLs count = %v, want %v", len(share.URLs), len(tt.urls))
				}
			}
		})
	}
}

func TestShare_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "no expiration (nil)",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "expired (past time)",
			expiresAt: &past,
			want:      true,
		},
		{
			name:      "not expired (future time)",
			expiresAt: &future,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			share := &Share{
				ID:        "test-id",
				URLs:      []string{"https://example.com"},
				CreatedAt: now,
				ExpiresAt: tt.expiresAt,
			}
			if got := share.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}