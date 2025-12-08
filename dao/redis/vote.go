package redis

import (
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 投票相关常量
const (
	// 一周的秒数,超过一周的帖子不允许投票
	OneWeekInSeconds = 7 * 24 * 3600
	// 每一票的分数权重: 86400秒/天 ÷ 200票 = 432分/票
	// 含义: 一个帖子需要200张赞成票才能在热榜上"续命"一天
	ScorePerVote = 432
)

// 投票相关错误
var (
	ErrVoteTimeExpire = errors.New("投票时间已过")
	ErrVoteRepeated   = errors.New("不允许重复投票")
)

// VoteForPost 为帖子投票
// 参数:
//   userID: 投票用户ID (字符串格式)
//   postID: 目标帖子ID (字符串格式)
//   value: 投票值 (1:赞成, -1:反对, 0:取消投票)
//
// 核心算法:
//   利用新旧投票值的差值,计算帖子分数的变化量
//   例如: 从赞成(1)改为反对(-1), 差值为2, 分数变化为 -2*432 = -864
func VoteForPost(userID, postID string, value float64) error {
	// 1. 判断投票时间限制
	// 从 Redis 的 ZSet 中获取帖子的发布时间戳
	postTime := rdb.ZScore(ctx, getRedisKey(KeyPostTimeZSet), postID).Val()

	// 如果当前时间距离发帖时间超过一周,不允许投票
	if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
		return ErrVoteTimeExpire
	}

	// 2. 查询用户之前对该帖子的投票记录
	// key: bluebell:post:voted:{post_id}
	// 该 ZSet 的 member 是 userID, score 是投票值(1/-1/0)
	oldValue := rdb.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID).Val()

	// 3. 如果新旧投票值相同,直接返回(避免重复投票)
	if value == oldValue {
		return ErrVoteRepeated
	}

	// 4. 计算分数变化
	// op: 操作方向 (1表示加分, -1表示减分)
	var op float64
	if value > oldValue {
		op = 1 // 例如: 从0到1, 从-1到0, 从-1到1 都是加分
	} else {
		op = -1 // 例如: 从1到0, 从0到-1, 从1到-1 都是减分
	}

	// diff: 新旧投票值的差值绝对值
	// 例如: 从1变为-1, diff=2; 从0变为1, diff=1
	diff := math.Abs(value - oldValue)

	// 5. 使用 Redis Pipeline 保证原子性
	// 需要同时更新两个 ZSet: 帖子分数表 和 用户投票记录表
	pipeline := rdb.TxPipeline()

	// 5.1 更新帖子的总分数
	// key: bluebell:post:score
	// 分数变化 = 操作方向 * 差值 * 单票分数
	pipeline.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), op*diff*ScorePerVote, postID)

	// 5.2 更新用户的投票记录
	if value == 0 {
		// 如果是取消投票,从 ZSet 中删除该用户记录
		pipeline.ZRem(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID)
	} else {
		// 否则,添加或更新用户的投票记录
		pipeline.ZAdd(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), redis.Z{
			Score:  value,  // 1 或 -1
			Member: userID, // 用户ID
		})
	}

	// 6. 执行 Pipeline 中的所有命令
	_, err := pipeline.Exec(ctx)
	return err
}

// CreatePost 创建帖子时初始化 Redis 数据
// 在发帖时调用,设置帖子的初始分数和发布时间
func CreatePost(postID, communityID int64) error {
	pipeline := rdb.TxPipeline()

	// 1. 将帖子发布时间存入 ZSet
	// key: bluebell:post:time, score: 当前时间戳, member: postID
	pipeline.ZAdd(ctx, getRedisKey(KeyPostTimeZSet), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: postID,
	})

	// 2. 将帖子初始分数存入 ZSet
	// key: bluebell:post:score, score: 初始分数(发布时间戳), member: postID
	// 初始分数设置为发布时间戳,这样新帖子会排在前面
	pipeline.ZAdd(ctx, getRedisKey(KeyPostScoreZSet), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: postID,
	})

	// 3. 执行 Pipeline
	_, err := pipeline.Exec(ctx)
	return err
}

