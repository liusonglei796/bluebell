package postsvc

import (
	// 领域层 - Repository 接口
	"bluebell/internal/domain"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO
	postreq "bluebell/internal/interfaces/http/dto/request/post"
	postResp "bluebell/internal/interfaces/http/dto/response/post"

	// 基础设施
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/infrastructure/mq"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"strconv"

	"go.uber.org/zap"

	// 日志
	"bluebell/internal/infrastructure/logger"
)

// postServiceStruct 帖子业务逻辑服务
type postServiceStruct struct {
	postRepo   domain.PostRepository
	postCache  domain.PostCacheRepository
	voteRepo   domain.VoteRepository
	remarkRepo domain.RemarkRepository
	publisher  *mq.Publisher
	esClient   *es.Client
	voteBuffer *mq.VoteBuffer
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postRepo domain.PostRepository,
	postCache domain.PostCacheRepository,
	voteRepo domain.VoteRepository,
	remarkRepo domain.RemarkRepository,
	publisher *mq.Publisher,
	esClient *es.Client,
	voteBuffer *mq.VoteBuffer,
) application.PostService {
	return &postServiceStruct{
		postRepo:   postRepo,
		postCache:  postCache,
		voteRepo:   voteRepo,
		remarkRepo: remarkRepo,
		publisher:  publisher,
		esClient:   esClient,
		voteBuffer: voteBuffer,
	}
}

// CreatePost 创建帖子
func (s *postServiceStruct) CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (postID string, err error) {
	postIDInt := snowflake.GenID()
	postID = strconv.FormatInt(postIDInt, 10)

	post := &entity.Post{
		PostID:      postID,
		AuthorID:    authorID,
		CommunityID: p.CommunityID,
		PostTitle:   p.Title,
		Content:     p.Content,
		Status:      entity.PostStatusPublished,
	}

	if !post.IsValid() {
		return "", entity.ErrInvalidParam
	}

	err = s.postRepo.CreatePost(ctx, post)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
		return "", entity.Wrap(entity.ErrServerBusy, err)
	}

	// 同步到 Redis
	err = s.postCache.CreatePost(ctx, postIDInt, p.CommunityID)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
	}

	return postID, nil
}

