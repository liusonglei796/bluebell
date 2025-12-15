package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/models"
	"strconv"

	"go.uber.org/zap"
)

// VoteForPost 投票业务逻辑
// 参数:
//   userID: 投票用户ID
//   p: 投票参数(包含帖子ID和投票方向)
//
// 业务规则:
//   1. 检查帖子发布时间,超过一周不允许投票
//   2. 根据用户之前的投票状态和当前操作,计算分数变化
//   3. 更新 Redis 中的帖子分数和用户投票记录
func VoteForPost(userID int64, p *models.ParamVoteData) error {
	// 记录投票操作日志
	zap.L().Debug("VoteForPost",
		zap.Int64("userID", userID),
		zap.Int64("postID", p.PostID),
		zap.Int8("direction", p.Direction))

	// 1. 先查询帖子获取社区ID
	// 投票时需要同时更新社区维度的分数ZSet
	post, err := mysql.GetPostByID(p.PostID)
	if err != nil {
		zap.L().Error("mysql.GetPostByID failed",
			zap.Int64("post_id", p.PostID),
			zap.Error(err))
		return err
	}
	if post == nil {
		zap.L().Error("post not found", zap.Int64("post_id", p.PostID))
		return mysql.ErrorInvalidID
	}

	// 2. 调用 Redis 层执行投票逻辑
	// 将 int64 类型的 ID 转换为 string (Redis 中统一使用 string)
	// 将 int8 类型的 direction 转换为 float64 (Redis ZSet 的 score 是 float64)
	return redis.VoteForPost(
		strconv.FormatInt(userID, 10),
		strconv.FormatInt(p.PostID, 10),
		strconv.FormatInt(post.CommunityID, 10), // 传递社区ID
		float64(p.Direction),
	)
}
