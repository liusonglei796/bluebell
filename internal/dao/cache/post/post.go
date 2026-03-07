package postcache

import (
	"context"
	"strconv"
	"time"

	"bluebell/internal/domain/cachedomain"
	"bluebell/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

// Redis Keys 相关常量
const (
	keyPrefix              = "bluebell:"
	keyPostTimeZSet        = "post:time"   // bluebell:post:time - 所有帖子按时间排序
	keyPostScoreZSet       = "post:score"  // bluebell:post:score - 所有帖子按分数排序
	keyPostVotedZSetPrefix = "post:voted:" // bluebell:post:voted:{postID} - 帖子的投票记录

	// 社区维度的帖子排序 Key
	keyCommunityPostTimePrefix  = "community:post:time:"
	keyCommunityPostScorePrefix = "community:post:score:"

	// 投票相关常量
	oneWeekInSeconds = 7 * 24 * 3600 // 一周的秒数，超过一周的帖子不允许投票
	scorePerVote     = 432           // 每一票的分数权重: 86400秒/天 ÷ 200票 = 432分/票
)

// cacheStruct 帖子缓存仓储实现
// 实现 PostRepository（含帖子排序、分页与投票）
type cacheStruct struct {
	rdb *redis.Client
}

// NewCache 创建 cacheStruct 实例
func NewCache(rdb *redis.Client) cachedomain.PostRepository {
	return &cacheStruct{rdb: rdb}
}

func redisKey(key string) string {
	return keyPrefix + key
}

func timeNow() int64 {
	return time.Now().Unix()
}

// ========= PostRepository 实现 =========

// CreatePost 创建帖子时初始化 Redis 数据
func (c *cacheStruct) CreatePost(ctx context.Context, postID, communityID int64) error {
	pipeline := c.rdb.TxPipeline()
	timestamp := float64(timeNow())
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)

	// 全局维度
	pipeline.ZAdd(ctx, redisKey(keyPostTimeZSet), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})
	pipeline.ZAdd(ctx, redisKey(keyPostScoreZSet), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})

	// 社区维度
	pipeline.ZAdd(ctx, redisKey(keyCommunityPostTimePrefix+communityIDStr), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})
	pipeline.ZAdd(ctx, redisKey(keyCommunityPostScorePrefix+communityIDStr), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "create post pipeline exec failed (post_id: %d)", postID)
	}
	return nil
}

// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
func (c *cacheStruct) GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error) {
	key := redisKey(keyPostTimeZSet)
	if orderKey == "score" {
		key = redisKey(keyPostScoreZSet)
	}
	start := (page - 1) * size
	end := start + size - 1

	ids, err := c.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  end,
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, errorx.Wrapf(err, errorx.CodeCacheError, "get post ids failed (order: %s)", orderKey)
	}
	return ids, nil
}

// GetCommunityPostIDsInOrder 按照指定顺序获取指定社区的帖子ID列表
func (c *cacheStruct) GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error) {
	kp := keyCommunityPostTimePrefix
	if orderKey == "score" {
		kp = keyCommunityPostScorePrefix
	}

	key := redisKey(kp + strconv.FormatInt(communityID, 10))

	start := (page - 1) * size
	end := start + size - 1

	ids, err := c.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  end,
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, errorx.Wrapf(err, errorx.CodeCacheError, "get community post ids failed (community_id: %d, order: %s)", communityID, orderKey)
	}
	return ids, nil
}

// VoteForPost 为帖子投票
func (c *cacheStruct) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
	// 1. 判断投票时间限制
	postTime := c.rdb.ZScore(ctx, redisKey(keyPostTimeZSet), postID).Val()
	if float64(timeNow())-postTime > oneWeekInSeconds {
		return errorx.ErrVoteTimeExpire
	}
	// 2. 查询用户之前对该帖子的投票记录
	oldValue := c.rdb.ZScore(ctx, redisKey(keyPostVotedZSetPrefix+postID), userID).Val()
	// 3. 如果新旧投票值相同，直接返回
	if value == oldValue {
		return errorx.ErrVoteRepeated
	}
	// 4. 计算分数变化
	scoreDiff := (value - oldValue) * scorePerVote
	// 5. 使用 Redis Pipeline 保证原子性
	pipeline := c.rdb.TxPipeline()
	// 更新两类帖子的 ZSET
	pipeline.ZIncrBy(ctx, redisKey(keyPostScoreZSet), scoreDiff, postID)
	pipeline.ZIncrBy(ctx, redisKey(keyCommunityPostScorePrefix+communityID), scoreDiff, postID)
	// 更新个人用户对该帖子的投票记录
	if value == 0 {
		pipeline.ZRem(ctx, redisKey(keyPostVotedZSetPrefix+postID), userID)
	} else {
		pipeline.ZAdd(ctx, redisKey(keyPostVotedZSetPrefix+postID), redis.Z{
			Score:  value,
			Member: userID,
		})
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "vote pipeline exec failed (post_id: %s, user_id: %s)", postID, userID)
	}
	return nil
}

// GetPostsVoteData 批量获取多个帖子的投票数（赞成票数）
func (c *cacheStruct) GetPostsVoteData(ctx context.Context, ids []string) (data []int64, err error) {
	pipeline := c.rdb.Pipeline()

	for _, id := range ids {
		key := redisKey(keyPostVotedZSetPrefix + id)
		pipeline.ZCount(ctx, key, "1", "1")
	}

	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, errorx.Wrapf(err, errorx.CodeCacheError, "get posts vote data pipeline exec failed (count: %d)", len(ids))
	}

	data = make([]int64, 0, len(cmders))
	for _, cmder := range cmders {
		v := cmder.(*redis.IntCmd).Val()
		data = append(data, v)
	}
	return
}
