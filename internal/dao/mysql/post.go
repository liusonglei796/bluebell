package mysql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// PostDao 帖子数据访问对象
type PostDao struct {
	db *gorm.DB
}

// NewPostDao 创建帖子 DAO 实例
func NewPostDao(db *gorm.DB) *PostDao {
	return &PostDao{db: db}
}

// CreatePost 创建帖子
func (d *PostDao) CreatePost(ctx context.Context, post *model.Post) error {
	err := d.db.WithContext(ctx).Create(post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.ErrDuplicateSubmit
		}
		return fmt.Errorf("创建帖子失败: %w", err)
	}
	return nil
}

// CreatePostWithAuthor 创建帖子并关联作者（多对多）
func (d *PostDao) CreatePostWithAuthor(ctx context.Context, post *model.Post, author *model.User) error {
	err := d.db.WithContext(ctx).Create(post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.ErrDuplicateSubmit
		}
		return fmt.Errorf("创建帖子失败: %w", err)
	}
	// 关联作者到中间表
	postAuthor := &model.PostAuthor{
		PostID: post.PostID,
		UserID: author.UserID,
	}
	if err := d.db.WithContext(ctx).Create(postAuthor).Error; err != nil {
		return fmt.Errorf("关联作者失败: %w", err)
	}
	return nil
}

// GetPostByID 根据帖子ID查询帖子详情
func (d *PostDao) GetPostByID(ctx context.Context, pid int64) (*model.Post, error) {
	m := new(model.Post)

	err := d.db.WithContext(ctx).
		Preload("Authors").
		Joins("Community").
		Where("post.post_id = ?", pid).
		Where("post.status = ?", model.PostStatusPublished).
		First(m).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询帖子失败: %w", err)
	}
	return m, nil
}

// GetPostListByIDsWithPreload 根据给定的ID列表查询帖子详情（使用 LEFT JOIN 替代 Preload，将三表联查缩减为1次SQL）
func (d *PostDao) GetPostListByIDsWithPreload(ctx context.Context, ids []string) (posts []*model.Post, err error) {
	return d.GetPostListByIDsWithJoins(ctx, ids)
}

// GetPostListByIDsWithJoins 根据给定的ID列表批量联表查询帖子详情
func (d *PostDao) GetPostListByIDsWithJoins(ctx context.Context, ids []string) (posts []*model.Post, err error) {
	if len(ids) == 0 {
		return make([]*model.Post, 0), nil
	}

	var mPosts []*model.Post

	err = d.db.WithContext(ctx).
		Preload("Authors").
		Joins("Community").
		Where("post.post_id IN ?", ids).
		Where("post.status = ?", model.PostStatusPublished).
		Find(&mPosts).Error

	if err != nil {
		return nil, fmt.Errorf("批量查询帖子失败: %w", err)
	}

	// 按照传入的 ids 顺序排列结果
	postMap := make(map[string]*model.Post, len(mPosts))
	for _, m := range mPosts {
		postMap[strconv.FormatInt(m.PostID, 10)] = m
	}

	orderedPosts := make([]*model.Post, 0, len(ids))
	for _, id := range ids {
		if post, ok := postMap[id]; ok {
			orderedPosts = append(orderedPosts, post)
		}
	}

	return orderedPosts, nil
}

// GetPostListByIDsSingleTable 基于反范式冗余字段单表极速批量查帖子（0 次 JOIN / Preload）
func (d *PostDao) GetPostListByIDsSingleTable(ctx context.Context, ids []string) (posts []*model.Post, err error) {
	if len(ids) == 0 {
		return make([]*model.Post, 0), nil
	}

	var mPosts []*model.Post
	err = d.db.WithContext(ctx).
		Select("id", "post_id", "author_id", "author_name", "community_id", "community_name", "post_title", "tag_names", "status", "is_pinned", "is_highlighted", "bookmark_count", "comment_count", "vote_num", "score", "created_at").
		Where("post_id IN ?", ids).
		Where("status = ?", model.PostStatusPublished).
		Find(&mPosts).Error
	if err != nil {
		return nil, fmt.Errorf("批量单表查询帖子失败: %w", err)
	}

	postMap := make(map[string]*model.Post, len(mPosts))
	for _, m := range mPosts {
		postMap[strconv.FormatInt(m.PostID, 10)] = m
	}

	orderedPosts := make([]*model.Post, 0, len(ids))
	for _, id := range ids {
		if post, ok := postMap[id]; ok {
			orderedPosts = append(orderedPosts, post)
		}
	}
	return orderedPosts, nil
}

// DeletePostByAuthor 软删除帖子（权限已在领域模型层校验，无需重复 SELECT 鉴权）
func (d *PostDao) DeletePostByAuthor(ctx context.Context, postID int64, authorID int64) error {
	result := d.db.WithContext(ctx).Model(&model.Post{}).
		Where("post_id = ?", postID).
		Where("status = ?", model.PostStatusPublished).
		Update("status", model.PostStatusDeleted)

	if result.Error != nil {
		return fmt.Errorf("删除帖子失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

// GetPostListByAuthorIDs 批量获取指定作者列表的最新帖子 (支撑 Feed 流读扩散归并)
func (d *PostDao) GetPostListByAuthorIDs(ctx context.Context, authorIDs []int64, limit int) ([]*model.Post, error) {
	if len(authorIDs) == 0 {
		return make([]*model.Post, 0), nil
	}

	var posts []*model.Post
	err := d.db.WithContext(ctx).
		Table("post").
		Joins("JOIN post_author ON post.post_id = post_author.post_id").
		Where("post_author.user_id IN ? AND post.status = ?", authorIDs, model.PostStatusPublished).
		Order("post.created_at DESC").
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("find posts by author ids failed: %w", err)
	}
	return posts, nil
}

// GetPostList 直查 MySQL 分页列表（无 Redis 模式，基于单表冗余字段极速分页，排除 content 大字段）
func (d *PostDao) GetPostList(ctx context.Context, page, size int64, order string) ([]*model.Post, error) {
	var posts []*model.Post
	offset := int((page - 1) * size)
	query := d.db.WithContext(ctx).
		Select("id", "post_id", "author_id", "author_name", "community_id", "community_name", "post_title", "tag_names", "status", "is_pinned", "is_highlighted", "bookmark_count", "comment_count", "vote_num", "score", "created_at").
		Where("status = ?", model.PostStatusPublished)

	if order == "score" {
		query = query.Order("score DESC, created_at DESC")
	} else {
		query = query.Order("created_at DESC, id DESC")
	}

	err := query.Offset(offset).Limit(int(size)).Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("直查帖子列表失败: %w", err)
	}
	return posts, nil
}

// VotePost 直写 MySQL 投票（单条原子更新，无长事务与 FOR UPDATE 排他锁排队）
func (d *PostDao) VotePost(ctx context.Context, postID int64, direction int8) error {
	txCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	delta := int64(direction)
	res := d.db.WithContext(txCtx).Model(&model.Post{}).
		Where("post_id = ? AND status = ?", postID, model.PostStatusPublished).
		Updates(map[string]interface{}{
			"vote_num": gorm.Expr("vote_num + ?", delta),
			"score":    gorm.Expr("score + ?", delta*432),
		})

	if res.Error != nil {
		return fmt.Errorf("原子更新投票失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}
