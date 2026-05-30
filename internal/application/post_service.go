package application

import (
	"bluebell/internal/domain"
	postreq "bluebell/internal/application/dto/request/post"
	postResp "bluebell/internal/application/dto/response/post"
	searchResp "bluebell/internal/application/dto/response/search"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/infrastructure/mq"
	"bluebell/internal/domain/entity"
	"context"
	"fmt"
	"strconv"
	"time"
	"go.uber.org/zap"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/trace"
)

var tracerPost = trace.TracerForModule("service/post")

type PostService struct {
	postRepo       domain.PostRepository
	postCache      domain.PostCacheRepository
	remarkRepo     domain.RemarkRepository
	publisher      *mq.Publisher
	searchRepo     domain.PostSearchRepository
	searchSyncRepo domain.PostSearchSyncRepository
}

func NewPostService(
	postRepo domain.PostRepository,
	postCache domain.PostCacheRepository,
	remarkRepo domain.RemarkRepository,
	publisher *mq.Publisher,
	searchRepo domain.PostSearchRepository,
	searchSyncRepo domain.PostSearchSyncRepository,
) *PostService {
	return &PostService{
		postRepo:       postRepo,
		postCache:      postCache,
		remarkRepo:     remarkRepo,
		publisher:      publisher,
		searchRepo:     searchRepo,
		searchSyncRepo: searchSyncRepo,
	}
}

func (s *PostService) CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (postID string, err error) {
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

	err = s.postCache.CreatePost(ctx, postIDInt, p.CommunityID)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
	}

	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:     authorID,
		Type:       "post_created",
		TargetID:   postID,
		TargetName: p.Title,
		Timestamp:  time.Now().Unix(),
	})

	if s.searchSyncRepo != nil {
		if err := s.searchSyncRepo.SyncPostIndex(ctx, post); err != nil {
			logger.WithContext(ctx).Warn("searchSyncRepo.SyncPostIndex failed",
				zap.String("post_id", post.PostID),
				zap.Error(err))
		}
	}

	return postID, nil
}

func (s *PostService) GetPostByID(ctx context.Context, pid int64) (data *postResp.DetailResponse, err error) {
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

	voteData, err := s.postCache.GetPostsVoteData(ctx, []string{post.PostID})
	if err != nil {
		logger.WithContext(ctx).Warn("postCache.GetPostsVoteData failed in GetPostByID",
			zap.String("post_id", post.PostID),
			zap.Error(err))
	} else if len(voteData) > 0 {
		data.VoteNum = voteData[0]
	}

	return data, nil
}

