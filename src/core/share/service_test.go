package share

import (
	"context"
	"errors"
	"testing"
	"time"

	"digests-app-api/core/domain"
)

// mockShareStorage is a mock implementation of ShareStorage
type mockShareStorage struct {
	saveFunc func(ctx context.Context, share *domain.Share) error
	getFunc  func(ctx context.Context, id string) (*domain.Share, error)
}

func (m *mockShareStorage) Save(ctx context.Context, share *domain.Share) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, share)
	}
	return nil
}

func (m *mockShareStorage) Get(ctx context.Context, id string) (*domain.Share, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func TestNewShareService(t *testing.T) {
	storage := &mockShareStorage{}
	service := NewShareService(storage)
	
	if service == nil {
		t.Error("NewShareService returned nil")
	}
}

func TestCreateShare_EmptyURLs(t *testing.T) {
	storage := &mockShareStorage{}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.CreateShare(ctx, []string{})
	
	if err == nil {
		t.Error("CreateShare should return error for empty URLs")
	}
	if share != nil {
		t.Error("CreateShare should return nil share for empty URLs")
	}
}

func TestCreateShare_InvalidURL(t *testing.T) {
	storage := &mockShareStorage{}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.CreateShare(ctx, []string{"not a valid url"})
	
	if err == nil {
		t.Error("CreateShare should return error for invalid URL")
	}
	if share != nil {
		t.Error("CreateShare should return nil share for invalid URL")
	}
}

func TestCreateShare_CreatesWithUUID(t *testing.T) {
	storage := &mockShareStorage{
		saveFunc: func(ctx context.Context, share *domain.Share) error {
			return nil
		},
	}
	service := NewShareService(storage)
	
	ctx := context.Background()
	urls := []string{"https://example.com/feed1.xml", "https://example.com/feed2.xml"}
	share, err := service.CreateShare(ctx, urls)
	
	if err != nil {
		t.Errorf("CreateShare returned error: %v", err)
	}
	if share == nil {
		t.Fatal("CreateShare returned nil share")
	}
	if share.ID == "" {
		t.Error("CreateShare did not set share ID")
	}
	if len(share.ID) != 36 { // UUID length
		t.Errorf("Share ID length = %d, want 36 (UUID)", len(share.ID))
	}
}

func TestCreateShare_SetsCreatedAt(t *testing.T) {
	storage := &mockShareStorage{
		saveFunc: func(ctx context.Context, share *domain.Share) error {
			return nil
		},
	}
	service := NewShareService(storage)
	
	before := time.Now()
	ctx := context.Background()
	share, err := service.CreateShare(ctx, []string{"https://example.com/feed.xml"})
	after := time.Now()
	
	if err != nil {
		t.Errorf("CreateShare returned error: %v", err)
	}
	if share == nil {
		t.Fatal("CreateShare returned nil share")
	}
	if share.CreatedAt.Before(before) || share.CreatedAt.After(after) {
		t.Error("CreateShare did not set CreatedAt to current time")
	}
}

func TestCreateShare_SavesToStorage(t *testing.T) {
	saveCalled := false
	var savedShare *domain.Share
	
	storage := &mockShareStorage{
		saveFunc: func(ctx context.Context, share *domain.Share) error {
			saveCalled = true
			savedShare = share
			return nil
		},
	}
	service := NewShareService(storage)
	
	ctx := context.Background()
	urls := []string{"https://example.com/feed.xml"}
	share, err := service.CreateShare(ctx, urls)
	
	if err != nil {
		t.Errorf("CreateShare returned error: %v", err)
	}
	if !saveCalled {
		t.Error("CreateShare did not save to storage")
	}
	if savedShare != share {
		t.Error("CreateShare saved different share instance")
	}
}

func TestCreateShare_ReturnsStorageError(t *testing.T) {
	storage := &mockShareStorage{
		saveFunc: func(ctx context.Context, share *domain.Share) error {
			return errors.New("storage error")
		},
	}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.CreateShare(ctx, []string{"https://example.com/feed.xml"})
	
	if err == nil {
		t.Error("CreateShare should return storage error")
	}
	if share != nil {
		t.Error("CreateShare should return nil share on storage error")
	}
}

func TestGetShare_EmptyID(t *testing.T) {
	storage := &mockShareStorage{}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.GetShare(ctx, "")
	
	if err == nil {
		t.Error("GetShare should return error for empty ID")
	}
	if share != nil {
		t.Error("GetShare should return nil share for empty ID")
	}
}

func TestGetShare_InvalidUUID(t *testing.T) {
	storage := &mockShareStorage{}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.GetShare(ctx, "not-a-uuid")
	
	if err == nil {
		t.Error("GetShare should return error for invalid UUID")
	}
	if share != nil {
		t.Error("GetShare should return nil share for invalid UUID")
	}
}

func TestGetShare_ReturnsFromStorage(t *testing.T) {
	expectedShare := &domain.Share{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		URLs:      []string{"https://example.com/feed.xml"},
		CreatedAt: time.Now(),
	}
	
	storage := &mockShareStorage{
		getFunc: func(ctx context.Context, id string) (*domain.Share, error) {
			if id == expectedShare.ID {
				return expectedShare, nil
			}
			return nil, nil
		},
	}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.GetShare(ctx, expectedShare.ID)
	
	if err != nil {
		t.Errorf("GetShare returned error: %v", err)
	}
	if share != expectedShare {
		t.Error("GetShare did not return expected share")
	}
}

func TestGetShare_NonExistent(t *testing.T) {
	storage := &mockShareStorage{
		getFunc: func(ctx context.Context, id string) (*domain.Share, error) {
			return nil, nil
		},
	}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.GetShare(ctx, "550e8400-e29b-41d4-a716-446655440000")
	
	if err != nil {
		t.Errorf("GetShare returned error: %v", err)
	}
	if share != nil {
		t.Error("GetShare should return nil for non-existent share")
	}
}

func TestGetShare_ExpiredShare(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	expiredShare := &domain.Share{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		URLs:      []string{"https://example.com/feed.xml"},
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: &past,
	}
	
	storage := &mockShareStorage{
		getFunc: func(ctx context.Context, id string) (*domain.Share, error) {
			return expiredShare, nil
		},
	}
	service := NewShareService(storage)
	
	ctx := context.Background()
	share, err := service.GetShare(ctx, expiredShare.ID)
	
	if err == nil {
		t.Error("GetShare should return error for expired share")
	}
	if share != nil {
		t.Error("GetShare should return nil for expired share")
	}
}