// GetPostVoteData 获取帖子的投票数据
// 返回: 赞成票数, 反对票数, error
func GetPostVoteData(postID string) (upVotes, downVotes int64, err error) {
	// 获取所有对该帖子投过票的用户及其投票值
	// ZRANGE key 0 -1 WITHSCORES
	data, err := rdb.ZRangeWithScores(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), 0, -1).Result()
	if err != nil {
		return 0, 0, err
	}

	// 统计赞成票和反对票
	for _, z := range data {
		if z.Score > 0 {
			upVotes++
		} else if z.Score < 0 {
			downVotes++
		}
	}

	return upVotes, downVotes, nil
}

// GetPostsVoteData 批量获取多个帖子的投票数（赞成票数）
// 使用 Redis Pipeline 提高性能
// 参数: ids - 帖子ID列表（字符串格式）
// 返回: []int64 - 每个帖子的赞成票数，顺序与 ids 一致
func GetPostsVoteData(ids []string) (data []int64, err error) {
	// 使用 Pipeline 减少 RTT (Round Trip Time)
	pipeline := rdb.Pipeline()

	// 1. 组装 Pipeline 命令
	for _, id := range ids {
		key := getRedisKey(KeyPostVotedZSetPrefix + id) // 拼接Key: bluebell:post:voted:{id}
		// ZCount 计算分数在 [1, 1] 之间的数量，即赞成票的数量
		pipeline.ZCount(ctx, key, "1", "1")
	}

	// 2. 执行 Pipeline
	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 获取结果
	data = make([]int64, 0, len(cmders))
	for _, cmder := range cmders {
		// 类型断言，从 cmder 中拿到 IntCmd 的结果
		v := cmder.(*redis.IntCmd).Val()
		data = append(data, v)
	}
	return
}

// GetCommunityPostIDsInOrder 按照指定顺序获取指定社区的帖子ID列表
// communityID: 社区ID
// orderKey: "time" 或 "score"
// page: 页码(从1开始)
// size: 每页数量
func GetCommunityPostIDsInOrder(communityID int64, orderKey string, page, size int64) ([]string, error) {
	// TODO: 实现根据社区ID获取帖子ID列表的逻辑
	// 这里需要先确定如何在Redis中存储社区与帖子的关系
	// 可能需要一个新的Key结构来存储每个社区的帖子列表
	return nil, nil
}

// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
// orderKey: "time" 或 "score"
// page: 页码(从1开始)
// size: 每页数量
func GetPostIDsInOrder(orderKey string, page, size int64) ([]string, error) {
	// 1. 确定查询的 Redis Key
	key := getRedisKey(KeyPostTimeZSet)
	if orderKey == "score" {
		key = getRedisKey(KeyPostScoreZSet)
	}

	// 2. 计算分页的起始和结束位置
	// Redis ZSet 的索引从0开始
	start := (page - 1) * size
	end := start + size - 1

	// 3. 按分数从大到小查询 (ZREVRANGE)
	// 返回的是帖子ID列表
	return rdb.ZRevRange(ctx, key, start, end).Result()
}

// GetPostScore 获取帖子的当前分数
func GetPostScore(postID string) (float64, error) {
	return rdb.ZScore(ctx, getRedisKey(KeyPostScoreZSet), postID).Result()
}

// GetPostVoteStatus 获取用户对某个帖子的投票状态
// 返回: 1(赞成), -1(反对), 0(未投票或取消投票)
func GetPostVoteStatus(userID, postID string) (int8, error) {
	score, err := rdb.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID).Result()
	if err != nil {
		// 如果 Redis 返回 Nil, 说明用户没有投票记录
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}

	// 将 float64 的 score 转换为 int8
	return int8(score), nil
}

// BatchGetPostVoteStatus 批量获取用户对多个帖子的投票状态
func BatchGetPostVoteStatus(userID string, postIDs []string) (map[string]int8, error) {
	// 使用 Pipeline 提高性能
	pipeline := rdb.Pipeline()

	// 为每个帖子创建一个 ZScore 命令
	cmds := make([]*redis.FloatCmd, 0, len(postIDs))
	for _, postID := range postIDs {
		cmds = append(cmds, pipeline.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID))
	}

	// 执行 Pipeline
	_, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	// 构建结果 map
	result := make(map[string]int8, len(postIDs))
	for i, cmd := range cmds {
		score, err := cmd.Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				result[postIDs[i]] = 0 // 未投票
			} else {
				return nil, err
			}
		} else {
			result[postIDs[i]] = int8(score)
		}
	}

	return result, nil
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
