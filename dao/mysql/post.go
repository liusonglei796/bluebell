package mysql

import (
	"bluebell/models"
	"database/sql"
	"strings"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// CreatePost 创建帖子
func CreatePost(post *models.Post) (err error) {
	sqlStr := `INSERT INTO post(post_id, title, content, author_id, community_id) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(sqlStr, post.ID, post.Title, post.Content, post.AuthorID, post.CommunityID)
	if err != nil {
		zap.L().Error("insert post failed", zap.Error(err))
		return err
	}
	return nil
}

// GetPostByID 根据帖子ID查询帖子详情
func GetPostByID(pid int64) (post *models.Post, err error) {
	post = new(models.Post)
	sqlStr := `SELECT post_id, title, content, author_id, community_id, status, create_time 
				FROM post 
				WHERE post_id = ?`
	err = db.Get(post, sqlStr, pid)
	if err != nil {
		if err == sql.ErrNoRows {
			zap.L().Warn("there is no post in db", zap.Int64("pid", pid))
			return nil, nil // 返回nil而不是error，让上层决定如何处理
		}
		zap.L().Error("query post failed", zap.Int64("pid", pid), zap.Error(err))
		return nil, err
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
		zap.L().Warn("there is no post in db")
		err = nil
	}
	return
}

// GetPostListByIDs 根据给定的ID列表查询帖子的详细数据
func GetPostListByIDs(ids []string) (posts []*models.Post, err error) {
	// 1. 构造 SQL 查询语句
	// 核心1: 使用 `IN (?)` 作为占位符，后续 `sqlx.In` 会将其展开。
	// 核心2: 使用 `ORDER BY FIND_IN_SET(post_id, ?)` 来保证查询结果的顺序
	//       与传入的 ids 顺序一致。
	sqlStr := `SELECT post_id, title, content, author_id, community_id, create_time
               FROM post
               WHERE post_id IN (?)
               ORDER BY FIND_IN_SET(post_id, ?)`

	// 2. 使用 sqlx.In 来处理 `IN` 查询
	// 它会返回一个带 `?` 占位符的查询语句和对应的参数列表
	query, args, err := sqlx.In(sqlStr, ids, strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}

	// 3. 使用 db.Rebind 重新绑定SQL语句
	// 这是因为不同的数据库（MySQL, PostgreSQL）对占位符的处理方式不同
	// (例如, MySQL 是 '?', PostgreSQL 是 '$1', '$2')
	// db.Rebind 会自动将其转换为当前数据库驱动支持的格式。
	query = db.Rebind(query)

	// 4. 执行查询
	// 注意：Select 的目标参数必须是指针的切片 (e.g., &posts)，
	// 并且 args 后面必须跟 ... 来解包切片。
	err = db.Select(&posts, query, args...)
	return
}