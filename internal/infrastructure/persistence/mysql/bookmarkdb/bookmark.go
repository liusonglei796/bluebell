package bookmarkdb

import (
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/persistence/mysql/model"
	infratrace "bluebell/internal/infrastructure/trace"
	"context"
	"fmt"

	"gorm.io/gorm"
)

var tracer = infratrace.TracerForModule("dao/mysql/bookmark")

type bookmarkRepoStruct struct {
	db *gorm.DB
}

func NewBookmarkRepo(db *gorm.DB) domain.BookmarkRepository {
	return &bookmarkRepoStruct{db: db}
}

func toModelBookmark(b *entity.Bookmark) *model.Bookmark {
	if b == nil {
		return nil
	}
	return &model.Bookmark{
		UserID: b.UserID,
		PostID: b.PostID,
	}
}

func fromModelBookmark(m *model.Bookmark) *entity.Bookmark {
	if m == nil {
		return nil
	}
	return &entity.Bookmark{
		UserID:    m.UserID,
		PostID:    m.PostID,
		CreatedAt: m.CreatedAt,
	}
}

func (r *bookmarkRepoStruct) CreateBookmark(ctx context.Context, bookmark *entity.Bookmark) error {
	ctx, span := tracer.Start(ctx, "BookmarkDAO.CreateBookmark")
	defer span.End()

	m := toModelBookmark(bookmark)
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		return fmt.Errorf("create bookmark failed: %w", err)
	}
	return nil
}

func (r *bookmarkRepoStruct) DeleteBookmark(ctx context.Context, userID, postID int64) error {
	ctx, span := tracer.Start(ctx, "BookmarkDAO.DeleteBookmark")
	defer span.End()

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.Bookmark{}).Error
	if err != nil {
		return fmt.Errorf("delete bookmark failed: %w", err)
	}
	return nil
}

func (r *bookmarkRepoStruct) IsBookmarked(ctx context.Context, userID, postID int64) (bool, error) {
	ctx, span := tracer.Start(ctx, "BookmarkDAO.IsBookmarked")
	defer span.End()

	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Bookmark{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check bookmark failed: %w", err)
	}
	return count > 0, nil
}

func (r *bookmarkRepoStruct) GetUserBookmarks(ctx context.Context, userID int64, page, size int) ([]*entity.Bookmark, error) {
	ctx, span := tracer.Start(ctx, "BookmarkDAO.GetUserBookmarks")
	defer span.End()

	var models []*model.Bookmark
	offset := (page - 1) * size
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Offset(offset).Limit(size).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("get user bookmarks failed: %w", err)
	}
	result := make([]*entity.Bookmark, len(models))
	for i, m := range models {
		result[i] = fromModelBookmark(m)
	}
	return result, nil
}

func (r *bookmarkRepoStruct) GetBookmarkCount(ctx context.Context, userID int64) (int, error) {
	ctx, span := tracer.Start(ctx, "BookmarkDAO.GetBookmarkCount")
	defer span.End()

	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Bookmark{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("get bookmark count failed: %w", err)
	}
	return int(count), nil
}
