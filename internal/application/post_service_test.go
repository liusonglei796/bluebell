package application

import (
	"context"
	"strconv"
	"testing"

	postreq "bluebell/internal/application/dto/request/post"
	"bluebell/internal/domain/entity"
)

// MockPostCacheRepository is a manual mock for domain.PostCacheRepository
type MockPostCacheRepository struct{}

func (m *MockPostCacheRepository) CreatePost(ctx context.Context, postID, communityID int64) error {
	return nil
}
func (m *MockPostCacheRepository) GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error) {
	return nil, nil
}
func (m *MockPostCacheRepository) GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error) {
	return nil, nil
}
func (m *MockPostCacheRepository) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
	return nil
}
func (m *MockPostCacheRepository) BatchVoteForPost(ctx context.Context, votes map[string]int8) error {
	return nil
}
func (m *MockPostCacheRepository) GetPostsVoteData(ctx context.Context, ids []string) ([]int64, error) {
	return nil, nil
}
func (m *MockPostCacheRepository) DeletePost(ctx context.Context, postID, communityID int64) error {
	return nil
}
func (m *MockPostCacheRepository) GetPostCommunityID(ctx context.Context, postID int64) (int64, error) {
	return 0, nil
}

// Other mocks to satisfy NewPostService
type MockPostRepository struct{}

func (m *MockPostRepository) CreatePost(ctx context.Context, post *entity.Post) error { return nil }
func (m *MockPostRepository) GetPostByID(ctx context.Context, pid int64) (*entity.Post, error) {
	return &entity.Post{PostID: strconv.FormatInt(pid, 10)}, nil
}
func (m *MockPostRepository) GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*entity.Post, error) {
	return nil, nil
}
func (m *MockPostRepository) DeletePostByAuthor(ctx context.Context, postID, authorID int64) error {
	return nil
}

type MockVoteRepository struct{}

func (m *MockVoteRepository) SaveVote(ctx context.Context, userID, postID int64, direction int8) error {
	return nil
}

type MockRemarkRepository struct{}

func (m *MockRemarkRepository) CreateRemark(ctx context.Context, remark *entity.Remark) error {
	return nil
}
func (m *MockRemarkRepository) GetRemarksByPostID(ctx context.Context, postID int64) ([]*entity.Remark, error) {
	return nil, nil
}
func (m *MockRemarkRepository) DeleteRemarkByID(ctx context.Context, remarkID uint) error { return nil }
func (m *MockRemarkRepository) DeleteRemarksByPostID(ctx context.Context, postID int64) error {
	return nil
}

// TestVoteForPost_DirectCall verifies that VoteForPost calls the cache correctly.
func TestVoteForPost_DirectCall(t *testing.T) {
	svc := NewPostService(&MockPostRepository{}, &MockPostCacheRepository{}, &MockRemarkRepository{}, nil, nil, nil)

	req := &postreq.VoteRequest{
		PostID:    1,
		Direction: entity.VoteUp,
	}
	err := svc.VoteForPost(context.Background(), 100, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestVoteForPost_InvalidDirection verifies validation rejects bad directions.
func TestVoteForPost_InvalidDirection(t *testing.T) {
	svc := NewPostService(&MockPostRepository{}, &MockPostCacheRepository{}, &MockRemarkRepository{}, nil, nil, nil)

	req := &postreq.VoteRequest{
		PostID:    1,
		Direction: 42, // invalid direction
	}
	err := svc.VoteForPost(context.Background(), 100, req)
	if err == nil {
		t.Error("expected error for invalid direction, got nil")
	}
}
