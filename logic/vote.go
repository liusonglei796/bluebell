package logic

import (
	"bluebell/dao/redis"
	"bluebell/dto/request"
	"bluebell/pkg/errorx"
	"context"
	"errors"
	"strconv"

	"go.uber.org/zap"
)

// VoteService 投票业务逻辑服务
type VoteService struct {
	postRepo PostRepository
}

// NewVoteService 创建投票服务实例
func NewVoteService(postRepo PostRepository) *VoteService {
	return &VoteService{postRepo: postRepo}
}

// VoteForPost 投票业务逻辑
func (s *VoteService) VoteForPost(ctx context.Context, userID int64, p *request.VoteRequest) error {
	zap.L().Debug("VoteForPost",
		zap.Int64("userID", userID),
		zap.Int64("postID", p.PostID),
		zap.Int8("direction", p.Direction))

	post, err := s.postRepo.GetPostByID(ctx, p.PostID)
	if err != nil {
		zap.L().Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", p.PostID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}
	if post == nil {
		return errorx.ErrNotFound
	}

	err = redis.VoteForPost(
		ctx,
		strconv.FormatInt(userID, 10),
		strconv.FormatInt(p.PostID, 10),
		strconv.FormatInt(post.CommunityID, 10),
		float64(p.Direction),
	)

	if err != nil {
		if errors.Is(err, redis.ErrVoteTimeExpire) {
			return errorx.ErrVoteTimeExpire
		}
		if errors.Is(err, redis.ErrVoteRepeated) {
			return errorx.ErrVoteRepeated
		}

		zap.L().Error("redis.VoteForPost failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", p.PostID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}