func (s *PostService) GetPostList(ctx context.Context, p *postreq.PostListRequest) (data []*postResp.DetailResponse, err error) {
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

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.GetPostsVoteData failed", zap.Error(err))
		voteData = make([]int64, len(ids))
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

func (s *PostService) GetCommunityPostList(ctx context.Context, p *postreq.PostListRequest) (data []*postResp.DetailResponse, err error) {
	ids, err := s.postCache.GetCommunityPostIDsInOrder(ctx, p.CommunityID, p.Order, p.Page, p.Size)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.GetCommunityPostIDsInOrder failed",
			zap.Int64("community_id", p.CommunityID),
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	if len(ids) == 0 {
		data = make([]*postResp.DetailResponse, 0)
		return data, nil
	}

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
	if err != nil {
		logger.WithContext(ctx).Error("postCache.GetPostsVoteData failed", zap.Error(err))
		voteData = make([]int64, len(ids))
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

func (s *PostService) DeletePost(ctx context.Context, postID int64, userID int64) error {
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

	if err := post.CanBeDeletedBy(userID); err != nil {
		return err
	}

	if err := s.remarkRepo.DeleteRemarksByPostID(ctx, postID); err != nil {
		logger.WithContext(ctx).Error("remarkRepo.DeleteRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	err = s.postRepo.DeletePostByAuthor(ctx, postID, userID)
	if err != nil {
		logger.WithContext(ctx).Error("postRepo.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	if s.searchSyncRepo != nil {
		postIDStr := strconv.FormatInt(postID, 10)
		if err := s.searchSyncRepo.DeletePostIndex(ctx, postIDStr); err != nil {
			logger.WithContext(ctx).Error("searchSyncRepo.DeletePostIndex failed",
				zap.Int64("post_id", postID),
				zap.Error(err))
		}
	}

	if err := s.postCache.DeletePost(ctx, postID, post.CommunityID); err != nil {
		logger.WithContext(ctx).Error("postCache.DeletePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
	}

	return nil
}

func (s *PostService) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
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

	communityID, err := s.postCache.GetPostCommunityID(ctx, p.PostID)
	if err != nil {
		return fmt.Errorf("get community id for vote failed: %w", err)
	}

	if err := s.postCache.VoteForPost(ctx, userIDStr, postIDStr, strconv.FormatInt(communityID, 10), float64(p.Direction)); err != nil {
		return err
	}

	if s.publisher != nil {
		_ = s.publisher.PublishVote(ctx, &mq.VoteMessage{
			MsgID:  strconv.FormatInt(snowflake.GenID(), 10),
			PostID: postIDStr,
			UserID: userIDStr,
			Action: int(p.Direction),
		})
		_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
			UserID:    userID,
			Type:      "vote_up",
			TargetID:  postIDStr,
			Timestamp: time.Now().Unix(),
		})
	}

	return nil
}

func (s *PostService) RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (remarkID uint, err error) {
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

	remark := &entity.Remark{
		PostID:   req.PostID,
		Content:  req.Content,
		AuthorID: userID,
		ReplyTo:  req.ReplyTo,
	}
	if err := remark.Validate(); err != nil {
		return 0, err
	}

	if err := s.remarkRepo.CreateRemark(ctx, remark); err != nil {
		logger.WithContext(ctx).Error("remarkPost: remarkRepo.CreateRemark failed",
			zap.Int64("post_id", req.PostID),
			zap.Int64("author_id", userID),
			zap.Error(err))
		return 0, entity.Wrap(entity.ErrServerBusy, err)
	}

	return remark.ID, nil
}

func (s *PostService) GetPostRemarks(ctx context.Context, postID int64, replyTo int64) ([]*postResp.RemarkDetail, error) {
	remarks, err := s.remarkRepo.GetRemarksByPostID(ctx, postID)
	if err != nil {
		logger.WithContext(ctx).Error("getPostRemarks: remarkRepo.GetRemarksByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	resp := make([]*postResp.RemarkDetail, 0, len(remarks))
	for _, r := range remarks {
		if replyTo > 0 && r.ReplyTo != replyTo {
			continue
		}
		if replyTo == 0 && r.ReplyTo > 0 {
			continue
		}

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

func (s *PostService) SearchPosts(ctx context.Context, keyword string, page, pageSize int) (*searchResp.SearchResponse, error) {
	if s.searchRepo == nil {
		logger.WithContext(ctx).Warn("searchRepo is not initialized")
		return &searchResp.SearchResponse{Posts: []searchResp.SearchPostDoc{}}, nil
	}

	res, err := s.searchRepo.Search(ctx, keyword, page, pageSize)
	if err != nil {
		return nil, err
	}

	resp := &searchResp.SearchResponse{
		Total:    res.Total,
		Page:     res.Page,
		PageSize: res.PageSize,
		Posts:    make([]searchResp.SearchPostDoc, 0, len(res.Posts)),
	}
	for _, p := range res.Posts {
		resp.Posts = append(resp.Posts, searchResp.SearchPostDoc{
			PostID:           p.PostID,
			AuthorID:         p.AuthorID,
			CommunityID:      p.CommunityID,
			PostTitle:        p.PostTitle,
			Content:          p.Content,
			Status:           p.Status,
			CreatedAt:        p.CreatedAt,
			HighlightTitle:   p.HighlightTitle,
			HighlightContent: p.HighlightContent,
		})
	}
	return resp, nil
}
