package postsvc

import (
	// 领域层 - Repository 接口
	"bluebell/internal/domain/cachedomain"
	"bluebell/internal/domain/dbdomain"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO
	postreq "bluebell/internal/dto/request/post"
	postResp "bluebell/internal/dto/response/post"

	// 基础设施
	"bluebell/internal/infrastructure/snowflake"

	// 模型
	"bluebell/internal/model"

	// 错误处理
	"bluebell/pkg/errorx"

	"context"
	"strconv"

	"go.uber.org/zap"
)

// postServiceStruct 帖子业务逻辑服务
type postServiceStruct struct {
	postRepo   dbdomain.PostRepository
	postCache  cachedomain.PostRepository
	voteRepo   dbdomain.VoteRepository
	remarkRepo dbdomain.RemarkRepository
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postRepo dbdomain.PostRepository,
	postCache cachedomain.PostRepository,
	voteRepo dbdomain.VoteRepository,
	remarkRepo dbdomain.RemarkRepository,
) svcdomain.PostService {
	return &postServiceStruct{
		postRepo:   postRepo,
		postCache:  postCache,
		voteRepo:   voteRepo,
		remarkRepo: remarkRepo,
	}
}

// CreatePost 创建帖子
func (s *postServiceStruct) CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) error {
	postID := snowflake.GenID()

	post := &model.Post{
		PostID:   strconv.FormatInt(postID, 10),
		AuthorID: authorID,

		CommunityID: p.CommunityID,
		PostTitle:   p.Title,
		Content:     p.Content,
		Status:      1,
	}

	err := s.postRepo.CreatePost(ctx, post)
	if err != nil {
		zap.L().Error("postRepo.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	// 同步到 Redis
	err = s.postCache.CreatePost(ctx, postID, p.CommunityID)
	if err != nil {
		zap.L().Error("postCache.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
	}

	return nil
}

// GetPostByID 查询单个帖子详情
func (s *postServiceStruct) GetPostByID(ctx context.Context, pid int64) (data *postResp.DetailResponse, err error) {
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

	data = &postResp.DetailResponse{
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
func (s *postServiceStruct) GetPostList(ctx context.Context, p *postreq.PostListRequest) (data []*postResp.DetailResponse, err error) {
	ids, err := s.postCache.GetPostIDsInOrder(ctx, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("postCache.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if len(ids) == 0 {
		zap.L().Warn("postCache.GetPostIDsInOrder() return 0 data")
		data = make([]*postResp.DetailResponse, 0)
		return
	}

	zap.L().Debug("GetPostList", zap.Any("ids", ids))

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		zap.L().Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	zap.L().Debug("GetPostListByIDsWithPreload", zap.Any("posts", posts))

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("postCache.GetPostsVoteData failed", zap.Error(err))
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

		postDetail := &postResp.DetailResponse{
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
func (s *postServiceStruct) GetCommunityPostList(ctx context.Context, p *postreq.PostListRequest) (data []*postResp.DetailResponse, err error) {
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
		data = make([]*postResp.DetailResponse, 0)
		return data, nil
	}

	zap.L().Debug("GetCommunityPostList", zap.Any("ids", ids))

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		zap.L().Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("postCache.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	data = make([]*postResp.DetailResponse, 0, len(posts))
	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.UserName
		} else {
			zap.L().Error("author not preloaded for post",
				zap.String("post_id", post.PostID),
				zap.Int64("author_id", post.AuthorID))
		}

		postDetail := &postResp.DetailResponse{
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

// VoteForPost 投票业务逻辑
func (s *postServiceStruct) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
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

	// Redis 模式：先写 Redis，再同步 MySQL
	err = s.postCache.VoteForPost(
		ctx,
		strconv.FormatInt(userID, 10),
		strconv.FormatInt(p.PostID, 10),
		strconv.FormatInt(post.CommunityID, 10),
		float64(p.Direction),
	)
	if err != nil {
		code := errorx.GetCode(err)
		if code == errorx.CodeVoteTimeExpire || code == errorx.CodeVoteRepeated {
			return err
		}

		return errorx.ErrServerBusy
	}

	// 同步落盘到 MySQL
	// 修复：移除异步 goroutine 避免高并发下 goroutine 泄漏
	err = s.voteRepo.SaveVote(ctx, userID, p.PostID, p.Direction)
	if err != nil {
		zap.L().Error("save vote to mysql failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", p.PostID),
			zap.Error(err),
		)
		return errorx.ErrServerBusy
	}

	return nil
}
func (s *postServiceStruct) RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) error {
	// 1. 校验帖子是否存在
	post, err := s.postRepo.GetPostByID(ctx, req.PostID)
	if err != nil {
		zap.L().Error("remarkPost: postRepo.GetPostByID failed",
			zap.Int64("post_id", req.PostID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}
	if post == nil || post.PostID == "" {
		return errorx.ErrNotFound
	}

	// 2. 构建评论模型
	remark := &model.Remark{
		PostID:   req.PostID,
		Content:  req.Content,
		AuthorID: userID,
	}

	// 3. 保存到数据库
	if err := s.remarkRepo.CreateRemark(ctx, remark); err != nil {
		zap.L().Error("remarkPost: remarkRepo.CreateRemark failed",
			zap.Int64("post_id", req.PostID),
			zap.Int64("author_id", userID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}

// GetPostRemarks 获取帖子评论列表
func (s *postServiceStruct) GetPostRemarks(ctx context.Context, postID int64) ([]*postResp.RemarkDetail, error) {
	// 1. 获取原始评论列表
	remarks, err := s.remarkRepo.GetRemarksByPostID(ctx, postID)
	if err != nil {
		zap.L().Error("getPostRemarks: remarkRepo.GetRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 2. 转换为 DTO
	resp := make([]*postResp.RemarkDetail, 0, len(remarks))
	for _, r := range remarks {
		authorName := "已注销用户"
		if r.Author != nil {
			authorName = r.Author.UserName
		}
		resp = append(resp, &postResp.RemarkDetail{
			ID:         r.ID,
			Content:    r.Content,
			AuthorName: authorName,
			CreateTime: r.CreatedAt,
		})
	}

	return resp, nil
}
