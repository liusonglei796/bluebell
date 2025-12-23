package logic

import (
	"bluebell/dao/mysql"
	"bluebell/models"
	"bluebell/pkg/errorx"

	"go.uber.org/zap"
)

// GetCommunityList 获取社区列表
func GetCommunityList() ([]*models.CommunityDetail, error) {
	// 调用 DAO 层的方法
	data, err := mysql.GetCommunityList()
	if err != nil {
		// 数据库错误：记录日志并返回系统错误
		zap.L().Error("mysql.GetCommunityList failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}
	return data, nil
}

// GetCommunityDetail 根据ID获取社区详情
func GetCommunityDetail(id int64) (*models.CommunityDetail, error) {
	// 调用 DAO 层查询数据库
	data, err := mysql.GetCommunityDetailByID(id)
	if err != nil {
		// 数据库错误：记录日志并返回系统错误
		zap.L().Error("mysql.GetCommunityDetailByID failed",
			zap.Int64("community_id", id),
			zap.Error(err),
		)
		return nil, errorx.ErrServerBusy
	}

	// DAO 层返回 nil 表示未找到数据（参见 community.go:48）
	if data == nil {
		// 业务错误：社区不存在
		return nil, errorx.ErrNotFound
	}

	return data, nil
}
