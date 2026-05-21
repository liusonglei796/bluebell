package postsvc

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bluebell/internal/domain/entity"
	postreq "bluebell/internal/application/dto/request/post"
	"bluebell/internal/infrastructure/mq"
)

// MockPostCacheRepository is a manual mock for domain.PostCacheRepository
type MockPostCacheRepository struct {
	BatchVoteForPostFunc func(ctx context.Context, votes map[string]int8) error
}

func (m *MockPostCacheRepository) CreatePost(ctx context.Context, postID, communityID int64) error { return nil }
func (m *MockPostCacheRepository) GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error) { return nil, nil }
func (m *MockPostCacheRepository) GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error) { return nil, nil }
func (m *MockPostCacheRepository) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error { return nil }
func (m *MockPostCacheRepository) BatchVoteForPost(ctx context.Context, votes map[string]int8) error {
	if m.BatchVoteForPostFunc != nil {
		return m.BatchVoteForPostFunc(ctx, votes)
	}
	return nil
}
func (m *MockPostCacheRepository) GetPostsVoteData(ctx context.Context, ids []string) ([]int64, error) { return nil, nil }
func (m *MockPostCacheRepository) DeletePost(ctx context.Context, postID, communityID int64) error { return nil }
func (m *MockPostCacheRepository) GetPostCommunityID(ctx context.Context, postID int64) (int64, error) { return 0, nil }

// Other mocks to satisfy NewPostService
type MockPostRepository struct{}
func (m *MockPostRepository) CreatePost(ctx context.Context, post *entity.Post) error { return nil }
func (m *MockPostRepository) GetPostByID(ctx context.Context, pid int64) (*entity.Post, error) { return &entity.Post{PostID: strconv.FormatInt(pid, 10)}, nil }
func (m *MockPostRepository) GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*entity.Post, error) { return nil, nil }
func (m *MockPostRepository) DeletePostByAuthor(ctx context.Context, postID, authorID int64) error { return nil }

type MockVoteRepository struct{}
func (m *MockVoteRepository) SaveVote(ctx context.Context, userID, postID int64, direction int8) error { return nil }

type MockRemarkRepository struct{}
func (m *MockRemarkRepository) CreateRemark(ctx context.Context, remark *entity.Remark) error { return nil }
func (m *MockRemarkRepository) GetRemarksByPostID(ctx context.Context, postID int64) ([]*entity.Remark, error) { return nil, nil }
func (m *MockRemarkRepository) DeleteRemarkByID(ctx context.Context, remarkID uint) error { return nil }
func (m *MockRemarkRepository) DeleteRemarksByPostID(ctx context.Context, postID int64) error { return nil }

// TestVoteForPost_Consistency verifies that high-concurrency voting eventually 
// flushes the correct number of unique votes to the cache.
func TestVoteForPost_Consistency(t *testing.T) {
	var mu sync.Mutex
	finalVotes := make(map[string]int8)
	var flushCount int32

	mockCache := &MockPostCacheRepository{
		BatchVoteForPostFunc: func(ctx context.Context, votes map[string]int8) error {
			mu.Lock()
			defer mu.Unlock()
			for k, v := range votes {
				finalVotes[k] = v
			}
			atomic.AddInt32(&flushCount, 1)
			return nil
		},
	}

	// Buffer with short interval for testing
	interval := 10 * time.Millisecond
	buffer := mq.NewVoteBuffer(interval, mockCache.BatchVoteForPost)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go buffer.Start(ctx)

	svc := NewPostService(&MockPostRepository{}, mockCache, &MockVoteRepository{}, &MockRemarkRepository{}, nil, nil, nil, buffer)

	const (
		numUsers = 100
		numPosts = 10
	)

	var wg sync.WaitGroup
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			for j := 0; j < numPosts; j++ {
				req := &postreq.VoteRequest{
					PostID:    int64(j + 1),
					Direction: entity.VoteUp,
				}
				_ = svc.VoteForPost(context.Background(), int64(userID+1), req)
			}
		}(i)
	}
	wg.Wait()

	// Wait for a few flushes
	time.Sleep(interval * 5)

	mu.Lock()
	expectedUniqueVotes := numUsers * numPosts
	if len(finalVotes) != expectedUniqueVotes {
		t.Errorf("expected %d unique votes in cache, got %d", expectedUniqueVotes, len(finalVotes))
	}
	mu.Unlock()
	
	if atomic.LoadInt32(&flushCount) == 0 {
		t.Error("expected at least one flush to occur")
	}
}

// BenchmarkVoteForPost benchmarks the VoteForPost call with the buffer.
// This measures how quickly we can "ingest" votes into the local buffer.
func BenchmarkVoteForPost(b *testing.B) {
	mockCache := &MockPostCacheRepository{}
	// Large interval to avoid flushing during the benchmark ingestion phase if possible,
	// though flushing happens in background and shouldn't block AddVote much due to mutex.
	buffer := mq.NewVoteBuffer(time.Hour, mockCache.BatchVoteForPost)
	
	svc := NewPostService(&MockPostRepository{}, mockCache, &MockVoteRepository{}, &MockRemarkRepository{}, nil, nil, nil, buffer)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			val := atomic.AddInt64(&i, 1)
			req := &postreq.VoteRequest{
				PostID:    1,
				Direction: entity.VoteUp,
			}
			// Use unique userIDs to avoid map key collisions affecting performance too much in this specific test,
			// though the mutex is the main bottleneck.
			_ = svc.VoteForPost(context.Background(), val, req)
		}
	})
}

// BenchmarkVoteForPost_NoBuffer benchmarks the VoteForPost call without the buffer (direct cache call simulation).
func BenchmarkVoteForPost_NoBuffer(b *testing.B) {
	mockCache := &MockPostCacheRepository{}
	
	// Simulation of direct call (bypassing the buffer)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			val := atomic.AddInt64(&i, 1)
			postIDStr := "1"
			userIDStr := strconv.FormatInt(val, 10)
			_ = mockCache.VoteForPost(context.Background(), userIDStr, postIDStr, "1", 1)
		}
	})
}
