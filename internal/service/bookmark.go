package service

import (
	"context"
	"strconv"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	bookmarkreq "bluebell/internal/dto/request/bookmark"
	bookmarkresp "bluebell/internal/dto/response/bookmark"
	postResp "bluebell/internal/dto/response/post"
	"bluebell/internal/model"
	"bluebell/internal/snowflake"

	"go.uber.org/zap"
)

// BookmarkService 收藏夹业务服务
type BookmarkService struct {
	bookmarkDao   *mysql.BookmarkDao
	postDao       *mysql.PostDao
	bookmarkCache *redis.BookmarkCache
	postService   *PostService
}

// NewBookmarkService 创建收藏夹服务实例
func NewBookmarkService(
	bookmarkDao *mysql.BookmarkDao,
	postDao *mysql.PostDao,
	bookmarkCache *redis.BookmarkCache,
	postService *PostService,
) *BookmarkService {
	return &BookmarkService{
		bookmarkDao:   bookmarkDao,
		postDao:       postDao,
		bookmarkCache: bookmarkCache,
		postService:   postService,
	}
}

// CreateFolder 创建自定义收藏夹
func (s *BookmarkService) CreateFolder(ctx context.Context, userID int64, p *bookmarkreq.CreateFolderRequest) (*bookmarkresp.FolderResponse, error) {
	folderID := snowflake.GenID()
	folder := &model.BookmarkFolder{
		ID:        folderID,
		UserID:    userID,
		Name:      p.Name,
		IsDefault: 0,
		IsPublic:  p.IsPublic,
	}

	if err := s.bookmarkDao.CreateFolder(ctx, folder); err != nil {
		zap.L().Error("bookmarkDao.CreateFolder failed", zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	return &bookmarkresp.FolderResponse{
		ID:        strconv.FormatInt(folderID, 10),
		UserID:    strconv.FormatInt(userID, 10),
		Name:      folder.Name,
		IsDefault: false,
		IsPublic:  folder.IsPublic == 1,
		CreatedAt: folder.CreatedAt,
	}, nil
}

// GetUserFolders 获取用户的所有收藏夹
func (s *BookmarkService) GetUserFolders(ctx context.Context, userID int64) ([]*bookmarkresp.FolderResponse, error) {
	// 确保默认收藏夹存在
	_, _ = s.bookmarkDao.GetOrCreateDefaultFolder(ctx, userID, snowflake.GenID())

	folders, err := s.bookmarkDao.GetUserFolders(ctx, userID)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	res := make([]*bookmarkresp.FolderResponse, len(folders))
	for i, f := range folders {
		res[i] = &bookmarkresp.FolderResponse{
			ID:        strconv.FormatInt(f.ID, 10),
			UserID:    strconv.FormatInt(f.UserID, 10),
			Name:      f.Name,
			IsDefault: f.IsDefault == 1,
			IsPublic:  f.IsPublic == 1,
			PostCount: f.PostCount,
			CreatedAt: f.CreatedAt,
		}
	}
	return res, nil
}

// AddBookmark 收藏帖子
func (s *BookmarkService) AddBookmark(ctx context.Context, userID int64, p *bookmarkreq.AddBookmarkRequest) error {
	folderID := p.FolderID
	if folderID == 0 {
		defFolder, err := s.bookmarkDao.GetOrCreateDefaultFolder(ctx, userID, snowflake.GenID())
		if err != nil {
			return model.Wrap(model.ErrServerBusy, err)
		}
		folderID = defFolder.ID
	}

	if err := s.bookmarkDao.AddBookmark(ctx, userID, folderID, p.PostID); err != nil {
		zap.L().Error("bookmarkDao.AddBookmark failed", zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	_ = s.bookmarkCache.AddBookmark(ctx, userID, p.PostID)
	return nil
}

// RemoveBookmark 取消收藏
func (s *BookmarkService) RemoveBookmark(ctx context.Context, userID int64, p *bookmarkreq.RemoveBookmarkRequest) error {
	if err := s.bookmarkDao.RemoveBookmark(ctx, userID, p.FolderID, p.PostID); err != nil {
		zap.L().Error("bookmarkDao.RemoveBookmark failed", zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 检查是否在其他收藏夹还有该贴
	hasOther, _ := s.bookmarkDao.IsPostBookmarked(ctx, userID, p.PostID)
	if !hasOther {
		_ = s.bookmarkCache.RemoveBookmark(ctx, userID, p.PostID)
	}
	return nil
}

// GetBookmarkPostList 获取收藏夹内的帖子列表
func (s *BookmarkService) GetBookmarkPostList(ctx context.Context, userID int64, p *bookmarkreq.BookmarkListRequest) (*bookmarkresp.BookmarkListResponse, error) {
	if p.Size <= 0 || p.Size > 50 {
		p.Size = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}

	var postIDs []string
	var total int64
	var err error

	if p.FolderID > 0 {
		postIDs, total, err = s.bookmarkDao.GetFolderPostIDs(ctx, p.FolderID, p.Page, p.Size)
	} else {
		postIDs, total, err = s.bookmarkDao.GetUserAllBookmarkedPostIDs(ctx, userID, p.Page, p.Size)
	}

	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	if len(postIDs) == 0 {
		return &bookmarkresp.BookmarkListResponse{Total: total, Posts: make([]*postResp.DetailResponse, 0)}, nil
	}

	posts, err := s.postDao.GetPostListByIDsWithPreload(ctx, postIDs)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	dtos := s.postService.FormatPostListDTOs(ctx, posts, postIDs, userID)
	return &bookmarkresp.BookmarkListResponse{
		Total: total,
		Posts: dtos,
	}, nil
}
