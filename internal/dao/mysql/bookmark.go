package mysql

import (
	"context"
	"errors"

	"bluebell/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BookmarkDao 收藏夹数据访问对象
type BookmarkDao struct {
	db *gorm.DB
}

// NewBookmarkDao 创建收藏夹 DAO 实例
func NewBookmarkDao(db *gorm.DB) *BookmarkDao {
	return &BookmarkDao{db: db}
}

// CreateFolder 创建收藏夹
func (d *BookmarkDao) CreateFolder(ctx context.Context, folder *model.BookmarkFolder) error {
	return d.db.WithContext(ctx).Create(folder).Error
}

// GetOrCreateDefaultFolder 获取或创建默认收藏夹
func (d *BookmarkDao) GetOrCreateDefaultFolder(ctx context.Context, userID int64, defaultFolderID int64) (*model.BookmarkFolder, error) {
	var folder model.BookmarkFolder
	err := d.db.WithContext(ctx).Where("user_id = ? AND is_default = 1", userID).First(&folder).Error
	if err == nil {
		return &folder, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		folder = model.BookmarkFolder{
			ID:        defaultFolderID,
			UserID:    userID,
			Name:      "默认收藏夹",
			IsDefault: 1,
			IsPublic:  0,
		}
		if err := d.db.WithContext(ctx).Create(&folder).Error; err != nil {
			return nil, err
		}
		return &folder, nil
	}

	return nil, err
}

// GetUserFolders 获取用户的所有收藏夹
func (d *BookmarkDao) GetUserFolders(ctx context.Context, userID int64) ([]*model.BookmarkFolder, error) {
	var folders []*model.BookmarkFolder
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).Order("is_default DESC, created_at ASC").Find(&folders).Error
	if err != nil {
		return nil, err
	}
	return folders, nil
}

// AddBookmark 添加帖子到收藏夹
func (d *BookmarkDao) AddBookmark(ctx context.Context, userID, folderID int64, postID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		bookmark := &model.PostBookmark{
			UserID:   userID,
			FolderID: folderID,
			PostID:   postID,
		}

		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(bookmark)
		if res.Error != nil {
			return res.Error
		}

		// 仅当新插入时递增计数
		if res.RowsAffected > 0 {
			tx.Model(&model.BookmarkFolder{}).Where("id = ?", folderID).Update("post_count", gorm.Expr("post_count + 1"))
			tx.Model(&model.Post{}).Where("post_id = ?", postID).Update("bookmark_count", gorm.Expr("bookmark_count + 1"))
		}

		return nil
	})
}

// RemoveBookmark 从收藏夹移除帖子
func (d *BookmarkDao) RemoveBookmark(ctx context.Context, userID, folderID int64, postID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Where("user_id = ? AND post_id = ?", userID, postID)
		if folderID > 0 {
			query = query.Where("folder_id = ?", folderID)
		}

		res := query.Delete(&model.PostBookmark{})
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected > 0 {
			if folderID > 0 {
				tx.Model(&model.BookmarkFolder{}).Where("id = ? AND post_count > 0", folderID).Update("post_count", gorm.Expr("post_count - 1"))
			}
			tx.Model(&model.Post{}).Where("post_id = ? AND bookmark_count > 0", postID).Update("bookmark_count", gorm.Expr("bookmark_count - 1"))
		}

		return nil
	})
}

// IsPostBookmarked 检查帖子是否已被该用户收藏
func (d *BookmarkDao) IsPostBookmarked(ctx context.Context, userID, postID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.PostBookmark{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFolderPostIDs 分页获取某收藏夹内的帖子ID列表
func (d *BookmarkDao) GetFolderPostIDs(ctx context.Context, folderID int64, page, size int64) ([]string, int64, error) {
	var ids []string
	var total int64

	db := d.db.WithContext(ctx).Model(&model.PostBookmark{}).Where("folder_id = ?", folderID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at DESC").Pluck("post_id", &ids).Error; err != nil {
		return nil, 0, err
	}

	return ids, total, nil
}

// GetUserAllBookmarkedPostIDs 分页获取用户所有收藏的帖子ID
func (d *BookmarkDao) GetUserAllBookmarkedPostIDs(ctx context.Context, userID int64, page, size int64) ([]string, int64, error) {
	var ids []string
	var total int64

	db := d.db.WithContext(ctx).Model(&model.PostBookmark{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at DESC").Pluck("post_id", &ids).Error; err != nil {
		return nil, 0, err
	}

	return ids, total, nil
}
