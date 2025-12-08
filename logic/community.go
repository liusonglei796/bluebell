package logic

import (
	"bluebell/dao/mysql"
	"bluebell/models"
	"bluebell/pkg/errno"
)

// GetCommunityList 获取社区列表
func GetCommunityList() ([]*models.CommunityDetail, error) {
	// 直接调用 DAO 层的方法
	// 将来如果有复杂的业务逻辑（比如缓存、数据拼接）可以在这里添加
	return mysql.GetCommunityList()
}

// GetCommunityDetail 根据ID获取社区详情
func GetCommunityDetail(id int64) (*models.CommunityDetail, error) {
	// 直接调用 DAO 层查询数据库
	data, err := mysql.GetCommunityDetailByID(id)
	// 如果有自定义的无效ID错误，则向上传递
	if err != nil {
		// 区分不同类型的错误
		if err == errno.ErrorInvalidID {
			return nil, errno.ErrorInvalidID
		}
		// 其他情况视为查询失败
		return nil, errno.ErrorQueryFailed
	}
	return data, nil
}