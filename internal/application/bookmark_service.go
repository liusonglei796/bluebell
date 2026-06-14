package application

import (
	bookmarkresp "bluebell/internal/application/dto/response/bookmark"
	"bluebell/internal/application/port"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"context"
	"strconv"
)

type BookmarkService struct {
	bookmarkRepo  domain.BookmarkRepository
	postRepo      domain.PostRepository
	userRepo      domain.UserRepository
	communityRepo domain.CommunityRepository
	logger        port.Logger
}

func NewBookmarkService(
	bookmarkRepo domain.BookmarkRepository,
	postRepo domain.PostRepository,
	userRepo domain.UserRepository,
	communityRepo domain.CommunityRepository,
	logger port.Logger,
) *BookmarkService {
	return &BookmarkService{
		bookmarkRepo:  bookmarkRepo,
		postRepo:      postRepo,
		userRepo:      userRepo,
		communityRepo: communityRepo,
		logger:        logger,
	}
}

func (s *BookmarkService) CreateBookmark(ctx context.Context, userID, postID int64) error {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		s.logger.Error(ctx, "GetPostByID failed",
			port.Int64("post_id", postID),
			port.Err(err))
		return err
	}
	if post == nil {
		return entity.ErrInvalidParam
	}

	bookmarked, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
	if err != nil {
		s.logger.Error(ctx, "Check bookmark status failed",
			port.Int64("user_id", userID),
			port.Int64("post_id", postID),
			port.Err(err))
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
		s.logger.Error(ctx, "Create bookmark failed",
			port.Int64("user_id", userID),
			port.Int64("post_id", postID),
			port.Err(err))
		return err
	}

	return nil
}

func (s *BookmarkService) DeleteBookmark(ctx context.Context, userID, postID int64) error {
	if err := s.bookmarkRepo.DeleteBookmark(ctx, userID, postID); err != nil {
		s.logger.Error(ctx, "Delete bookmark failed",
			port.Int64("user_id", userID),
			port.Int64("post_id", postID),
			port.Err(err))
		return err
	}

	return nil
}

func (s *BookmarkService) GetUserBookmarks(ctx context.Context, userID int64, page, size int) (*bookmarkresp.BookmarkListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}

	bookmarks, err := s.bookmarkRepo.GetUserBookmarks(ctx, userID, page, size)
	if err != nil {
		s.logger.Error(ctx, "Get user bookmarks failed",
			port.Int64("user_id", userID),
			port.Err(err))
		return nil, err
	}

	total, err := s.bookmarkRepo.GetBookmarkCount(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "Get bookmark count failed",
			port.Int64("user_id", userID),
			port.Err(err))
		return nil, err
	}

	var postIDStrs []string
	for _, b := range bookmarks {
		postIDStrs = append(postIDStrs, strconv.FormatInt(b.PostID, 10))
	}

	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, postIDStrs)
	if err != nil {
		s.logger.Error(ctx, "Get posts failed", port.Err(err))
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
			s.logger.Error(ctx, "Get authors failed", port.Err(err))
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
			s.logger.Error(ctx, "Get community failed", port.Err(err))
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

	return &bookmarkresp.BookmarkListResponse{
		Bookmarks: responses,
		Total:     total,
		Page:      page,
		Size:      size,
	}, nil
}

func (s *BookmarkService) IsBookmarked(ctx context.Context, userID, postID int64) (*bookmarkresp.BookmarkStatusResponse, error) {
	bookmarked, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
	if err != nil {
		s.logger.Error(ctx, "Check bookmark status failed",
			port.Int64("user_id", userID),
			port.Int64("post_id", postID),
			port.Err(err))
		return nil, err
	}

	count, err := s.bookmarkRepo.GetBookmarkCount(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "Get bookmark count failed",
			port.Int64("user_id", userID),
			port.Err(err))
		return nil, err
	}

	return &bookmarkresp.BookmarkStatusResponse{
		Bookmarked: bookmarked,
		Count:      count,
	}, nil
}
