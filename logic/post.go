package logic

import (
	"bluebell/dao/redis"
	"bluebell/dto/request"
	"bluebell/dto/response"
	"bluebell/models"
	"bluebell/pkg/errorx"
	"bluebell/pkg/snowflake"
	"context"

	"go.uber.org/zap"
)

// PostService 帖子业务逻辑服务
type PostService struct {
	postRepo PostRepository
}

// NewPostService 创建帖子服务实例
func NewPostService(postRepo PostRepository) *PostService {
	return &PostService{postRepo: postRepo}
}

// CreatePost 创建帖子,返回新创建的帖子ID
func (s *PostService) CreatePost(ctx context.Context, p *request.CreatePostRequest) (postID int64, err error) {
	// 1. 生成帖子ID
	postID = snowflake.GenID()

	// 2. 构造Post结构体
	post := &models.Post{
		ID:          postID,
		AuthorID:    p.AuthorID,
		CommunityID: p.CommunityID,
		Title:       p.Title,
		Content:     p.Content,
		Status:      1,
	}

	// 3. 保存到数据库
	err = s.postRepo.CreatePost(ctx, post)
	if err != nil {
		zap.L().Error("postRepo.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return 0, errorx.ErrServerBusy
	}

	// 4. 同步到 Redis
	err = redis.CreatePost(ctx, postID, p.CommunityID)
	if err != nil {
		zap.L().Error("redis.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
	}

	return postID, nil
}

// GetPostByID 查询单个帖子详情
func (s *PostService) GetPostByID(ctx context.Context, pid int64) (data *response.PostDetailResponse, err error) {
	post, err := s.postRepo.GetPostByID(ctx, pid)
	if err != nil {
		zap.L().Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", pid),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if post == nil || post.ID == 0 {
		return nil, errorx.ErrNotFound
	}

	if post.Author == nil || post.Author.UserID == 0 {
		zap.L().Warn("author not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("author_id", post.AuthorID))
		return nil, errorx.ErrNotFound
	}

	if post.Community == nil || post.Community.CommunityID == 0 {
		zap.L().Warn("community not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("community_id", post.CommunityID))
		return nil, errorx.ErrNotFound
	}

	data = &response.PostDetailResponse{
		Post:       post,
		AuthorName: post.Author.Username,
	}

	return data, nil
}

// GetPostList 获取帖子列表
func (s *PostService) GetPostList(ctx context.Context, p *request.PostListRequest) (data []*response.PostDetailResponse, err error) {
	ids, err := redis.GetPostIDsInOrder(ctx, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("redis.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if len(ids) == 0 {
		zap.L().Warn("redis.GetPostIDsInOrder() return 0 data")
		data = make([]*response.PostDetailResponse, 0)
		return
	}

	zap.L().Debug("GetPostList", zap.Any("ids", ids))

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		zap.L().Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	zap.L().Debug("GetPostListByIDsWithPreload", zap.Any("posts", posts))

	voteData, err := redis.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("redis.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.Username
		} else {
			zap.L().Error("author not preloaded for post",
				zap.Int64("post_id", post.ID),
				zap.Int64("author_id", post.AuthorID))
		}

		postDetail := &response.PostDetailResponse{
			AuthorName: authorName,
			Post:       post,
			VoteNum:    voteData[idx],
		}
		data = append(data, postDetail)
	}

	return
}

// GetCommunityPostList 根据社区ID获取帖子列表
func (s *PostService) GetCommunityPostList(ctx context.Context, p *request.PostListRequest) (data []*response.PostDetailResponse, err error) {
	ids, err := redis.GetCommunityPostIDsInOrder(ctx, p.CommunityID, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("redis.GetCommunityPostIDsInOrder failed",
			zap.Int64("community_id", p.CommunityID),
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if len(ids) == 0 {
		zap.L().Info("GetCommunityPostList: no posts found",
			zap.Int64("community_id", p.CommunityID))
		data = make([]*response.PostDetailResponse, 0)
		return data, nil
	}

	zap.L().Debug("GetCommunityPostList", zap.Any("ids", ids))

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		zap.L().Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	voteData, err := redis.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("redis.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	data = make([]*response.PostDetailResponse, 0, len(posts))
	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.Username
		} else {
			zap.L().Error("author not preloaded for post",
				zap.Int64("post_id", post.ID),
				zap.Int64("author_id", post.AuthorID))
		}

		postDetail := &response.PostDetailResponse{
			AuthorName: authorName,
			Post:       post,
			VoteNum:    voteData[idx],
		}
		data = append(data, postDetail)
	}

	return data, nil
}

// DeletePost 删除帖子（软删除）
func (s *PostService) DeletePost(ctx context.Context, postID, userID int64) error {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		zap.L().Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}
	if post == nil {
		return errorx.ErrNotFound
	}

	if post.AuthorID != userID {
		return errorx.ErrForbidden
	}

	err = s.postRepo.DeletePostByAuthor(ctx, postID, userID)
	if err != nil {
		zap.L().Error("postRepo.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}
