package mysql

import (
	"bluebell/models"
	"database/sql"
	"fmt"
	"strings"
	"github.com/jmoiron/sqlx"
)

// CreatePost 创建帖子
// DAO层只返回错误,不打印日志,由上层统一处理
func CreatePost(post *models.Post) (err error) {
	sqlStr := `INSERT INTO post(post_id, title, content, author_id, community_id) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(sqlStr, post.ID, post.Title, post.Content, post.AuthorID, post.CommunityID)
	if err != nil {
		return fmt.Errorf("insert post failed: %w", err)
	}
	return nil
}

// GetPostByID 根据帖子ID查询帖子详情
// DAO层只返回错误,不打印日志,由上层统一处理
func GetPostByID(pid int64) (post *models.Post, err error) {
	post = new(models.Post)
	sqlStr := `SELECT post_id, title, content, author_id, community_id, status, create_time
				FROM post
				WHERE post_id = ?`
	err = db.Get(post, sqlStr, pid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 查不到数据返回nil,不是错误
		}
		return nil, fmt.Errorf("query post by id failed: %w", err)
	}
	return
}

// GetPostListByCommunityID 根据社区ID查询帖子列表
func GetPostListByCommunityID(communityID int64, page, size int64) (posts []*models.Post, err error) {
	sqlStr := `SELECT post_id, title, content, author_id, community_id, status, create_time
				FROM post
				WHERE community_id = ?
				ORDER BY create_time
				DESC limit ?,?`
	posts = make([]*models.Post, 0, size)
	err = db.Select(&posts, sqlStr, communityID, (page-1)*size, size)
	if err == sql.ErrNoRows {
		return posts, nil // 查不到数据不是错误,返回空切片
	}
	if err != nil {
		return nil, fmt.Errorf("query post list by community failed: %w", err)
	}
	return
}

// GetPostListByIDs 根据给定的ID列表查询帖子的详细数据
func GetPostListByIDs(ids []string) (posts []*models.Post, err error) {
	// 1. 构造 SQL 查询语句
	// 核心1: 使用 `IN (?)` 作为占位符,后续 `sqlx.In` 会将其展开
	// 核心2: 使用 `ORDER BY FIND_IN_SET(post_id, ?)` 保证查询结果的顺序与传入的 ids 一致
	sqlStr := `SELECT post_id, title, content, author_id, community_id, create_time
               FROM post
               WHERE post_id IN (?)
               ORDER BY FIND_IN_SET(post_id, ?)`

	// 2. 使用 sqlx.In 来处理 `IN` 查询
	query, args, err := sqlx.In(sqlStr, ids, strings.Join(ids, ","))
	if err != nil {
		return nil, fmt.Errorf("sqlx.In failed: %w", err)
	}

	// 3. 使用 db.Rebind 重新绑定SQL语句
	// 不同数据库对占位符的处理方式不同(MySQL是'?', PostgreSQL是'$1')
	query = db.Rebind(query)

	// 4. 执行查询
	err = db.Select(&posts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query post list by ids failed: %w", err)
	}
	return
}