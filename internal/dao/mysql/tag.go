package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"bluebell/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TagDao 标签数据访问对象
type TagDao struct {
	db *gorm.DB
}

// NewTagDao 创建标签 DAO 实例
func NewTagDao(db *gorm.DB) *TagDao {
	return &TagDao{db: db}
}

// CreateTag 创建社区标签
func (d *TagDao) CreateTag(ctx context.Context, communityID int64, name string) (*model.Tag, error) {
	tag := &model.Tag{
		CommunityID: communityID,
		Name:        name,
	}
	if err := d.db.WithContext(ctx).Create(tag).Error; err != nil {
		return nil, fmt.Errorf("create tag failed: %w", err)
	}
	return tag, nil
}

// GetCommunityTags 获取社区下的所有标签
func (d *TagDao) GetCommunityTags(ctx context.Context, communityID int64) ([]*model.Tag, error) {
	var tags []*model.Tag
	err := d.db.WithContext(ctx).Where("community_id = ?", communityID).Order("post_count DESC").Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// BindPostTags 绑定帖子与标签关联并同步冗余 tag_names 到 post 表
func (d *TagDao) BindPostTags(ctx context.Context, postID int64, communityID int64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var postTags []model.PostTag
		for _, tid := range tagIDs {
			postTags = append(postTags, model.PostTag{
				PostID:      postID,
				TagID:       tid,
				CommunityID: communityID,
			})
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&postTags).Error; err != nil {
			return err
		}

		// 递增标签的 post_count
		if err := tx.Model(&model.Tag{}).Where("id IN ?", tagIDs).Update("post_count", gorm.Expr("post_count + 1")).Error; err != nil {
			return err
		}

		// 查询标签名并同步冗余到 post 表的 tag_names 字段
		var tagNames []string
		if err := tx.Model(&model.Tag{}).Where("id IN ?", tagIDs).Pluck("name", &tagNames).Error; err == nil && len(tagNames) > 0 {
			tagStr := strings.Join(tagNames, ",")
			_ = tx.Model(&model.Post{}).Where("post_id = ?", strconv.FormatInt(postID, 10)).Update("tag_names", tagStr).Error
		}

		return nil
	})
}

// GetTagNamesByIDs 根据标签ID列表批量获取标签名称
func (d *TagDao) GetTagNamesByIDs(ctx context.Context, tagIDs []uint64) ([]string, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	var names []string
	err := d.db.WithContext(ctx).Model(&model.Tag{}).Where("id IN ?", tagIDs).Pluck("name", &names).Error
	return names, err
}

// GetPostTags 获取帖子关联的标签名称列表
func (d *TagDao) GetPostTags(ctx context.Context, postID int64) ([]string, error) {
	var names []string
	err := d.db.WithContext(ctx).Table("tag").
		Joins("JOIN post_tag ON tag.id = post_tag.tag_id").
		Where("post_tag.post_id = ?", postID).
		Pluck("tag.name", &names).Error
	if err != nil {
		return nil, err
	}
	return names, nil
}

// GetPostIDsByTag 分页获取打上某标签的帖子ID
func (d *TagDao) GetPostIDsByTag(ctx context.Context, tagID uint64, page, size int64) ([]string, int64, error) {
	var ids []string
	var total int64

	db := d.db.WithContext(ctx).Model(&model.PostTag{}).Where("tag_id = ?", tagID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at DESC").Pluck("post_id", &ids).Error; err != nil {
		return nil, 0, err
	}

	return ids, total, nil
}
