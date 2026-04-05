package postcache

import (
	"bluebell/internal/domain/cachedomain"
	"bluebell/pkg/errorx"
	"context"
	"github.com/redis/go-redis/v9"
	"math"
	"strconv"
	"strings"
	"time"
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
	oneWeekInSeconds = 7 * 24 * 3600 // 一周的秒数，超过一周的帖子不允许投票
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
func NewCacheWithRefresher(rdb *redis.Client, config *HotScoreRefresherConfig) (cachedomain.PostRepository, *HotScoreRefresher) {
	c := &cacheStruct{rdb: rdb}
	refresher := NewHotScoreRefresher(rdb, c, config)
	return c, refresher
}

func redisKey(key string) string {
	return keyPrefix + key
}

func timeNow() int64 {
	return time.Now().Unix()
}

// ========== PostRepository 实现 ==========

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

	// 初始化帖子元数据 Hash（用于 Gravity 算法）
	// [防御] HSet 幂等，vote_up/vote_down 初始化为 0 避免后续 HIncrBy 行为不一致
	pipeline.HSet(ctx, redisKey(keyPostMetaPrefix+postIDStr), map[string]interface{}{
		"create_time": strconv.FormatInt(timeNow(), 10),
		"vote_up":     0,
		"vote_down":   0,
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
// 使用 Gravity 算法：更新 Hash 中的投票计数，重新计算热度分数并覆盖 ZSet
// 使用 Lua 脚本保证"检查旧值 + 更新投票记录"的原子性，防止并发重复投票
func (c *cacheStruct) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
	// 1. 判断投票时间限制
	postTime := c.rdb.ZScore(ctx, redisKey(keyPostTimeZSet), postID).Val()
	if float64(timeNow())-postTime > oneWeekInSeconds {
		return errorx.ErrVoteTimeExpire
	}

	// 2. 使用 Lua 脚本原子执行：检查旧值 → 计算增量 → 更新投票记录
	// KEYS[1] = vote record ZSet (bluebell:post:voted:{postID})
	// ARGV[1] = userID
	// ARGV[2] = new vote value (1/-1/0)
	// 返回值: voteUpDelta,voteDownDelta (如 "1,0" / "-1,1" / "0,0")
	// 错误码: "err_repeated" / "err_unknown"
	const voteLua = `
		local key = KEYS[1]
		local userID = ARGV[1]
		local newValue = tonumber(ARGV[2])

		-- 查询旧投票值
		local oldValue = redis.call('ZSCORE', key, userID)
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
			redis.call('ZREM', key, userID)
		else
			redis.call('ZADD', key, newValue, userID)
		end

		return voteUpDelta .. ',' .. voteDownDelta
	`

	result, err := c.rdb.Eval(ctx, voteLua, []string{redisKey(keyPostVotedZSetPrefix + postID)}, userID, value).Text()
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "vote lua eval failed (post_id: %s, user_id: %s)", postID, userID)
	}

	if result == "err_repeated" {
		return errorx.ErrVoteRepeated
	}
	if result == "err_unknown" {
		return errorx.Newf(errorx.CodeCacheError, "unknown vote state change (post_id: %s, user_id: %s)", postID, userID)
	}

	// 3. 解析 Lua 返回的增量
	parts := strings.SplitN(result, ",", 2)
	if len(parts) != 2 {
		return errorx.Newf(errorx.CodeCacheError, "invalid vote lua result: %s", result)
	}
	voteUpDelta, _ := strconv.ParseInt(parts[0], 10, 64)
	voteDownDelta, _ := strconv.ParseInt(parts[1], 10, 64)

	// 4. 更新 Hash 中的投票计数
	if err := c.updatePostVoteCount(ctx, postID, voteUpDelta, voteDownDelta); err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "update post vote count failed (post_id: %s)", postID)
	}

	// 5. 重新计算 Gravity 分数并更新 ZSet
	if err := c.batchRefreshGravityScores(ctx, []string{postID}); err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "refresh gravity score failed (post_id: %s)", postID)
	}

	return nil
}

// GetPostsVoteData 批量获取多个帖子的投票数（赞成票数）
func (c *cacheStruct) GetPostsVoteData(ctx context.Context, ids []string) (data []int64, err error) {
	return c.getPostVoteCounts(ctx, ids)
}

