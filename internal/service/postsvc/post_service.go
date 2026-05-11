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
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/service/mq"

	// 模型
	"bluebell/internal/model"

	// 错误处理
	"bluebell/pkg/errorx"

	"context"
	"strconv"

	"bluebell/internal/middleware"

	"go.opentelemetry.io/otel/attribute"

	"go.uber.org/zap"
)

// postServiceStruct 帖子业务逻辑服务
type postServiceStruct struct {
	postRepo   dbdomain.PostRepository
	postCache  cachedomain.PostRepository
	voteRepo   dbdomain.VoteRepository
	remarkRepo dbdomain.RemarkRepository
	publisher  *mq.MQPublisher
	esClient   *es.Client
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postRepo dbdomain.PostRepository,
	postCache cachedomain.PostRepository,
	voteRepo dbdomain.VoteRepository,
	remarkRepo dbdomain.RemarkRepository,
	publisher *mq.MQPublisher,
	esClient *es.Client,
) svcdomain.PostService {
	return &postServiceStruct{
		postRepo:   postRepo,
		postCache:  postCache,
		voteRepo:   voteRepo,
		remarkRepo: remarkRepo,
		publisher:  publisher,
		esClient:   esClient,
	}
}

// CreatePost 创建帖子 (同步审核版)
func (s *postServiceStruct) CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (postID string, err error) {
	ctx, span := middleware.StartSpanFromContext(ctx, "CreatePost",
		attribute.Int64("user.id", authorID),
		attribute.Int64("community.id", p.CommunityID),
	)
	defer span.End()

	postIDInt := snowflake.GenID()
	postID = strconv.FormatInt(postIDInt, 10)

	post := &model.Post{
		PostID:   postID,
		AuthorID: authorID,

		CommunityID: p.CommunityID,
		PostTitle:   p.Title,
		Content:     p.Content,
		Status:      1,
	}

	err = s.postRepo.CreatePost(ctx, post)
	if err != nil {
		zap.L().Error("postRepo.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
		return "", errorx.ErrServerBusy
	}

	// 同步到 Redis
	err = s.postCache.CreatePost(ctx, postIDInt, p.CommunityID)
	if err != nil {
		zap.L().Error("postCache.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
	}

	return postID, nil
}

// GetPostByID 查询单个帖子详情
func (s *postServiceStruct) GetPostByID(ctx context.Context, pid int64) (data *postResp.DetailResponse, err error) {
	ctx, span := middleware.StartSpanFromContext(ctx, "GetPostByID",
		attribute.Int64("post.id", pid),
	)
	defer span.End()

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
	ctx, span := middleware.StartSpanFromContext(ctx, "GetPostList",
		attribute.Int64("page", p.Page),
		attribute.Int64("size", p.Size),
		attribute.String("order", p.Order),
	)
	defer span.End()

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
	ctx, span := middleware.StartSpanFromContext(ctx, "GetCommunityPostList",
		attribute.Int64("community.id", p.CommunityID),
		attribute.Int64("page", p.Page),
		attribute.Int64("size", p.Size),
		attribute.String("order", p.Order),
	)
	defer span.End()

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

// DeletePost 删除帖子及其评论（级联软删除）
func (s *postServiceStruct) DeletePost(ctx context.Context, postID int64, userID int64) error {
	ctx, span := middleware.StartSpanFromContext(ctx, "DeletePost",
		attribute.Int64("post.id", postID),
		attribute.Int64("user.id", userID),
	)
	defer span.End()

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

	// 1. 删除该帖子的所有评论
	if err := s.remarkRepo.DeleteRemarksByPostID(ctx, postID); err != nil {
		zap.L().Error("remarkRepo.DeleteRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	// 2. 软删除帖子 (status = 0)
	err = s.postRepo.DeletePostByAuthor(ctx, postID, userID)
	if err != nil {
		zap.L().Error("postRepo.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	// 3. 删除 ES 中的帖子文档
	if s.esClient != nil {
		postIDStr := strconv.FormatInt(postID, 10)
		if err := s.esClient.DeleteDocument(ctx, es.IndexPost, postIDStr); err != nil {
			zap.L().Error("esClient.DeleteDocument failed",
				zap.Int64("post_id", postID),
				zap.Error(err))
			// ES 删除失败不影响主流程，仅记录日志
		}
	}

	// 清理 Redis 缓存
	if err := s.postCache.DeletePost(ctx, postID, post.CommunityID); err != nil {
		zap.L().Error("postCache.DeletePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		// 缓存清理失败不影响主流程，仅记录日志
	}

	return nil
}

// VoteForPost 投票业务逻辑 (Situation C: Full Optimization)
func (s *postServiceStruct) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
	ctx, span := middleware.StartSpanFromContext(ctx, "VoteForPost",
		attribute.Int64("user.id", userID),
	)
	defer span.End()

	// 1. 优先从 Redis 检查帖子是否存在
	exists, _ := s.postCache.CheckPostExists(ctx, p.PostID)
	if !exists {
		post, err := s.postRepo.GetPostByID(ctx, p.PostID)
		if err != nil || post == nil {
			return errorx.ErrNotFound
		}
	}

	// 2. 异步入队 (MQ)
	postIDStr := strconv.FormatInt(p.PostID, 10)
	userIDStr := strconv.FormatInt(userID, 10)

	if s.publisher != nil {
		msg := &mq.VoteMessage{
			MsgID:  strconv.FormatInt(snowflake.GenID(), 10), // 生成全局唯一消息ID
			PostID: postIDStr,
			UserID: userIDStr,
			Action: int(p.Direction),
		}
		_ = s.publisher.PublishVote(ctx, msg)
	}

	return nil
}

func (s *postServiceStruct) RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (remarkID uint, err error) {
	ctx, span := middleware.StartSpanFromContext(ctx, "RemarkPost",
		attribute.Int64("post.id", req.PostID),
		attribute.Int64("user.id", userID),
	)
	defer span.End()

	// 1. 校验帖子是否存在
	post, err := s.postRepo.GetPostByID(ctx, req.PostID)
	if err != nil {
		zap.L().Error("remarkPost: postRepo.GetPostByID failed",
			zap.Int64("post_id", req.PostID),
			zap.Error(err))
		return 0, errorx.ErrServerBusy
	}
	if post == nil || post.PostID == "" {
		return 0, errorx.ErrNotFound
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
		return 0, errorx.ErrServerBusy
	}

	return remark.ID, nil
}

// GetPostRemarks 获取帖子评论列表
func (s *postServiceStruct) GetPostRemarks(ctx context.Context, postID int64) ([]*postResp.RemarkDetail, error) {
	ctx, span := middleware.StartSpanFromContext(ctx, "GetPostRemarks",
		attribute.Int64("post.id", postID),
	)
	defer span.End()

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
