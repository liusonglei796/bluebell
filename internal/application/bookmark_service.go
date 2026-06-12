package application

import (
	bookmarkresp "bluebell/internal/application/dto/response/bookmark"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/logger"
	infratrace "bluebell/internal/infrastructure/trace"
	"context"
	"go.uber.org/zap"
	"strconv"
)

var tracerBookmark = infratrace.TracerForModule("service/bookmark")

type BookmarkService struct {
	bookmarkRepo  domain.BookmarkRepository
	postRepo      domain.PostRepository
	userRepo      domain.UserRepository
	communityRepo domain.CommunityRepository
}

func NewBookmarkService(
	bookmarkRepo domain.BookmarkRepository,
	postRepo domain.PostRepository,
	userRepo domain.UserRepository,
	communityRepo domain.CommunityRepository,
) *BookmarkService {
	return &BookmarkService{
		bookmarkRepo:  bookmarkRepo,
		postRepo:      postRepo,
		userRepo:      userRepo,
		communityRepo: communityRepo,
	}
}

func (s *BookmarkService) CreateBookmark(ctx context.Context, userID, postID int64) error {
	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.CreateBookmark")
	defer span.End()
	infratrace.WithUserID(ctx, userID)
	infratrace.WithPostID(ctx, postID)

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		logger.WithContext(ctx).Error("GetPostByID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return err
	}
	if post == nil {
		return entity.ErrInvalidParam
	}

	bookmarked, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
	if err != nil {
		logger.WithContext(ctx).Error("Check bookmark status failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", postID),
			zap.Error(err))
		return err
	}
	if bookmarked {
		return nil
	}

	bookmark := &entity.Bookmark{
		UserID: userID,
		PostID: postID,
	}
	if err := s.bookmarkRepo.CreateBookmark(ctx, bookmark); err != nil {
		logger.WithContext(ctx).Error("Create bookmark failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", postID),
			zap.Error(err))
		return err
	}

	infratrace.SetSpanSuccess(ctx)
	return nil
}

func (s *BookmarkService) DeleteBookmark(ctx context.Context, userID, postID int64) error {
	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.DeleteBookmark")
	defer span.End()
	infratrace.WithUserID(ctx, userID)
	infratrace.WithPostID(ctx, postID)

	if err := s.bookmarkRepo.DeleteBookmark(ctx, userID, postID); err != nil {
		logger.WithContext(ctx).Error("Delete bookmark failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", postID),
			zap.Error(err))
		return err
	}

	infratrace.SetSpanSuccess(ctx)
	return nil
}

func (s *BookmarkService) GetUserBookmarks(ctx context.Context, userID int64, page, size int) (*bookmarkresp.BookmarkListResponse, error) {
	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.GetUserBookmarks")
	defer span.End()
	infratrace.WithUserID(ctx, userID)

	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}

	bookmarks, err := s.bookmarkRepo.GetUserBookmarks(ctx, userID, page, size)
	if err != nil {
		logger.WithContext(ctx).Error("Get user bookmarks failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	total, err := s.bookmarkRepo.GetBookmarkCount(ctx, userID)
	if err != nil {
		logger.WithContext(ctx).Error("Get bookmark count failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	var postIDStrs []string
	for _, b := range bookmarks {
		postIDStrs = append(postIDStrs, strconv.FormatInt(b.PostID, 10))
	}

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, postIDStrs)
	if err != nil {
		logger.WithContext(ctx).Error("Get posts failed", zap.Error(err))
		return nil, err
	}

	postMap := make(map[string]*entity.Post)
	for _, p := range posts {
		postMap[p.PostID] = p
	}

	var authorIDs []int64
	for _, p := range posts {
		if p.AuthorID > 0 {
			authorIDs = append(authorIDs, p.AuthorID)
		}
	}

	authorMap := make(map[int64]*entity.User)
	if len(authorIDs) > 0 {
		authors, err := s.userRepo.GetUsersByIDs(ctx, authorIDs)
		if err != nil {
			logger.WithContext(ctx).Error("Get authors failed", zap.Error(err))
		} else {
			for _, a := range authors {
				authorMap[a.UserID] = a
			}
		}
	}

	var communityIDs []int64
	for _, p := range posts {
		if p.CommunityID > 0 {
			communityIDs = append(communityIDs, p.CommunityID)
		}
	}

	communityMap := make(map[int64]*entity.Community)
	for _, cid := range communityIDs {
		if _, ok := communityMap[cid]; ok {
			continue
		}
		c, err := s.communityRepo.GetCommunityDetailByID(ctx, cid)
		if err != nil {
			logger.WithContext(ctx).Error("Get community failed", zap.Error(err))
		} else {
			communityMap[cid] = c
		}
	}

	responses := make([]*bookmarkresp.BookmarkResponse, 0, len(bookmarks))
	for _, b := range bookmarks {
		postIDStr := strconv.FormatInt(b.PostID, 10)
		resp := &bookmarkresp.BookmarkResponse{
			PostID:    postIDStr,
			CreatedAt: b.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if p, ok := postMap[postIDStr]; ok {
			resp.PostTitle = p.PostTitle
			if a, ok := authorMap[p.AuthorID]; ok {
				resp.AuthorName = a.UserName
			}
			resp.CommunityID = p.CommunityID
			if c, ok := communityMap[p.CommunityID]; ok {
				resp.CommunityName = c.CommunityName
			}
		}
		responses = append(responses, resp)
	}

	infratrace.SetSpanSuccess(ctx)
	return &bookmarkresp.BookmarkListResponse{
		Bookmarks: responses,
		Total:     total,
		Page:      page,
		Size:      size,
	}, nil
}

func (s *BookmarkService) IsBookmarked(ctx context.Context, userID, postID int64) (*bookmarkresp.BookmarkStatusResponse, error) {
	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.IsBookmarked")
	defer span.End()
	infratrace.WithUserID(ctx, userID)
	infratrace.WithPostID(ctx, postID)

	bookmarked, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
	if err != nil {
		logger.WithContext(ctx).Error("Check bookmark status failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", postID),
			zap.Error(err))
		return nil, err
	}

	count, err := s.bookmarkRepo.GetBookmarkCount(ctx, userID)
	if err != nil {
		logger.WithContext(ctx).Error("Get bookmark count failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	infratrace.SetSpanSuccess(ctx)
	return &bookmarkresp.BookmarkStatusResponse{
		Bookmarked: bookmarked,
		Count:      count,
	}, nil
}
