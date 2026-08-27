package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	postResp "bluebell/internal/dto/response/post"
	"bluebell/internal/model"

	goredis "github.com/redis/go-redis/v9"
)

// ========== Redis Keys 常量 ==========

const (
	keyPostTimeZSet             = "post:time"   // bluebell:post:time - 所有帖子按时间排序
	keyPostScoreZSet            = "post:score"  // bluebell:post:score - 所有帖子按分数排序
	keyPostVotedZSetPrefix      = "post:voted:" // bluebell:post:voted:{postID} - 帖子的投票记录
	keyPostMetaPrefix           = "post:meta:"  // bluebell:post:meta:{postID} - 帖子元数据 Hash
	keyCommunityPostTimePrefix  = "community:post:time:"
	keyCommunityPostScorePrefix = "community:post:score:"
	keyPostCreateDedupPrefix    = "post:create_dedup:" // post:create_dedup:{authorID}:{titleHash} - 防重复提交
	keyPostDetailPrefix         = "post:detail:"       // bluebell:post:detail:{postID} - 帖子实体快照 JSON 缓存
	defaultPostDetailTTL        = 24 * time.Hour
	// 投票相关常量
	oneWeekInSeconds = 100 * 7 * 24 * 3600 // 增加到100周，方便压测
	// Gravity 算法衰减因子（Reddit/Hacker News 标准值）
	// [防御] 值越大衰减越快，1.8 是 HN 验证过的经验值
	gravity = 1.8
)

// ========== PostCache ==========

// PostCache 帖子缓存数据访问对象
// 负责帖子在 Redis 中的排序、分页与投票数据
type PostCache struct {
	rdb *goredis.Client
}

// NewPostCache 创建 PostCache 实例
func NewPostCache(rdb *goredis.Client) *PostCache {
	return &PostCache{rdb: rdb}
}

// NewPostCacheWithRefresher 创建 PostCache 实例和热度刷新器
func NewPostCacheWithRefresher(rdb *goredis.Client) (*PostCache, *HotScoreRefresher) {
	c := &PostCache{rdb: rdb}
	refresher := NewHotScoreRefresher(rdb)
	return c, refresher
}

func timeNow() int64 {
	return time.Now().Unix()
}

// ========== 帖子缓存操作 ==========

