package post

import (
	"bluebell/internal/domain/repointerface"
	"bluebell/internal/dto/request"
	"bluebell/internal/dto/response"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"context"

	"strconv"

	"go.uber.org/zap"
)

// postServiceStruct 帖子业务逻辑服务
type postServiceStruct struct {
	postRepo  repointerface.PostRepository
	postCache repointerface.PostCacheRepository
	voteCache repointerface.VoteCacheRepository
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postRepo repointerface.PostRepository,
	postCache repointerface.PostCacheRepository,
	voteCache repointerface.VoteCacheRepository,
) *postServiceStruct {
	return &postServiceStruct{
		postRepo:  postRepo,
		postCache: postCache,
		voteCache: voteCache,
	}
}

// CreatePost 创建帖子,返回新创建的帖子ID
func (s *postServiceStruct) CreatePost(ctx context.Context, p *request.CreatePostRequest, authorID int64) (postID int64, err error) {
	postID = snowflake.GenID()

	post := &model.Post{
		PostID:   strconv.FormatInt(postID, 10),
		AuthorID: authorID,

		CommunityID: p.CommunityID,
		PostTitle:   p.Title,
		Content:     p.Content,
		Status:      1,
	}

	err = s.postRepo.CreatePost(ctx, post)
	if err != nil {
		zap.L().Error("postRepo.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return 0, errorx.ErrServerBusy
	}

	// 同步到 Redis
	err = s.postCache.CreatePost(ctx, postID, p.CommunityID)
	if err != nil {
		zap.L().Error("postCache.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
	}

	return postID, nil
}

// GetPostByID 查询单个帖子详情
func (s *postServiceStruct) GetPostByID(ctx context.Context, pid int64) (data *response.PostDetailResponse, err error) {
	post, err := s.postRepo.GetPostByID(ctx, pid)
	if err != nil {
		zap.L().Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", pid),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if post == nil || post.PostID == "" {
		return nil, errorx.ErrNotFound
	}

	if post.Author == nil || post.Author.UserID == 0 {
		zap.L().Warn("author not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("author_id", post.AuthorID))
		return nil, errorx.ErrNotFound
	}

	if post.Community == nil || post.Community.ID == 0 {
		zap.L().Warn("community not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("community_id", post.CommunityID))
		return nil, errorx.ErrNotFound
	}

	data = &response.PostDetailResponse{
		ID:          post.PostID,
		AuthorID:    strconv.FormatInt(post.AuthorID, 10),
		CommunityID: post.CommunityID,
		Status:      post.Status,
		Title:       post.PostTitle,
		Content:     post.Content,
		CreateTime:  post.CreatedAt,
		AuthorName:  post.Author.UserName,
	}

	return data, nil
}

// GetPostList 获取帖子列表
func (s *postServiceStruct) GetPostList(ctx context.Context, p *request.PostListRequest) (data []*response.PostDetailResponse, err error) {
	ids, err := s.postCache.GetPostIDsInOrder(ctx, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("postCache.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if len(ids) == 0 {
		zap.L().Warn("postCache.GetPostIDsInOrder() return 0 data")
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

	voteData, err := s.voteCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("voteCache.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.UserName
		} else {
			zap.L().Error("author not preloaded for post",
				zap.String("post_id", post.PostID),
				zap.Int64("author_id", post.AuthorID))
		}

		postDetail := &response.PostDetailResponse{
			ID:          post.PostID,
			AuthorID:    strconv.FormatInt(post.AuthorID, 10),
			CommunityID: post.CommunityID,
			Status:      post.Status,
			Title:       post.PostTitle,
			Content:     post.Content,
			CreateTime:  post.CreatedAt,
			AuthorName:  authorName,
			VoteNum:     voteData[idx],
		}
		data = append(data, postDetail)
	}

	return
}

// GetCommunityPostList 根据社区ID获取帖子列表
func (s *postServiceStruct) GetCommunityPostList(ctx context.Context, p *request.PostListRequest) (data []*response.PostDetailResponse, err error) {
	ids, err := s.postCache.GetCommunityPostIDsInOrder(ctx, p.CommunityID, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("postCache.GetCommunityPostIDsInOrder failed",
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

	voteData, err := s.voteCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("voteCache.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	data = make([]*response.PostDetailResponse, 0, len(posts))
	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.UserName
		} else {
			zap.L().Error("author not preloaded for post",
				zap.String("post_id", post.PostID),
				zap.Int64("author_id", post.AuthorID))
		}

		postDetail := &response.PostDetailResponse{
			ID:          post.PostID,
			AuthorID:    strconv.FormatInt(post.AuthorID, 10),
			CommunityID: post.CommunityID,
			Status:      post.Status,
			Title:       post.PostTitle,
			Content:     post.Content,
			CreateTime:  post.CreatedAt,
			AuthorName:  authorName,
			VoteNum:     voteData[idx],
		}
		data = append(data, postDetail)
	}

	return data, nil
}

// DeletePost 删除帖子（软删除）
func (s *postServiceStruct) DeletePost(ctx context.Context, postID int64, userID int64) error {
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
