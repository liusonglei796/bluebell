package mysql

import (
	"bluebell/models"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// CreatePost 创建帖子
// DAO层只返回错误，不打印日志，由上层统一处理
func CreatePost(ctx context.Context, post *models.Post) (err error) {
	// GORM 使用 Create 方法插入记录
	err = db.WithContext(ctx).Create(post).Error
	if err != nil {
		return fmt.Errorf("insert post failed: %w", err)
	}
	return nil
}

// GetPostByID 根据帖子ID查询帖子详情（带预加载）
// 使用 Preload 自动加载关联的作者和社区信息，避免 N+1 查询问题
// 自动过滤已删除帖子（status = 0）
func GetPostByID(ctx context.Context, pid int64) (post *models.Post, err error) {
	post = new(models.Post)

	// Preload 会自动执行以下 SQL:
	// 1. SELECT * FROM post WHERE post_id = ? AND status = 1
	// 2. SELECT * FROM user WHERE user_id IN (post.author_id)
	// 3. SELECT * FROM community WHERE community_id IN (post.community_id)
	err = db.WithContext(ctx).Preload("Author"). // 预加载作者信息
							Preload("Community"). // 预加载社区信息
							Where("post_id = ?", pid).
							Where("status = ?", 1). // 过滤已删除帖子
							First(post).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query post by id with preload failed: %w", err)
	}
	return
}

// GetPostListByIDsWithPreload 根据给定的ID列表查询帖子详情（带预加载）
// 为什么新增：批量查询帖子的同时，自动预加载所有帖子的作者和社区信息
// 解决 N+1 问题：如果查询 100 个帖子，传统方式需要 1 + 100 + 100 = 201 次查询
//
//	使用 Preload 只需要 1 + 1 + 1 = 3 次查询
//
// 自动过滤已删除帖子（status = 0）
func GetPostListByIDsWithPreload(ctx context.Context, ids []string) (posts []*models.Post, err error) {
	if len(ids) == 0 {
		return make([]*models.Post, 0), nil
	}

	posts = make([]*models.Post, 0, len(ids))

	// Preload 会自动批量查询:
	// 1. SELECT * FROM post WHERE post_id IN (ids) AND status = 1
	// 2. SELECT * FROM user WHERE user_id IN (所有帖子的 author_id)
	// 3. SELECT * FROM community WHERE community_id IN (所有帖子的 community_id)
	err = db.WithContext(ctx).Preload("Author"). // 批量预加载所有作者
							Preload("Community"). // 批量预加载所有社区
							Where("post_id IN ?", ids).
							Where("status = ?", 1). // 过滤已删除帖子
							Find(&posts).Error

	if err != nil {
		return nil, fmt.Errorf("query post list by ids with preload failed: %w", err)
	}

	// 按照传入的 ids 顺序排列结果
	postMap := make(map[string]*models.Post, len(posts))
	for _, post := range posts {
		postMap[fmt.Sprintf("%d", post.ID)] = post
	}

	orderedPosts := make([]*models.Post, 0, len(ids))
	for _, id := range ids {
		if post, ok := postMap[id]; ok {
			orderedPosts = append(orderedPosts, post)
		}
	}

	return orderedPosts, nil
}

// DeletePost 软删除帖子（更新 status 为 0）
func DeletePost(ctx context.Context, postID int64) error {
	result := db.WithContext(ctx).Model(&models.Post{}).
		Where("post_id = ?", postID).
		Where("status = ?", 1). // 确保只删除正常的帖子
		Update("status", 0)

	if result.Error != nil {
		return fmt.Errorf("delete post failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeletePostByAuthor 软删除帖子（带作者验证）
func DeletePostByAuthor(ctx context.Context, postID, authorID int64) error {
	result := db.WithContext(ctx).Model(&models.Post{}).
		Where("post_id = ?", postID).
		Where("author_id = ?", authorID).
		Where("status = ?", 1).
		Update("status", 0)

	if result.Error != nil {
		return fmt.Errorf("delete post failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
