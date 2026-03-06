package vote

import (
	"bluebell/internal/domain/repointerface"
	"bluebell/internal/dto/request"
	"bluebell/pkg/errorx"
	"context"
	"errors"
	"strconv"

	"go.uber.org/zap"
)

// voteServiceStruct 投票业务逻辑服务
type voteServiceStruct struct {
	postRepo  repointerface.PostRepository
	voteCache repointerface.VoteCacheRepository
}

// NewVoteService 创建投票服务实例
func NewVoteService(postRepo repointerface.PostRepository, voteCache repointerface.VoteCacheRepository) *voteServiceStruct {
	return &voteServiceStruct{
		postRepo:  postRepo,
		voteCache: voteCache,
	}
}

// VoteForPost 投票业务逻辑
func (s *voteServiceStruct) VoteForPost(ctx context.Context, userID int64, p *request.VoteRequest) error {
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

	err = s.voteCache.VoteForPost(
		ctx,
		strconv.FormatInt(userID, 10),
		strconv.FormatInt(p.PostID, 10),
		strconv.FormatInt(post.CommunityID, 10),
		float64(p.Direction),
	)

	if err != nil {
		if errors.Is(err, repointerface.ErrVoteTimeExpire) {
			return errorx.ErrVoteTimeExpire
		}
		if errors.Is(err, repointerface.ErrVoteRepeated) {
			return errorx.ErrVoteRepeated
		}

		zap.L().Error("voteCache.VoteForPost failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", p.PostID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}
