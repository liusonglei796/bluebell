package mysql

import (
	"bluebell/models"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// GetCommunityList 查询社区列表数据
func GetCommunityList() (data []*models.CommunityDetail, err error) {
	sqlStr := "select community_id, community_name from community"
	
	// 初始化切片，防止查询为空时返回 nil
	if err = db.Select(&data, sqlStr); err != nil {
		// 如果查询为空（sql.ErrNoRows），在列表查询中通常不视为错误，返回空列表即可
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query community list failed: %w", err)
	}
	return data, nil
}

// GetCommunityDetailByID 根据ID查询数据库详情
func GetCommunityDetailByID(id int64) (community *models.CommunityDetail, err error) {
	community = new(models.CommunityDetail)
	
	// 编写 SQL 语句
	sqlStr := `select 
				community_id, community_name, introduction, create_time 
			   from community 
			   where community_id = ?`
	
	// 执行查询
	// 注意：Get 方法用于查询单条数据，如果查不到会返回 error
	err = db.Get(community, sqlStr, id)

	if err != nil {
		// 特殊处理：如果没有查到数据
		if err == sql.ErrNoRows {
			return nil, nil // 返回nil而不是error，让上层决定如何处理
		}
		// 其他数据库错误（如连接断开、SQL语法错误等）
		return nil, fmt.Errorf("query community detail failed: %w", err)
	}
	return community, nil 
}

// GetCommunitiesByIDs 根据社区ID列表批量获取社区信息
func GetCommunitiesByIDs(ids []int64) (communities []*models.CommunityDetail, err error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 构造 IN 查询
	query, args, err := sqlx.In(`SELECT
								community_id, community_name, introduction, create_time
								FROM community
								WHERE community_id IN (?)`, ids)
	if err != nil {
		return nil, fmt.Errorf("sqlx.In failed: %w", err)
	}

	communities = make([]*models.CommunityDetail, 0, len(ids))
	err = db.Select(&communities, db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("query communities by ids failed: %w", err)
	}
	return communities, nil
}
