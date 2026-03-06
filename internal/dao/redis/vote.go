package redis

import (
	"bluebell/internal/domain/repointerface"
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 投票相关常量
const (
	// OneWeekInSeconds 一周的秒数,超过一周的帖子不允许投票
	OneWeekInSeconds = 7 * 24 * 3600
	// ScorePerVote 每一票的分数权重: 86400秒/天 ÷ 200票 = 432分/票
	ScorePerVote = 432
)

// VoteCache 投票缓存仓储实现
// 同时实现 VoteCacheRepository 和 PostCacheRepository
type VoteCache struct{}

// NewVoteCache 创建 VoteCache 实例
func NewVoteCache() *VoteCache {
	return &VoteCache{}
}

// ========= VoteCacheRepository 实现 =========

// VoteForPost 为帖子投票
func (c *VoteCache) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
	// 1. 判断投票时间限制
	postTime := rdb.ZScore(ctx, getRedisKey(KeyPostTimeZSet), postID).Val()
	if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
		return repointerface.ErrVoteTimeExpire
	}

	// 2. 查询用户之前对该帖子的投票记录
	oldValue := rdb.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID).Val()

	// 3. 如果新旧投票值相同,直接返回
	if value == oldValue {
		return repointerface.ErrVoteRepeated
	}

	// 4. 计算分数变化
	var op float64
	if value > oldValue {
		op = 1
	} else {
		op = -1
	}

	diff := math.Abs(value - oldValue)
	scoreDiff := op * diff * ScorePerVote

	// 5. 使用 Redis Pipeline 保证原子性
	pipeline := rdb.TxPipeline()

	pipeline.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), scoreDiff, postID)
	pipeline.ZIncrBy(ctx, getRedisKey(KeyCommunityPostScorePrefix+communityID), scoreDiff, postID)

	if value == 0 {
		pipeline.ZRem(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID)
	} else {
		pipeline.ZAdd(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), redis.Z{
			Score:  value,
			Member: userID,
		})
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("vote pipeline exec failed (post_id: %s, user_id: %s): %w", postID, userID, err)
	}
	return nil
}

// GetPostsVoteData 批量获取多个帖子的投票数（赞成票数）
func (c *VoteCache) GetPostsVoteData(ctx context.Context, ids []string) (data []int64, err error) {
	pipeline := rdb.Pipeline()

	for _, id := range ids {
		key := getRedisKey(KeyPostVotedZSetPrefix + id)
		pipeline.ZCount(ctx, key, "1", "1")
	}

	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("get posts vote data pipeline exec failed (count: %d): %w", len(ids), err)
	}

	data = make([]int64, 0, len(cmders))
	for _, cmder := range cmders {
		v := cmder.(*redis.IntCmd).Val()
		data = append(data, v)
	}
	return
}

// GetPostVoteData 获取帖子的投票数据
func (c *VoteCache) GetPostVoteData(ctx context.Context, postID string) (upVotes, downVotes int64, err error) {
	data, err := rdb.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:   getRedisKey(KeyPostVotedZSetPrefix + postID),
		Start: 0,
		Stop:  -1,
	}).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("get post vote data failed (post_id: %s): %w", postID, err)
	}

	for _, z := range data {
		if z.Score > 0 {
			upVotes++
		} else if z.Score < 0 {
			downVotes++
		}
	}

	return upVotes, downVotes, nil
}

// GetPostScore 获取帖子的当前分数
func (c *VoteCache) GetPostScore(ctx context.Context, postID string) (float64, error) {
	score, err := rdb.ZScore(ctx, getRedisKey(KeyPostScoreZSet), postID).Result()
	if err != nil {
		return 0, fmt.Errorf("get post score failed (post_id: %s): %w", postID, err)
	}
	return score, nil
}

// GetPostVoteStatus 获取用户对某个帖子的投票状态
func (c *VoteCache) GetPostVoteStatus(ctx context.Context, userID, postID string) (int8, error) {
	score, err := rdb.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	return int8(score), nil
}

// BatchGetPostVoteStatus 批量获取用户对多个帖子的投票状态
func (c *VoteCache) BatchGetPostVoteStatus(ctx context.Context, userID string, postIDs []string) (map[string]int8, error) {
	pipeline := rdb.Pipeline()

	cmds := make([]*redis.FloatCmd, 0, len(postIDs))
	for _, postID := range postIDs {
		cmds = append(cmds, pipeline.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID))
	}

	_, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("batch get post vote status pipeline exec failed (user_id: %s, count: %d): %w", userID, len(postIDs), err)
	}

	result := make(map[string]int8, len(postIDs))
	for i, cmd := range cmds {
		score, err := cmd.Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				result[postIDs[i]] = 0
			} else {
				return nil, err
			}
		} else {
			result[postIDs[i]] = int8(score)
		}
	}

	return result, nil
}

// ========= PostCacheRepository 实现 =========

// CreatePost 创建帖子时初始化 Redis 数据
func (c *VoteCache) CreatePost(ctx context.Context, postID, communityID int64) error {
	pipeline := rdb.TxPipeline()

	timestamp := float64(time.Now().Unix())
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)

	// 全局维度
	pipeline.ZAdd(ctx, getRedisKey(KeyPostTimeZSet), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})
	pipeline.ZAdd(ctx, getRedisKey(KeyPostScoreZSet), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})

	// 社区维度
	pipeline.ZAdd(ctx, getRedisKey(KeyCommunityPostTimePrefix+communityIDStr), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})
	pipeline.ZAdd(ctx, getRedisKey(KeyCommunityPostScorePrefix+communityIDStr), redis.Z{
		Score:  timestamp,
		Member: postIDStr,
	})

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("create post pipeline exec failed (post_id: %d): %w", postID, err)
	}
	return nil
}

// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
func (c *VoteCache) GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error) {
	key := getRedisKey(KeyPostTimeZSet)
	if orderKey == "score" {
		key = getRedisKey(KeyPostScoreZSet)
	}
	start := (page - 1) * size
	end := start + size - 1

	ids, err := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  end,
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("get post ids failed (order: %s): %w", orderKey, err)
	}
	return ids, nil
}

// GetCommunityPostIDsInOrder 按照指定顺序获取指定社区的帖子ID列表
func (c *VoteCache) GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error) {
	keyPrefix := KeyCommunityPostTimePrefix
	if orderKey == "score" {
		keyPrefix = KeyCommunityPostScorePrefix
	}

	key := getRedisKey(keyPrefix + strconv.FormatInt(communityID, 10))

	start := (page - 1) * size
	end := start + size - 1

	ids, err := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  end,
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("get community post ids failed (community_id: %d, order: %s): %w", communityID, orderKey, err)
	}
	return ids, nil
}

// ConvertIDsToInt64 将字符串ID列表转换为int64列表
func ConvertIDsToInt64(ids []string) ([]int64, error) {
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		intID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, intID)
	}
	return result, nil
}
