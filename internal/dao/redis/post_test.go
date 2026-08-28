package redis

import (
	"context"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	postResp "bluebell/internal/dto/response/post"
	"bluebell/internal/model"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateGravityScore(t *testing.T) {
	now := time.Now()

	// 1. votes <= 0 时 score 必须为 0
	assert.Equal(t, float64(0), CalculateGravityScore(0, 0, now))
	assert.Equal(t, float64(0), CalculateGravityScore(1, 2, now))
	assert.Equal(t, float64(0), CalculateGravityScore(5, 5, now))

	// 2. 新帖 1 赞成票：votes=1, (1-1) / (0+2)^1.8 = 0
	assert.Equal(t, float64(0), CalculateGravityScore(1, 0, now))

	// 3. 新帖 10 赞成票：votes=10, hours=0, score = 9 / (2^1.8)
	expectedScore := 9.0 / math.Pow(2.0, 1.8)
	actualScore := CalculateGravityScore(10, 0, now)
	assert.InDelta(t, expectedScore, actualScore, 0.0001)

	// 4. 发帖 2 小时后，10 赞成票：hours=2, score = 9 / (4^1.8)
	twoHoursAgo := now.Add(-2 * time.Hour)
	expectedDecayedScore := 9.0 / math.Pow(4.0, 1.8)
	actualDecayedScore := CalculateGravityScore(10, 0, twoHoursAgo)
	assert.InDelta(t, expectedDecayedScore, actualDecayedScore, 0.0001)
	assert.True(t, actualDecayedScore < actualScore, "衰减后的分数应当低于即时分数")
}

func setupTestRedis(t *testing.T) (*goredis.Client, *PostCache, func()) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15, // 使用测试专属 DB 15
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("跳过 Redis 集成测试：无法连接到本地 Redis (127.0.0.1:6379)")
	}

	_ = rdb.FlushDB(context.Background()).Err()

	cache := NewPostCache(rdb)

	cleanup := func() {
		_ = rdb.FlushDB(context.Background()).Err()
		_ = rdb.Close()
	}

	return rdb, cache, cleanup
}

func TestVoteForPost_LifecycleAndIdempotency(t *testing.T) {
	rdb, cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	postID := int64(1001)
	communityID := int64(1)
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)
	userIDStr := "999"

	// 0. 未创建帖子直接投票 -> 应返回 ErrNotFound
	err := cache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, 1)
	assert.ErrorIs(t, err, model.ErrNotFound)

	// 1. 初始化帖子
	err = cache.CreatePost(ctx, postID, communityID)
	require.NoError(t, err)

	// 2. 正常投赞成票 (value = 1)
	err = cache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, 1)
	assert.NoError(t, err)

	// 验证 Redis 数据结构
	voteRecord := rdb.ZScore(ctx, redisKey(keyPostVotedZSetPrefix+postIDStr), userIDStr).Val()
	assert.Equal(t, float64(1), voteRecord)

	meta := rdb.HGetAll(ctx, redisKey(keyPostMetaPrefix+postIDStr)).Val()
	assert.Equal(t, "1", meta["vote_up"])
	assert.Equal(t, "0", meta["vote_down"])

	// 3. 重复投赞成票 -> 拦截并返回 ErrVoteRepeated
	err = cache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, 1)
	assert.ErrorIs(t, err, model.ErrVoteRepeated)

	// 4. 改投反对票 (value = -1)
	err = cache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, -1)
	assert.NoError(t, err)

	voteRecord = rdb.ZScore(ctx, redisKey(keyPostVotedZSetPrefix+postIDStr), userIDStr).Val()
	assert.Equal(t, float64(-1), voteRecord)

	meta = rdb.HGetAll(ctx, redisKey(keyPostMetaPrefix+postIDStr)).Val()
	assert.Equal(t, "0", meta["vote_up"])
	assert.Equal(t, "1", meta["vote_down"])

	// 5. 取消投票 (value = 0)
	err = cache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, 0)
	assert.NoError(t, err)

	// 投票记录应当被清除 (ZScore 返回 0 且 err == redis.Nil)
	_, err = rdb.ZScore(ctx, redisKey(keyPostVotedZSetPrefix+postIDStr), userIDStr).Result()
	assert.ErrorIs(t, err, goredis.Nil)

	meta = rdb.HGetAll(ctx, redisKey(keyPostMetaPrefix+postIDStr)).Val()
	assert.Equal(t, "0", meta["vote_up"])
	assert.Equal(t, "0", meta["vote_down"])
}

