package mysql

import (
	"bluebell/models"
	"bluebell/pkg/errno"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var (
	ErrorInvalidID = errno.ErrorInvalidID
)

// GetCommunityList 查询社区列表数据
func GetCommunityList() (data []*models.CommunityDetail, err error) {
	sqlStr := "select community_id, community_name from community"
	
	// 初始化切片，防止查询为空时返回 nil
	if err = db.Select(&data, sqlStr); err != nil {
		// 如果查询为空（sql.ErrNoRows），在列表查询中通常不视为错误，返回空列表即可
		if err == sql.ErrNoRows {
			zap.L().Warn("there is no community in db")
			err = nil
		}
	}
	return
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
			zap.L().Warn("there is no community in db", zap.Int64("community_id", id))
			return nil, nil // 返回nil而不是error，让上层决定如何处理
		}
		// 其他数据库错误（如连接断开、SQL语法错误等）
		zap.L().Error("query community detail failed", zap.Error(err))
		return nil, err
	}
	return 
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
		return nil, err
	}

	communities = make([]*models.CommunityDetail, 0, len(ids))
	err = db.Select(&communities, db.Rebind(query), args...)
	return communities, err
}
