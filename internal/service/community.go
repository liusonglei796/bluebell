package service

import (
	"context"
	"strconv"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dto/response/community"
	"bluebell/internal/model"

	"go.uber.org/zap"
)

// CommunityService 社区业务逻辑服务
type CommunityService struct {
	communityDao *mysql.CommunityDao
	userDao      *mysql.UserDao
}

// NewCommunityService 创建社区服务实例
func NewCommunityService(communityDao *mysql.CommunityDao, userDao *mysql.UserDao) *CommunityService {
	return &CommunityService{
		communityDao: communityDao,
		userDao:      userDao,
	}
}

// toResponse 将 model.Community 转换为 communityResp.Response
func toResponse(c *model.Community) *communityResp.Response {
	return &communityResp.Response{
		ID:           strconv.FormatInt(int64(c.ID), 10),
		Name:         c.CommunityName,
		Introduction: c.Introduction,
		CreateTime:   c.CreatedAt,
	}
}

// GetCommunityList 获取社区列表
func (s *CommunityService) GetCommunityList(ctx context.Context) ([]*communityResp.Response, error) {
	data, err := s.communityDao.GetCommunityList(ctx)
	if err != nil {
		zap.L().Error("communityDao.GetCommunityList failed", zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	result := make([]*communityResp.Response, 0, len(data))
	for _, c := range data {
		result = append(result, toResponse(c))
	}
	return result, nil
}

// GetCommunityDetail 根据ID获取社区详情
func (s *CommunityService) GetCommunityDetail(ctx context.Context, id int64) (*communityResp.Response, error) {
	data, err := s.communityDao.GetCommunityDetailByID(ctx, id)
	if err != nil {
		zap.L().Error("communityDao.GetCommunityDetailByID failed",
			zap.Int64("community_id", id),
			zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	if data == nil {
		return nil, model.ErrNotFound
	}

	return toResponse(data), nil
}

// CreateCommunity 创建社区（仅管理员）
func (s *CommunityService) CreateCommunity(ctx context.Context, name, introduction string, userID int64) error {
	// 1. 校验用户角色是否为管理员
	user, err := s.userDao.CheckUserExistsByID(ctx, userID)
	if err != nil {
		zap.L().Error("userDao.CheckUserExistsByID failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}
	if user == nil || !user.IsAdmin() {
		return model.ErrForbidden
	}

	// 2. 创建社区
	community := &model.Community{
		CommunityName: name,
		Introduction:  introduction,
	}
	if err := s.communityDao.CreateCommunity(ctx, community); err != nil {
		zap.L().Error("communityDao.CreateCommunity failed",
			zap.String("community_name", name),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	return nil
}
