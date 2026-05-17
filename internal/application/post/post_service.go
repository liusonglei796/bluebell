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
	"errors"
	"strconv"

	"go.uber.org/zap"
)

// postServiceStruct 帖子业务逻辑服务
type postServiceStruct struct {
	postRepo   domain.PostRepository
	postCache  domain.PostCacheRepository
	voteRepo   domain.VoteRepository
	remarkRepo domain.RemarkRepository
	publisher *mq.Publisher
	esClient   *es.Client
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postRepo domain.PostRepository,
	postCache domain.PostCacheRepository,
	voteRepo domain.VoteRepository,
	remarkRepo domain.RemarkRepository,
	publisher *mq.Publisher,
	esClient *es.Client,
) application.PostService {
	return &postServiceStruct{
		postRepo:   postRepo,
		postCache:  postCache,
		voteRepo:   voteRepo,
		remarkRepo: remarkRepo,
		publisher:  publisher,
		esClient:   esClient,
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
		zap.L().Error("postRepo.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
		return "", entity.Wrap(entity.ErrServerBusy, err)
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
	post, err := s.postRepo.GetPostByID(ctx, pid)
	if err != nil {
		zap.L().Error("postRepo.GetPostByID failed",
			zap.Int64("post_id", pid),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	if post == nil || post.PostID == "" {
		return nil, entity.ErrNotFound
	}

	if post.Author == nil || post.Author.UserID == 0 {
		zap.L().Warn("author not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("author_id", post.AuthorID))
		return nil, entity.ErrNotFound
	}

	if post.Community == nil || post.Community.ID == 0 {
		zap.L().Warn("community not found for post",
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
		zap.L().Error("postCache.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
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
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	zap.L().Debug("GetPostListByIDsWithPreload", zap.Any("posts", posts))

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("postCache.GetPostsVoteData failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
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
		return nil, entity.Wrap(entity.ErrServerBusy, err)
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
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		zap.L().Error("postCache.GetPostsVoteData failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
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
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		zap.L().Error("postRepo.GetPostByID failed",
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
		zap.L().Error("remarkRepo.DeleteRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	// 2. 软删除帖子 (status = 0)
	err = s.postRepo.DeletePostByAuthor(ctx, postID, userID)
	if err != nil {
		zap.L().Error("postRepo.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
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

// VoteForPost 投票业务逻辑 (Architecture D: Redis Lua + MQ 持久化)
//
//	请求 → Redis Lua 原子更新(ZSet+Hash+Gravity score) → 发 MQ → 返回
//	                                                    → Consumer → MySQL UPSERT(持久化兜底)
func (s *postServiceStruct) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
	// 领域校验
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

	// 1. 获取 community_id (优先 Redis → 回退 MySQL)
	communityID, err := s.postCache.GetPostCommunityID(ctx, p.PostID)
	if err != nil {
		// Redis 缓存缺失，回退到 MySQL 查找帖子
		post, err := s.postRepo.GetPostByID(ctx, p.PostID)
		if err != nil {
			return entity.Wrap(entity.ErrServerBusy, err)
		}
		if post == nil {
			return entity.ErrNotFound
		}
		communityID = post.CommunityID
		// 引导 Redis 缓存，让后续投票走快路径
		if err := s.postCache.CreatePost(ctx, p.PostID, communityID); err != nil {
			zap.L().Error("postCache.CreatePost bootstrap failed", zap.Error(err))
		}
	}
	communityIDStr := strconv.FormatInt(communityID, 10)

	// 2. Redis Lua 原子更新 (ZSet + Hash + Gravity score)
	err = s.postCache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, float64(p.Direction))
	if err != nil {
		if errors.Is(err, entity.ErrVoteTimeExpire) {
			return err
		}
		if errors.Is(err, entity.ErrVoteRepeated) {
			// 重复投票是幂等操作，不报错
			return nil
		}
		// Lua 执行失败（如 Redis 宕机），记录日志但继续发 MQ 让消费者兜底
		zap.L().Error("postCache.VoteForPost failed, fallback to MQ persistence",
			zap.String("post_id", postIDStr),
			zap.String("user_id", userIDStr),
			zap.Error(err))
	}

	// 3. 异步入队 (MQ) — MySQL 异步持久化兜底
	if s.publisher != nil {
		msg := &mq.VoteMessage{
			MsgID:  strconv.FormatInt(snowflake.GenID(), 10),
			PostID: postIDStr,
			UserID: userIDStr,
			Action: int(p.Direction),
		}
		if err := s.publisher.PublishVote(ctx, msg); err != nil {
			zap.L().Error("publish vote message failed", zap.Error(err))
		}
	}

	return nil
}

func (s *postServiceStruct) RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (remarkID uint, err error) {
	// 1. 校验帖子是否存在
	post, err := s.postRepo.GetPostByID(ctx, req.PostID)
	if err != nil {
		zap.L().Error("remarkPost: postRepo.GetPostByID failed",
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
		zap.L().Error("remarkPost: remarkRepo.CreateRemark failed",
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
		zap.L().Error("getPostRemarks: remarkRepo.GetRemarksByPostID failed",
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
		zap.L().Warn("esClient is not initialized")
		return &es.SearchResponse{Posts: []es.SearchPostDoc{}}, nil
	}

	esReq := &es.SearchRequest{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}

	return s.esClient.Search(ctx, esReq)
}
