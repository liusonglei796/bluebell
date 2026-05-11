package postcache

import (
	"bluebell/internal/domain/cachedomain"
	"bluebell/pkg/errorx"
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ========== Redis Keys 常量 ==========

const (
	keyPrefix                   = "bluebell:"
	keyPostTimeZSet             = "post:time"   // bluebell:post:time - 所有帖子按时间排序
	keyPostScoreZSet            = "post:score"  // bluebell:post:score - 所有帖子按分数排序
	keyPostVotedZSetPrefix      = "post:voted:" // bluebell:post:voted:{postID} - 帖子的投票记录
	keyPostMetaPrefix           = "post:meta:"  // bluebell:post:meta:{postID} - 帖子元数据 Hash
	keyCommunityPostTimePrefix  = "community:post:time:"
	keyCommunityPostScorePrefix = "community:post:score:"
	// 投票相关常量
	oneWeekInSeconds = 100 * 7 * 24 * 3600 // 增加到100周，方便压测
	// Gravity 算法衰减因子（Reddit/Hacker News 标准值）
	// [防御] 值越大衰减越快，1.8 是 HN 验证过的经验值
	gravity = 1.8
)

// ========== cacheStruct ==========

// cacheStruct 帖子缓存仓储实现
// 实现 PostRepository（含帖子排序、分页与投票）
type cacheStruct struct {
	rdb *redis.Client
}

// NewCache 创建 cacheStruct 实例
func NewCache(rdb *redis.Client) cachedomain.PostRepository {
	return &cacheStruct{rdb: rdb}
}

// NewCacheWithRefresher 创建 cacheStruct 实例和热度刷新器
func NewCacheWithRefresher(rdb *redis.Client) (cachedomain.PostRepository, *HotScoreRefresher) {
	c := &cacheStruct{rdb: rdb}
	refresher := NewHotScoreRefresher(rdb)
	return c, refresher
}

func redisKey(key string) string {
	return keyPrefix + key
}

func timeNow() int64 {
	return time.Now().Unix()
}

// ========== PostRepository 实现 ==========

// CreatePost 创建帖子时初始化 Redis 数据（全维度预热）
func (c *cacheStruct) CreatePost(ctx context.Context, postID, communityID int64) error {
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)
	timestamp := float64(time.Now().Unix())

	// 使用 TxPipelined 开启 Redis 事务管道：将 5 个写动作打包成 1 个网络包
	_, err := c.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// 1. 全局：最新排行榜
		pipe.ZAdd(ctx, redisKey(keyPostTimeZSet), redis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 2. 全局：最热排行榜（初始分为时间戳）
		pipe.ZAdd(ctx, redisKey(keyPostScoreZSet), redis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 3. 社区：社区内最新排行榜
		pipe.ZAdd(ctx, redisKey(keyCommunityPostTimePrefix+communityIDStr), redis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 4. 社区：社区内最热排行榜
		pipe.ZAdd(ctx, redisKey(keyCommunityPostScorePrefix+communityIDStr), redis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 5. 元数据：初始化元数据 Hash (供投票 API 的 HEXISTS 校验)
		pipe.HSet(ctx, redisKey(keyPostMetaPrefix+postIDStr), map[string]interface{}{
			"create_time": strconv.FormatInt(int64(timestamp), 10),
			"community":   communityIDStr, // 存入社区 ID，方便投票 Lua 脚本拿
			"vote_up":     0,
			"vote_down":   0,
		})
		return nil
	})

	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "create post pipeline failed (post_id: %d)", postID)
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
// 使用 Gravity 算法：Lua 原子更新投票记录和 Hash 计数，Go 侧重新计算热度分数并覆盖 ZSet
// 使用 Lua 脚本保证"检查旧值 + 更新投票记录 + 更新计数"的原子性，防止并发重复投票
func (c *cacheStruct) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
	// 1. 判断投票时间限制
	postTime := c.rdb.ZScore(ctx, redisKey(keyPostTimeZSet), postID).Val()
	if float64(timeNow())-postTime > oneWeekInSeconds {
		return errorx.ErrVoteTimeExpire
	}

	// 2. 使用 Lua 脚本原子执行：检查旧值 → 更新投票记录 → 更新 Hash 计数
	// KEYS[1] = vote record ZSet (bluebell:post:voted:{postID})
	// KEYS[2] = post meta Hash (bluebell:post:meta:{postID})
	// ARGV[1] = userID
	// ARGV[2] = new vote value (1/-1/0)
	// 返回值: voteUp,voteDown,createTime (如 "1,0,1710000000")
	// 错误码: "err_repeated" / "err_unknown"
	const voteLua = `
		local voteKey = KEYS[1]
		local metaKey = KEYS[2]
		local userID = ARGV[1]
		local newValue = tonumber(ARGV[2])
		-- 查询旧投票值
		local oldValue = redis.call('ZSCORE', voteKey, userID)
		if not oldValue then
			oldValue = 0
		else
			oldValue = tonumber(oldValue)
		end

		-- 新旧值相同，拒绝
		if newValue == oldValue then
			return 'err_repeated'
		end

		-- 计算增量
		local voteUpDelta = 0
		local voteDownDelta = 0
		if oldValue == 0 and newValue == 1 then
			voteUpDelta = 1
		elseif oldValue == 0 and newValue == -1 then
			voteDownDelta = 1
		elseif oldValue == 1 and newValue == 0 then
			voteUpDelta = -1
		elseif oldValue == 1 and newValue == -1 then
			voteUpDelta = -1
			voteDownDelta = 1
		elseif oldValue == -1 and newValue == 0 then
			voteDownDelta = -1
		elseif oldValue == -1 and newValue == 1 then
			voteDownDelta = -1
			voteUpDelta = 1
		else
			return 'err_unknown'
		end

		-- 更新投票记录
		if newValue == 0 then
			redis.call('ZREM', voteKey, userID)
		else
			redis.call('ZADD', voteKey, newValue, userID)
		end

		-- 更新帖子元数据 Hash 中的投票计数
		if voteUpDelta ~= 0 then
			redis.call('HINCRBY', metaKey, 'vote_up', voteUpDelta)
		end
		if voteDownDelta ~= 0 then
			redis.call('HINCRBY', metaKey, 'vote_down', voteDownDelta)
		end

		local result = redis.call('HMGET', metaKey, 'vote_up', 'vote_down', 'create_time')
		return result[1] .. ',' .. result[2] .. ',' .. result[3]
	`

	result, err := c.rdb.Eval(ctx, voteLua, []string{
		redisKey(keyPostVotedZSetPrefix + postID),
		redisKey(keyPostMetaPrefix + postID),
	}, userID, value).Text()
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "vote lua eval failed (post_id: %s, user_id: %s)", postID, userID)
	}

	if result == "err_repeated" {
		return errorx.ErrVoteRepeated
	}
	if result == "err_unknown" {
		return errorx.Newf(errorx.CodeCacheError, "unknown vote state change (post_id: %s, user_id: %s)", postID, userID)
	}

	// 3. 解析 Lua 返回的最新投票数据
	parts := strings.SplitN(result, ",", 3)
	if len(parts) != 3 {
		return errorx.Newf(errorx.CodeCacheError, "invalid vote lua result: %s", result)
	}
	voteUp, _ := strconv.ParseInt(parts[0], 10, 64)
	voteDown, _ := strconv.ParseInt(parts[1], 10, 64)
	createTimeUnix, _ := strconv.ParseInt(parts[2], 10, 64)

	// 4. 基于最新总票数重新计算 Gravity 分数并更新 ZSet（全局 + 社区）
	createTime := time.Unix(createTimeUnix, 0)
	score := CalculateGravityScore(voteUp, voteDown, createTime)

	// 更新全局热度榜
	if err := c.rdb.ZAdd(ctx, redisKey(keyPostScoreZSet), redis.Z{
		Score:  score,
		Member: postID,
	}).Err(); err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "update global gravity score failed (post_id: %s)", postID)
	}

	// 更新社区热度榜
	if err := c.rdb.ZAdd(ctx, redisKey(keyCommunityPostScorePrefix+communityID), redis.Z{
		Score:  score,
		Member: postID,
	}).Err(); err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "update community gravity score failed (post_id: %s, community_id: %s)", postID, communityID)
	}

	return nil
}