// GetTopPostIDsWithScores 获取全站排行榜数据（Top N 帖子ID及其分数）
func (c *cacheStruct) GetTopPostIDsWithScores(ctx context.Context, size int64) (ids []string, scores []float64, err error) {
	key := redisKey(keyPostScoreZSet)

	results, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, size-1).Result()
	if err != nil {
		return nil, nil, errorx.Wrapf(err, errorx.CodeCacheError, "get top posts with scores failed (size: %d)", size)
	}

	ids = make([]string, 0, len(results))
	scores = make([]float64, 0, len(results))

	for _, z := range results {
		ids = append(ids, z.Member.(string))
		scores = append(scores, z.Score)
	}

	return ids, scores, nil
}

// GetCommunityTopPostIDsWithScores 获取社区排行榜数据（Top N 帖子ID及其分数）
func (c *cacheStruct) GetCommunityTopPostIDsWithScores(ctx context.Context, communityID, size int64) (ids []string, scores []float64, err error) {
	key := redisKey(keyCommunityPostScorePrefix + strconv.FormatInt(communityID, 10))

	results, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, size-1).Result()
	if err != nil {
		return nil, nil, errorx.Wrapf(err, errorx.CodeCacheError, "get community top posts with scores failed (community_id: %d, size: %d)", communityID, size)
	}

	ids = make([]string, 0, len(results))
	scores = make([]float64, 0, len(results))

	for _, z := range results {
		ids = append(ids, z.Member.(string))
		scores = append(scores, z.Score)
	}

	return ids, scores, nil
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

// updatePostVoteCount 更新帖子投票计数（增量更新）
func (c *cacheStruct) updatePostVoteCount(ctx context.Context, postID string, voteUpDelta, voteDownDelta int64) error {
	key := redisKey(keyPostMetaPrefix + postID)
	pipe := c.rdb.Pipeline()

	// [防御] 只在 delta != 0 时才发命令，减少 Pipeline 命令数量
	if voteUpDelta != 0 {
		pipe.HIncrBy(ctx, key, "vote_up", voteUpDelta)
	}
	if voteDownDelta != 0 {
		pipe.HIncrBy(ctx, key, "vote_down", voteDownDelta)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// getPostVoteCounts 批量获取帖子投票数
func (c *cacheStruct) getPostVoteCounts(ctx context.Context, postIDs []string) ([]int64, error) {
	// [防御] 空切片直接返回
	if len(postIDs) == 0 {
		return nil, nil
	}

	pipe := c.rdb.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(postIDs))
	for i, postID := range postIDs {
		cmds[i] = pipe.HGetAll(ctx, redisKey(keyPostMetaPrefix+postID))
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	counts := make([]int64, len(postIDs))
	for i, cmd := range cmds {
		result, _ := cmd.Result()
		// [防御] 保证 counts 长度和 postIDs 严格一致，Hash 不存在时 voteUp=0 占位
		voteUp, _ := strconv.ParseInt(result["vote_up"], 10, 64)
		counts[i] = voteUp
	}
	return counts, nil
}

// batchRefreshGravityScores 批量刷新帖子的 Gravity 分数到 ZSet
func (c *cacheStruct) batchRefreshGravityScores(ctx context.Context, postIDs []string) error {
	// [防御] 空切片直接返回
	if len(postIDs) == 0 {
		return nil
	}

	pipe := c.rdb.Pipeline()

	for _, postID := range postIDs {
		result, err := c.rdb.HGetAll(ctx, redisKey(keyPostMetaPrefix+postID)).Result()
		// [防御] 单个帖子失败不影响其他帖子，不能因为一个帖子失败就 abort 整批
		if err != nil || len(result) == 0 {
			continue
		}

		// [防御] ParseInt 忽略 error，格式不对时降级为 0
		createTimeUnix, _ := strconv.ParseInt(result["create_time"], 10, 64)
		voteUp, _ := strconv.ParseInt(result["vote_up"], 10, 64)
		voteDown, _ := strconv.ParseInt(result["vote_down"], 10, 64)

		createTime := time.Unix(createTimeUnix, 0)
		score := CalculateGravityScore(voteUp, voteDown, createTime)

		pipe.ZAdd(ctx, redisKey(keyPostScoreZSet), redis.Z{
			Score:  score,
			Member: postID,
		})
	}

	// [防御] Pipeline 即使为空 Exec 也是安全的
	_, err := pipe.Exec(ctx)
	return err
}

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
