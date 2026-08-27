package service

import (
	"context"
	"testing"
	"time"

	"bluebell/internal/dao/redis"
	postreq "bluebell/internal/dto/request/post"
	postResp "bluebell/internal/dto/response/post"

	"github.com/stretchr/testify/assert"
)

func TestPostService_InMemoryRerank(t *testing.T) {
	localCache := redis.NewLocalPostCache()
	s := &PostService{
		localCache: localCache,
	}

	now := time.Now()

	// 准备 3 篇帖子：
	// p1: 10 分钟前发布，100 票，普通贴
	p1 := &postResp.DetailResponse{
		ID:         "101",
		Title:      "High Vote Post",
		CreateTime: now.Add(-10 * time.Minute),
		VoteNum:    100,
		Score:      100,
		IsPinned:   false,
	}
	// p2: 5 分钟前发布，10 票，置顶帖
	p2 := &postResp.DetailResponse{
		ID:         "102",
		Title:      "Pinned Post",
		CreateTime: now.Add(-5 * time.Minute),
		VoteNum:    10,
		Score:      10,
		IsPinned:   true,
	}
	// p3: 1 分钟前发布，1 票，最新普通贴
	p3 := &postResp.DetailResponse{
		ID:         "103",
		Title:      "Fresh Post",
		CreateTime: now.Add(-1 * time.Minute),
		VoteNum:    1,
		Score:      1,
		IsPinned:   false,
	}

	localCache.Set("101", p1, 1*time.Minute)
	localCache.Set("102", p2, 1*time.Minute)
	localCache.Set("103", p3, 1*time.Minute)

	ctx := context.Background()

	// 1. 测试按时间排序 (order = "time") -> 置顶在前，然后是最新发布在前 (p2 -> p3 -> p1)
	timeRes := s.HydrateAndRerankPosts(ctx, []string{"101", "102", "103"}, 0, postreq.OrderTime)
	assert.Len(t, timeRes, 3)
	assert.Equal(t, "102", timeRes[0].ID, "置顶帖必须排在最前")
	assert.Equal(t, "103", timeRes[1].ID, "时间最新的普通帖排在第二")
	assert.Equal(t, "101", timeRes[2].ID, "时间较早的普通帖排在最后")

	// 2. 测试按热度排序 (order = "score") -> 置顶在前，然后按 Gravity 算分最高在前 (p2 -> p1 -> p3)
	scoreRes := s.HydrateAndRerankPosts(ctx, []string{"101", "102", "103"}, 0, postreq.OrderScore)
	assert.Len(t, scoreRes, 3)
	assert.Equal(t, "102", scoreRes[0].ID, "置顶帖必须排在最前")
	assert.Equal(t, "101", scoreRes[1].ID, "高票帖子热度分更高，排在第二")
	assert.Equal(t, "103", scoreRes[2].ID, "低票帖子排在最后")
}