// GetPostByID 查询单个帖子详情
func (s *postServiceStruct) GetPostByID(ctx context.Context, pid int64) (data *postResp.DetailResponse, err error) {
	post, err := s.postRepo.GetPostByID(ctx, pid)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", pid),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	if post == nil || post.PostID == "" {
		return nil, entity.ErrNotFound
	}

	if post.Author == nil || post.Author.UserID == 0 {
		logger.WithContext(ctx).Warn("author not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("author_id", post.AuthorID))
		return nil, entity.ErrNotFound
	}

	if post.Community == nil || post.Community.ID == 0 {
		logger.WithContext(ctx).Warn("community not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("community_id", post.CommunityID))
		return nil, entity.ErrNotFound
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
		logger.WithContext(ctx).Error("postCache.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	if len(ids) == 0 {
		logger.WithContext(ctx).Warn("postCache.GetPostIDsInOrder() return 0 data")
		data = make([]*postResp.DetailResponse, 0)
		return
	}

	logger.WithContext(ctx).Debug("GetPostList", zap.Any("ids", ids))

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	logger.WithContext(ctx).Debug("GetPostListByIDsWithPreload", zap.Any("posts", posts))

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.GetPostsVoteData failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.UserName
		} else {
			logger.WithContext(ctx).Error("author not preloaded for post",
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
		logger.WithContext(ctx).Error("postCache.GetCommunityPostIDsInOrder failed",
			zap.Int64("community_id", p.CommunityID),
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	if len(ids) == 0 {
		logger.WithContext(ctx).Info("GetCommunityPostList: no posts found",
			zap.Int64("community_id", p.CommunityID))
		data = make([]*postResp.DetailResponse, 0)
		return data, nil
	}

	logger.WithContext(ctx).Debug("GetCommunityPostList", zap.Any("ids", ids))

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.GetPostsVoteData failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	data = make([]*postResp.DetailResponse, 0, len(posts))
	for idx, post := range posts {
		var authorName string
		if post.Author != nil {
			authorName = post.Author.UserName
		} else {
			logger.WithContext(ctx).Error("author not preloaded for post",
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

// DeletePost 删除帖子及其评论（级联软删除）
func (s *postServiceStruct) DeletePost(ctx context.Context, postID int64, userID int64) error {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}
	if !post.IsValid() {
		return entity.ErrNotFound
	}

	// 权限校验 (下沉到领域层)
	if err := post.CanBeDeletedBy(userID); err != nil {
		return err
	}

	// 1. 删除该帖子的所有评论
	if err := s.remarkRepo.DeleteRemarksByPostID(ctx, postID); err != nil {
		logger.WithContext(ctx).Error("remarkRepo.DeleteRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	// 2. 软删除帖子 (status = 0)
	err = s.postRepo.DeletePostByAuthor(ctx, postID, userID)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	// 3. 删除 ES 中的帖子文档
	if s.esClient != nil {
		postIDStr := strconv.FormatInt(postID, 10)
		if err := s.esClient.DeleteDocument(ctx, es.IndexPost, postIDStr); err != nil {
			logger.WithContext(ctx).Error("esClient.DeleteDocument failed",
				zap.Int64("post_id", postID),
				zap.Error(err))
			// ES 删除失败不影响主流程，仅记录日志
		}
	}

	// 清理 Redis 缓存
	if err := s.postCache.DeletePost(ctx, postID, post.CommunityID); err != nil {
		logger.WithContext(ctx).Error("postCache.DeletePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		// 缓存清理失败不影响主流程，仅记录日志
	}

	return nil
}

// VoteForPost 投票业务逻辑 (Architecture E: Local Buffer + Batch Redis Pipeline + MQ 持久化)
//
//	请求 → 领域校验 → 加入本地缓冲区 (VoteBuffer) → 返回
//	VoteBuffer (100ms) → Batch Redis Pipeline (ZAdd+HSet+Score) → 发 MQ → 消费者持久化
func (s *postServiceStruct) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
	// 1. 领域校验
	vote := &entity.Vote{
		PostID:    p.PostID,
		UserID:    userID,
		Direction: p.Direction,
	}
	if err := vote.Validate(); err != nil {
		return err
	}

	postIDStr := strconv.FormatInt(p.PostID, 10)
	userIDStr := strconv.FormatInt(userID, 10)

	// 2. 加入本地缓冲区 (聚合后批量写入 Redis 和 MQ)
	s.voteBuffer.AddVote(postIDStr, userIDStr, p.Direction)

	return nil
}

func (s *postServiceStruct) RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (remarkID uint, err error) {
	// 1. 校验帖子是否存在
	post, err := s.postRepo.GetPostByID(ctx, req.PostID)
	if err != nil {
		logger.WithContext(ctx).Error("remarkPost: postRepo.GetPostByID failed",
			zap.Int64("post_id", req.PostID),
			zap.Error(err))
		return 0, entity.Wrap(entity.ErrServerBusy, err)
	}
	if !post.IsValid() {
		return 0, entity.ErrNotFound
	}

	// 2. 构建评论领域实体
	remark := &entity.Remark{
		PostID:   req.PostID,
		Content:  req.Content,
		AuthorID: userID,
	}
	if err := remark.Validate(); err != nil {
		return 0, err
	}

	// 3. 保存到数据库
	if err := s.remarkRepo.CreateRemark(ctx, remark); err != nil {
		logger.WithContext(ctx).Error("remarkPost: remarkRepo.CreateRemark failed",
			zap.Int64("post_id", req.PostID),
			zap.Int64("author_id", userID),
			zap.Error(err))
		return 0, entity.Wrap(entity.ErrServerBusy, err)
	}

	return remark.ID, nil
}

// GetPostRemarks 获取帖子评论列表
func (s *postServiceStruct) GetPostRemarks(ctx context.Context, postID int64) ([]*postResp.RemarkDetail, error) {
	// 1. 获取原始评论列表
	remarks, err := s.remarkRepo.GetRemarksByPostID(ctx, postID)
	if err != nil {
		logger.WithContext(ctx).Error("getPostRemarks: remarkRepo.GetRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
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

// SearchPosts 全文搜索帖子
func (s *postServiceStruct) SearchPosts(ctx context.Context, keyword string, page, pageSize int) (*es.SearchResponse, error) {
	if s.esClient == nil {
		logger.WithContext(ctx).Warn("esClient is not initialized")
		return &es.SearchResponse{Posts: []es.SearchPostDoc{}}, nil
	}

	esReq := &es.SearchRequest{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}

	return s.esClient.Search(ctx, esReq)
}
