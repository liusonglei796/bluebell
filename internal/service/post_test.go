package service

import (
	"testing"
	"time"

	postreq "bluebell/internal/dto/request/post"
	postResp "bluebell/internal/dto/response/post"

	"github.com/stretchr/testify/assert"
)

func TestPostService_InMemoryRerank(t *testing.T) {
	now := time.Now()

	// 准备 3 篇帖子：
	// p1: 10 分钟前发布，100 票，普通贴
	// p2: 5 分钟前发布，10 票，置顶帖
	// p3: 1 分钟前发布，1 票，最新普通贴
	p1 := &postResp.DetailResponse{ID: "101", CreateTime: now.Add(-10 * time.Minute), VoteNum: 100, Score: 100}
	p2 := &postResp.DetailResponse{ID: "102", CreateTime: now.Add(-5 * time.Minute), VoteNum: 10, Score: 10, IsPinned: true}
	p3 := &postResp.DetailResponse{ID: "103", CreateTime: now.Add(-1 * time.Minute), VoteNum: 1, Score: 1}

	// rerankPosts 为纯函数，不依赖 Redis/本地缓存；输入为 ZSet 顺序装配后的列表。
	seed := []*postResp.DetailResponse{p1, p2, p3}

	// 1. 按时间排序 (order = "time") -> 置顶在前，然后最新发布在前 (p2 -> p3 -> p1)
	timeRes := rerankPosts(seed, postreq.OrderTime)
	assert.Len(t, timeRes, 3)
	assert.Equal(t, "102", timeRes[0].ID, "置顶帖必须排在最前")
	assert.Equal(t, "103", timeRes[1].ID, "时间最新的普通帖排在第二")
	assert.Equal(t, "101", timeRes[2].ID, "时间较早的普通帖排在最后")

	// 2. 按热度排序 (order = "score") -> 置顶在前，然后按 Gravity 算分最高在前 (p2 -> p1 -> p3)
	scoreRes := rerankPosts(seed, postreq.OrderScore)
	assert.Len(t, scoreRes, 3)
	assert.Equal(t, "102", scoreRes[0].ID, "置顶帖必须排在最前")
	assert.Equal(t, "101", scoreRes[1].ID, "高票帖子热度分更高，排在第二")
	assert.Equal(t, "103", scoreRes[2].ID, "低票帖子排在最后")
}
