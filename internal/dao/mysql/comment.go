package mysql

import (
	"context"
	"errors"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// CommentDao 二级评论数据访问对象
type CommentDao struct {
	db *gorm.DB
}

// NewCommentDao 创建评论 DAO 实例
func NewCommentDao(db *gorm.DB) *CommentDao {
	return &CommentDao{db: db}
}

// CreateComment 创建评论（根评论或子回复）
func (d *CommentDao) CreateComment(ctx context.Context, comment *model.Comment) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return fmt.Errorf("create comment failed: %w", err)
		}

		// 若为子评论，原子递增根评论的 reply_count
		if comment.RootID > 0 {
			res := tx.Model(&model.Comment{}).
				Where("id = ?", comment.RootID).
				Update("reply_count", gorm.Expr("reply_count + ?", 1))
			if res.Error != nil {
				return fmt.Errorf("incr root reply_count failed: %w", res.Error)
			}
		}

		// 原子递增 post 表的 comment_count
		res := tx.Model(&model.Post{}).
			Where("post_id = ?", comment.PostID).
			Update("comment_count", gorm.Expr("comment_count + ?", 1))
		if res.Error != nil {
			return fmt.Errorf("incr post comment_count failed: %w", res.Error)
		}

		return nil
	})
}

// GetCommentByID 根据评论ID获取单条评论
func (d *CommentDao) GetCommentByID(ctx context.Context, commentID int64) (*model.Comment, error) {
	var c model.Comment
	err := d.db.WithContext(ctx).Where("id = ?", commentID).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get comment by id failed: %w", err)
	}
	return &c, nil
}

// GetRootComments 分页获取帖子的根评论列表 (支持 hot/new 排序)
func (d *CommentDao) GetRootComments(ctx context.Context, postID int64, order string, page, size int64) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("post_id = ? AND root_id = 0 AND status != ?", postID, model.CommentStatusBlocked)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count root comments failed: %w", err)
	}

	offset := (page - 1) * size
	query := db.Offset(int(offset)).Limit(int(size))

	if order == "hot" {
		query = query.Order("like_count DESC, created_at DESC")
	} else {
		query = query.Order("created_at DESC")
	}

	if err := query.Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("find root comments failed: %w", err)
	}

	return comments, total, nil
}

// GetSubRepliesPreview 批量预览多个根评论各自的前 N 条子回复 (基于窗口函数 ROW_NUMBER)
func (d *CommentDao) GetSubRepliesPreview(ctx context.Context, postID int64, rootIDs []int64, limitPerRoot int) (map[int64][]*model.Comment, error) {
	resMap := make(map[int64][]*model.Comment, len(rootIDs))
	if len(rootIDs) == 0 {
		return resMap, nil
	}

	sqlStr := `
		WITH ranked_replies AS (
			SELECT id, post_id, root_id, parent_id, author_id, reply_to_uid, content, like_count, reply_count, status, created_at, updated_at,
			       ROW_NUMBER() OVER(PARTITION BY root_id ORDER BY created_at ASC) as rn
			FROM post_comment
			WHERE post_id = ? AND root_id IN ? AND status != ?
		)
		SELECT id, post_id, root_id, parent_id, author_id, reply_to_uid, content, like_count, reply_count, status, created_at, updated_at
		FROM ranked_replies
		WHERE rn <= ?
		ORDER BY root_id, created_at ASC;
	`

	var replies []*model.Comment
	if err := d.db.WithContext(ctx).Raw(sqlStr, postID, rootIDs, model.CommentStatusBlocked, limitPerRoot).Scan(&replies).Error; err != nil {
		return nil, fmt.Errorf("scan preview sub replies failed: %w", err)
	}

	for _, r := range replies {
		resMap[r.RootID] = append(resMap[r.RootID], r)
	}

	return resMap, nil
}

// GetSubReplies 展开某根评论下的全部子回复 (分页，按时间正序排列对话流)
func (d *CommentDao) GetSubReplies(ctx context.Context, postID, rootID int64, page, size int64) ([]*model.Comment, int64, error) {
	var replies []*model.Comment
	var total int64

	db := d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("post_id = ? AND root_id = ? AND status != ?", postID, rootID, model.CommentStatusBlocked)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sub replies failed: %w", err)
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at ASC").Find(&replies).Error; err != nil {
		return nil, 0, fmt.Errorf("find sub replies failed: %w", err)
	}

	return replies, total, nil
}

// DeleteComment 软删除评论
func (d *CommentDao) DeleteComment(ctx context.Context, commentID, authorID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c model.Comment
		if err := tx.Where("id = ? AND author_id = ?", commentID, authorID).First(&c).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrNotFound
			}
			return err
		}

		if err := tx.Model(&model.Comment{}).Where("id = ?", commentID).Update("status", model.CommentStatusDeleted).Error; err != nil {
			return err
		}

		// 若为子评论，扣减对应根评论的 reply_count
		if c.RootID > 0 {
			_ = tx.Model(&model.Comment{}).Where("id = ? AND reply_count > 0", c.RootID).Update("reply_count", gorm.Expr("reply_count - 1")).Error
		}

		// 扣减帖子总评论数
		_ = tx.Model(&model.Post{}).Where("post_id = ? AND comment_count > 0", c.PostID).Update("comment_count", gorm.Expr("comment_count - 1")).Error

		return nil
	})
}

// IncrCommentLike 增量修改评论点赞数
func (d *CommentDao) IncrCommentLike(ctx context.Context, commentID int64, delta int) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// DeleteCommentsByPostID 批量软删除帖子下的所有评论
func (d *CommentDao) DeleteCommentsByPostID(ctx context.Context, postID int64) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("post_id = ?", postID).
		Update("status", model.CommentStatusDeleted).Error
}