func TestVoteForPost_ExpiredPost(t *testing.T) {
	rdb, cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	postID := int64(1002)
	communityID := int64(1)
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)

	// 创建一个 200 周前的帖子
	err := cache.CreatePost(ctx, postID, communityID)
	require.NoError(t, err)

	oldTime := time.Now().Unix() - (200 * 7 * 86400)
	rdb.HSet(ctx, redisKey(keyPostMetaPrefix+postIDStr), "create_time", strconv.FormatInt(oldTime, 10))

	// 投票应提示过期
	err = cache.VoteForPost(ctx, "999", postIDStr, communityIDStr, 1)
	assert.ErrorIs(t, err, model.ErrVoteTimeExpire)
}

func TestVoteForPost_HighConcurrency(t *testing.T) {
	rdb, cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	postID := int64(2001)
	communityID := int64(10)
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)

	// 初始化帖子
	err := cache.CreatePost(ctx, postID, communityID)
	require.NoError(t, err)

	// 模拟 50 个用户同时发起投票
	concurrency := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 1; i <= concurrency; i++ {
		userID := strconv.Itoa(i)
		go func(u string) {
			defer wg.Done()
			voteErr := cache.VoteForPost(ctx, u, postIDStr, communityIDStr, 1)
			assert.NoError(t, voteErr)
		}(userID)
	}

	wg.Wait()

	// 验证 1：投票记录 ZSet 中的总人数准确为 50
	votedCount, err := rdb.ZCard(ctx, redisKey(keyPostVotedZSetPrefix+postIDStr)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(concurrency), votedCount)

	// 验证 2：元数据 Hash 中的 vote_up 精确等于 50
	meta := rdb.HGetAll(ctx, redisKey(keyPostMetaPrefix+postIDStr)).Val()
	assert.Equal(t, strconv.Itoa(concurrency), meta["vote_up"])
	assert.Equal(t, "0", meta["vote_down"])

	// 验证 3：全局排行榜与社区排行榜中的 Gravity 分数必须完全等于 50 票对应的最新分数（无脏写覆盖）
	// 在发帖当秒内投票，hours = 0，分母为 2^gravity
	expectedScore := float64(concurrency-1) / math.Pow(2.0, 1.8)

	globalScore, err := rdb.ZScore(ctx, redisKey(keyPostScoreZSet), postIDStr).Result()
	require.NoError(t, err)
	assert.InDelta(t, expectedScore, globalScore, 0.0001, "全局排行榜分数必须等于并发最终的正确热度分数")

	communityScore, err := rdb.ZScore(ctx, redisKey(keyCommunityPostScorePrefix+communityIDStr), postIDStr).Result()
	require.NoError(t, err)
	assert.InDelta(t, expectedScore, communityScore, 0.0001, "社区排行榜分数必须等于并发最终的正确热度分数")
}

func TestPostDetailCache_MGetAndSet(t *testing.T) {
	_, cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	p1 := &postResp.DetailResponse{
		ID:            "9001",
		Title:         "Post 1",
		Content:       "Content 1",
		AuthorName:    "Alice",
		CommunityName: "Go",
		Tags:          []string{"Go", "Concurrency"},
	}
	p2 := &postResp.DetailResponse{
		ID:            "9002",
		Title:         "Post 2",
		Content:       "Content 2",
		AuthorName:    "Bob",
		CommunityName: "Rust",
		Tags:          []string{"Rust"},
	}

	// 1. 批量写入缓存
	err := cache.SetPostDetails(ctx, []*postResp.DetailResponse{p1, p2}, 1*time.Hour)
	require.NoError(t, err)

	// 2. 单键查询 9001, 9002, 9003 (9003 未命中)
	ids := []string{"9001", "9002", "9003"}
	var got []*postResp.DetailResponse
	var missedIDs []string
	for _, id := range ids {
		item, err := cache.GetPostDetail(ctx, id)
		require.NoError(t, err)
		if item == nil {
			missedIDs = append(missedIDs, id)
			continue
		}
		got = append(got, item)
	}
	assert.Len(t, got, 2)
	assert.Equal(t, []string{"9003"}, missedIDs)
	assert.Equal(t, "Post 1", got[0].Title)
	assert.Equal(t, "Alice", got[0].AuthorName)
	assert.Equal(t, []string{"Go", "Concurrency"}, got[0].Tags)
	assert.Equal(t, "Post 2", got[1].Title)

	// 3. 删除 9001 缓存
	err = cache.DeletePostDetailCache(ctx, "9001")
	require.NoError(t, err)

	ids2 := []string{"9001", "9002"}
	var got2 []*postResp.DetailResponse
	var missedIDs2 []string
	for _, id := range ids2 {
		item, err := cache.GetPostDetail(ctx, id)
		require.NoError(t, err)
		if item == nil {
			missedIDs2 = append(missedIDs2, id)
			continue
		}
		got2 = append(got2, item)
	}
	assert.Len(t, got2, 1)
	assert.Equal(t, []string{"9001"}, missedIDs2)
	assert.Equal(t, "Post 2", got2[0].Title)
}