// GetPostsVoteData 批量获取多个帖子的投票数（净投票数 = vote_up - vote_down）
// 直接从“原始账本” ZSet 中统计，确保数据绝对准确（但性能略低于从 Hash 直接读取）
func (c *cacheStruct) GetPostsVoteData(ctx context.Context, ids []string) ([]int64, error) {
	// [防御] 空切片直接返回
	if len(ids) == 0 {
		return nil, nil
	}

	pipe := c.rdb.Pipeline()
	// 每个帖子需要两个统计操作：1 (赞成) 和 -1 (反对)
	upCmds := make([]*redis.IntCmd, len(ids))
	downCmds := make([]*redis.IntCmd, len(ids))

	for i, postID := range ids {
		key := redisKey(keyPostVotedZSetPrefix + postID)
		upCmds[i] = pipe.ZCount(ctx, key, "1", "1")
		downCmds[i] = pipe.ZCount(ctx, key, "-1", "-1")
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, errorx.Wrapf(err, errorx.CodeCacheError, "batch get vote count from zset failed")
	}

	counts := make([]int64, len(ids))
	for i := range ids {
		up, _ := upCmds[i].Result()
		down, _ := downCmds[i].Result()
		counts[i] = up - down
	}
	return counts, nil
}

// ========== Gravity 算法 ==========