// CreatePost 创建帖子时初始化 Redis 数据（全维度预热）
func (c *PostCache) CreatePost(ctx context.Context, postID, communityID int64) error {
	postIDStr := strconv.FormatInt(postID, 10)
	communityIDStr := strconv.FormatInt(communityID, 10)
	timestamp := float64(time.Now().Unix())

	// 使用 TxPipelined 开启 Redis 事务管道：将 5 个写动作打包成 1 个网络包
	_, err := c.rdb.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
		// 1. 全局：最新排行榜
		pipe.ZAdd(ctx, redisKey(keyPostTimeZSet), goredis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 2. 全局：最热排行榜（初始分为时间戳）
		pipe.ZAdd(ctx, redisKey(keyPostScoreZSet), goredis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 3. 社区：社区内最新排行榜
		pipe.ZAdd(ctx, redisKey(keyCommunityPostTimePrefix+communityIDStr), goredis.Z{
			Score:  timestamp,
			Member: postIDStr,
		})
		// 4. 社区：社区内最热排行榜
		pipe.ZAdd(ctx, redisKey(keyCommunityPostScorePrefix+communityIDStr), goredis.Z{
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
		return fmt.Errorf("create post pipeline failed (post_id: %d): %w", postID, err)
	}
	return nil
}

// CheckDuplicateSubmit 通用短时去重：基于请求指纹（用户ID + 请求路径 + 参数hash）防手抖连点
// fingerprint 由调用方拼装，格式建议：userID:path:paramHash
func (c *PostCache) CheckDuplicateSubmit(ctx context.Context, fingerprint string, ttl time.Duration) error {
	hash := sha256.Sum256([]byte(fingerprint))
	fpHash := hex.EncodeToString(hash[:16]) // 取前 32 字符

	key := redisKey(keyPostCreateDedupPrefix + fpHash)

	ok, err := c.rdb.SetNX(ctx, key, 1, ttl).Result()
	if err != nil {
		return fmt.Errorf("check duplicate submit failed: %w", err)
	}
	if !ok {
		return model.ErrDuplicateSubmit
	}
	return nil
}

// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
func (c *PostCache) GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error) {
	key := redisKey(keyPostTimeZSet)
	if orderKey == "score" {
		key = redisKey(keyPostScoreZSet)
	}
	start := (page - 1) * size
	end := start + size - 1

	ids, err := c.rdb.ZRangeArgs(ctx, goredis.ZRangeArgs{
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
func (c *PostCache) GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error) {
	kp := keyCommunityPostTimePrefix
	if orderKey == "score" {
		kp = keyCommunityPostScorePrefix
	}

	key := redisKey(kp + strconv.FormatInt(communityID, 10))

	start := (page - 1) * size
	end := start + size - 1

	ids, err := c.rdb.ZRangeArgs(ctx, goredis.ZRangeArgs{
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

// VoteForPost 为帖子投票
// 使用 Lua 脚本将时效校验、防重检查、投票记录更新、元数据计数变更、Gravity 热度计算及排行榜 ZSet 写入封装为单次原子化事务
// 彻底规避高并发场景下的重复投票、分数计算脏读以及排行榜分数被旧值覆盖的问题
func (c *PostCache) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
	const voteLua = `
		local voteKey = KEYS[1]
		local metaKey = KEYS[2]
		local globalScoreKey = KEYS[3]
		local communityScoreKey = KEYS[4]

		local userID = ARGV[1]
		local postID = ARGV[2]
		local newValue = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		local expireSec = tonumber(ARGV[5])
		local gravity = tonumber(ARGV[6])

		-- 1. 检查帖子元数据是否存在
		local meta = redis.call('HMGET', metaKey, 'create_time', 'vote_up', 'vote_down')
		local createTimeStr = meta[1]
		if not createTimeStr then
			return 'err_not_found'
		end

		local createTime = tonumber(createTimeStr)
		-- 2. 判断投票时间限制
		if (now - createTime) > expireSec then
			return 'err_expire'
		end

		-- 3. 查询旧投票值
		local oldValue = redis.call('ZSCORE', voteKey, userID)
		if not oldValue then
			oldValue = 0
		else
			oldValue = tonumber(oldValue)
		end

		-- 4. 新旧值相同，拒绝重复投票
		if newValue == oldValue then
			return 'err_repeated'
		end

		-- 5. 计算增量
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

		-- 6. 更新投票记录
		if newValue == 0 then
			redis.call('ZREM', voteKey, userID)
		else
			redis.call('ZADD', voteKey, newValue, userID)
		end

		-- 7. 原子更新元数据 Hash 中的投票计数
		local currentVoteUp = tonumber(meta[2] or 0)
		local currentVoteDown = tonumber(meta[3] or 0)
		local newVoteUp = currentVoteUp + voteUpDelta
		local newVoteDown = currentVoteDown + voteDownDelta

		if voteUpDelta ~= 0 then
			redis.call('HINCRBY', metaKey, 'vote_up', voteUpDelta)
		end
		if voteDownDelta ~= 0 then
			redis.call('HINCRBY', metaKey, 'vote_down', voteDownDelta)
		end

		-- 8. 原子计算 Gravity 热度分数并写入排行榜
		local votes = newVoteUp - newVoteDown
		local score = 0
		if votes > 0 then
			local hours = (now - createTime) / 3600.0
			if hours < 0 then
				hours = 0
			end
			local denominator = math.pow(hours + 2, gravity)
			score = (votes - 1) / denominator
		end

		-- 9. 原子更新全局和社区热度排行榜 ZSet
		redis.call('ZADD', globalScoreKey, score, postID)
		if communityScoreKey and communityScoreKey ~= '' then
			redis.call('ZADD', communityScoreKey, score, postID)
		end

		return 'ok'
	`

	var communityScoreKey string
	if communityID != "" {
		communityScoreKey = redisKey(keyCommunityPostScorePrefix + communityID)
	}

	keys := []string{
		redisKey(keyPostVotedZSetPrefix + postID),
		redisKey(keyPostMetaPrefix + postID),
		redisKey(keyPostScoreZSet),
		communityScoreKey,
	}

	result, err := c.rdb.Eval(ctx, voteLua, keys,
		userID,
		postID,
		value,
		timeNow(),
		oneWeekInSeconds,
		gravity,
	).Text()

	if err != nil {
		return fmt.Errorf("vote lua eval failed (post_id: %s, user_id: %s): %w", postID, userID, err)
	}

	switch result {
	case "ok":
		return nil
	case "err_repeated":
		return model.ErrVoteRepeated
	case "err_expire":
		return model.ErrVoteTimeExpire
	case "err_not_found":
		return model.ErrNotFound
	case "err_unknown":
		return fmt.Errorf("unknown vote state change (post_id: %s, user_id: %s)", postID, userID)
	default:
		return fmt.Errorf("unexpected vote lua result: %s (post_id: %s, user_id: %s)", result, postID, userID)
	}
}

// GetPostsVoteData 批量获取多个帖子的投票数（净投票数 = vote_up - vote_down）
// 直接从"原始账本" ZSet 中统计，确保数据绝对准确（但性能略低于从 Hash 直接读取）
func (c *PostCache) GetPostsVoteData(ctx context.Context, ids []string) ([]int64, error) {
	// [防御] 空切片直接返回
	if len(ids) == 0 {
		return nil, nil
	}

	pipe := c.rdb.Pipeline()
	// 每个帖子需要两个统计操作：1 (赞成) 和 -1 (反对)
	upCmds := make([]*goredis.IntCmd, len(ids))
	downCmds := make([]*goredis.IntCmd, len(ids))

	for i, postID := range ids {
		key := redisKey(keyPostVotedZSetPrefix + postID)
		upCmds[i] = pipe.ZCount(ctx, key, "1", "1")
		downCmds[i] = pipe.ZCount(ctx, key, "-1", "-1")
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil, fmt.Errorf("batch get vote count from zset failed: %w", err)
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
func (c *PostCache) DeletePost(ctx context.Context, postID, communityID int64) error {
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

	// 帖子实体快照缓存
	pipeline.Del(ctx, redisKey(keyPostDetailPrefix+postIDStr))

	// 投票记录 ZSet
	pipeline.Del(ctx, redisKey(keyPostVotedZSetPrefix+postIDStr))
	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete post cache cleanup failed (post_id: %d): %w", postID, err)
	}
	return nil
}

// GetPostCommunityID 从 Redis 帖子元数据 Hash 中获取社区 ID
func (c *PostCache) GetPostCommunityID(ctx context.Context, postID int64) (int64, error) {
	postIDStr := strconv.FormatInt(postID, 10)
	val, err := c.rdb.HGet(ctx, redisKey(keyPostMetaPrefix+postIDStr), "community").Result()
	if err != nil {
		return 0, fmt.Errorf("get community id failed (post_id: %s): %w", postIDStr, err)
	}
	communityID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse community id failed (post_id: %s): %w", postIDStr, err)
	}
	return communityID, nil
}

// ========== 帖子实体快照缓存操作 (Plan 2: MGet + 0 SQL 内存级读取) ==========

// GetPostDetailCache 从 Redis 获取单个帖子快照
func (c *PostCache) GetPostDetailCache(ctx context.Context, postID string) (*postResp.DetailResponse, error) {
	key := redisKey(keyPostDetailPrefix + postID)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var resp postResp.DetailResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MGetPostDetails 批量从 Redis 获取帖子实体快照，返回命中映射及未命中的 post_id 列表
func (c *PostCache) MGetPostDetails(ctx context.Context, postIDs []string) (hitMap map[string]*postResp.DetailResponse, missedIDs []string, err error) {
	hitMap = make(map[string]*postResp.DetailResponse, len(postIDs))
	if len(postIDs) == 0 {
		return hitMap, nil, nil
	}

	keys := make([]string, len(postIDs))
	for i, id := range postIDs {
		keys[i] = redisKey(keyPostDetailPrefix + id)
	}

	vals, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		// 缓存故障时全部当做未命中，平滑降级
		return hitMap, postIDs, nil
	}

	for i, v := range vals {
		id := postIDs[i]
		if v == nil {
			missedIDs = append(missedIDs, id)
			continue
		}
		str, ok := v.(string)
		if !ok || str == "" {
			missedIDs = append(missedIDs, id)
			continue
		}
		var detail postResp.DetailResponse
		if err := json.Unmarshal([]byte(str), &detail); err != nil {
			missedIDs = append(missedIDs, id)
			continue
		}
		hitMap[id] = &detail
	}

	return hitMap, missedIDs, nil
}

// SetPostDetails 批量写入帖子实体快照到 Redis
func (c *PostCache) SetPostDetails(ctx context.Context, posts []*postResp.DetailResponse, ttl time.Duration) error {
	if len(posts) == 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = defaultPostDetailTTL
	}

	pipe := c.rdb.Pipeline()
	for _, p := range posts {
		if p == nil || p.ID == "" {
			continue
		}
		data, err := json.Marshal(p)
		if err != nil {
			continue
		}
		key := redisKey(keyPostDetailPrefix + p.ID)
		pipe.Set(ctx, key, data, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// DeletePostDetailCache 删除单个帖子实体快照缓存
func (c *PostCache) DeletePostDetailCache(ctx context.Context, postID string) error {
	return c.rdb.Del(ctx, redisKey(keyPostDetailPrefix+postID)).Err()
}
