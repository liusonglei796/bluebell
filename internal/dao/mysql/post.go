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

// postJoinRow 接收单条 LEFT JOIN 查询的扁平结果
type postJoinRow struct {
	PostID        int64      `gorm:"column:post_id"`
	AuthorID      int64      `gorm:"column:author_id"`
	AuthorName    string     `gorm:"column:author_name"`
	CommunityID   int64      `gorm:"column:community_id"`
	CommunityName string     `gorm:"column:community_name"`
	PostTitle     string     `gorm:"column:post_title"`
	Content       string     `gorm:"column:content"`
	Status        int8       `gorm:"column:status"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	VoteNum       int64      `gorm:"column:vote_num"`
	Score         int64      `gorm:"column:score"`
	CID           *int64     `gorm:"column:c_id"`
	CName         *string    `gorm:"column:c_name"`
	CIntro        *string    `gorm:"column:c_intro"`
	CCreatedAt    *time.Time `gorm:"column:c_created_at"`
	UID           *int64     `gorm:"column:u_user_id"`
	UName         *string    `gorm:"column:u_user_name"`
}

// GetPostByID 根据帖子ID查询帖子详情（当前调用方案2：应用层轻量聚合方案）
func (d *PostDao) GetPostByID(ctx context.Context, pid int64) (*model.Post, error) {
	return d.GetPostByIDManualAggregate(ctx, pid)
}

// GetPostByIDWithJoin 方案1：单条 LEFT JOIN 方案（将 3 次 SQL 压缩为 1 次）
func (d *PostDao) GetPostByIDWithJoin(ctx context.Context, pid int64) (*model.Post, error) {
	var rows []postJoinRow
	query := `
		SELECT 
			p.post_id, p.author_id, p.author_name, p.community_id, p.community_name,
			p.post_title, p.content, p.status, p.created_at, p.vote_num, p.score,
			c.id AS c_id, c.community_name AS c_name, c.introduction AS c_intro, c.created_at AS c_created_at,
			u.user_id AS u_user_id, u.user_name AS u_user_name
		FROM post p
		LEFT JOIN community c ON p.community_id = c.id
		LEFT JOIN post_author pa ON p.post_id = pa.post_id
		LEFT JOIN user u ON pa.user_id = u.user_id
		WHERE p.post_id = ? AND p.status = 1
	`
	err := d.db.WithContext(ctx).Raw(query, pid).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("join查询帖子失败: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	first := rows[0]
	post := &model.Post{
		PostID:        first.PostID,
		AuthorID:      first.AuthorID,
		AuthorName:    first.AuthorName,
		CommunityID:   first.CommunityID,
		CommunityName: first.CommunityName,
		PostTitle:     first.PostTitle,
		Content:       first.Content,
		Status:        first.Status,
		VoteNum:       first.VoteNum,
		Score:         first.Score,
	}
	post.CreatedAt = first.CreatedAt

	if first.CID != nil && *first.CID > 0 {
		post.Community = &model.Community{
			CommunityName: *first.CName,
			Introduction:  *first.CIntro,
		}
		post.Community.ID = uint(*first.CID)
		if first.CCreatedAt != nil {
			post.Community.CreatedAt = *first.CCreatedAt
		}
	}

	seenAuthors := make(map[int64]bool)
	for _, r := range rows {
		if r.UID != nil && *r.UID > 0 && !seenAuthors[*r.UID] {
			seenAuthors[*r.UID] = true
			author := model.User{
				UserID: *r.UID,
			}
			if r.UName != nil {
				author.UserName = *r.UName
			}
			post.Authors = append(post.Authors, author)
		}
	}

	return post, nil
}

// GetPostByIDManualAggregate 方案2：应用层轻量聚合方案（无反射、轻量分步查）
func (d *PostDao) GetPostByIDManualAggregate(ctx context.Context, pid int64) (*model.Post, error) {
	// 1. 查 Post 主表 (单表主键点查)
	post := new(model.Post)
	err := d.db.WithContext(ctx).
		Where("post_id = ? AND status = ?", pid, model.PostStatusPublished).
		First(post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询帖子主表失败: %w", err)
	}

	// 2. 查 Community (单表点查)
	if post.CommunityID > 0 {
		comm := new(model.Community)
		if err := d.db.WithContext(ctx).Where("id = ?", post.CommunityID).First(comm).Error; err == nil {
			post.Community = comm
		}
	}

	// 3. 查 Authors (轻量 JOIN 查多对多关系，单次查询，无 GORM Preload 反射)
	var authors []model.User
	query := `SELECT u.user_id, u.user_name FROM user u JOIN post_author pa ON u.user_id = pa.user_id WHERE pa.post_id = ?`
	if err := d.db.WithContext(ctx).Raw(query, pid).Scan(&authors).Error; err == nil {
		post.Authors = authors
	}

	return post, nil
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