// CalculateGravityScore 使用 Gravity 算法计算帖子热度分数
// 公式: score = (votes - 1) / (hours_since_submission + 2)^gravity
// votes = voteUp - voteDown（净投票数）
func CalculateGravityScore(voteUp, voteDown int64, createTime time.Time) float64 {
	// [防御] 转 float64 再做除法，避免 int 整除丢失精度
	votes := float64(voteUp - voteDown)

	// [防御] 净票数为 0 或负数时直接返回 0
	if votes <= 0 {
		return 0
	}

	hoursSinceSubmission := time.Since(createTime).Hours()

	// [防御] 防止服务器时钟回拨导致负数，负数会导致分母 < 2^1.8 分数异常膨胀
	if hoursSinceSubmission < 0 {
		hoursSinceSubmission = 0
	}

	// [防御] +2 防止新帖 hours=0 时分母为 0，同时给新帖基础曝光窗口
	denominator := math.Pow(hoursSinceSubmission+2, gravity)

	// [防御] votes-1 是 HN 的设计：第一票不算分
	return (votes - 1) / denominator
}

// ========== Hash 元数据操作 ==========

// DeletePost 删除帖子时清理 Redis 缓存
// 清理范围：全局 ZSet（time/score）、社区 ZSet（time/score）、元数据 Hash、投票记录 ZSet
func (c *cacheStruct) DeletePost(ctx context.Context, postID, communityID int64) error {
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)

	pipeline := c.rdb.TxPipeline()

	// 全局维度 ZSet
	pipeline.ZRem(ctx, redisKey(keyPostTimeZSet), postIDStr)
	pipeline.ZRem(ctx, redisKey(keyPostScoreZSet), postIDStr)

	// 社区维度 ZSet
	pipeline.ZRem(ctx, redisKey(keyCommunityPostTimePrefix+communityIDStr), postIDStr)
	pipeline.ZRem(ctx, redisKey(keyCommunityPostScorePrefix+communityIDStr), postIDStr)

	// 帖子元数据 Hash
	pipeline.Del(ctx, redisKey(keyPostMetaPrefix+postIDStr))

	// 投票记录 ZSet
	pipeline.Del(ctx, redisKey(keyPostVotedZSetPrefix+postIDStr))

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "delete post cache cleanup failed (post_id: %d)", postID)
	}
	return nil
}

// CheckPostExists 检查帖子是否存在
func (c *cacheStruct) CheckPostExists(ctx context.Context, postID int64) (bool, error) {
	postIDStr := strconv.FormatInt(postID, 10)
	// 使用 EXISTS 命令检查元数据 Hash 是否存在
	// 为什么用这个：EXISTS 是 O(1) 操作，且只返回 0 或 1，网络负载极低
	n, err := c.rdb.Exists(ctx, redisKey(keyPostMetaPrefix+postIDStr)).Result()
	if err != nil {
		return false, errorx.Wrapf(err, errorx.CodeCacheError, "redis EXISTS failed (post_id: %d)", postID)
	}
	return n > 0, nil
}

