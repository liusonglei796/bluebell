package service

import (
	"context"
	"strconv"

	"bluebell/internal/dao/mysql"
	tagreq "bluebell/internal/dto/request/tag"
	tagresp "bluebell/internal/dto/response/tag"
	"bluebell/internal/model"

	"go.uber.org/zap"
)

// TagService 标签服务
type TagService struct {
	tagDao       *mysql.TagDao
	communityDao *mysql.CommunityDao
	userDao      *mysql.UserDao
}

// NewTagService 创建标签服务实例
func NewTagService(tagDao *mysql.TagDao, communityDao *mysql.CommunityDao, userDao *mysql.UserDao) *TagService {
	return &TagService{
		tagDao:       tagDao,
		communityDao: communityDao,
		userDao:      userDao,
	}
}

// CreateTag 创建社区标签
func (s *TagService) CreateTag(ctx context.Context, userID int64, p *tagreq.CreateTagRequest) (*tagresp.TagResponse, error) {
	// 1. 校验用户是否存在
	if s.userDao != nil {
		user, err := s.userDao.CheckUserExistsByID(ctx, userID)
		if err != nil {
			zap.L().Error("userDao.CheckUserExistsByID failed", zap.Int64("user_id", userID), zap.Error(err))
			return nil, model.Wrap(model.ErrServerBusy, err)
		}
		if user == nil {
			return nil, model.ErrNeedLogin
		}
	}

	// 2. 校验社区是否存在
	if s.communityDao != nil {
		community, err := s.communityDao.GetCommunityDetailByID(ctx, p.CommunityID)
		if err != nil || community == nil {
			return nil, model.ErrNotFound
		}
	}

	tag, err := s.tagDao.CreateTag(ctx, p.CommunityID, p.Name)
	if err != nil {
		zap.L().Error("tagDao.CreateTag failed", zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	return &tagresp.TagResponse{
		ID:          strconv.FormatUint(tag.ID, 10),
		CommunityID: strconv.FormatInt(tag.CommunityID, 10),
		Name:        tag.Name,
		PostCount:   0,
		CreatedAt:   tag.CreatedAt,
	}, nil
}

// GetCommunityTags 获取社区下的所有标签
func (s *TagService) GetCommunityTags(ctx context.Context, communityID int64) ([]*tagresp.TagResponse, error) {
	tags, err := s.tagDao.GetCommunityTags(ctx, communityID)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	res := make([]*tagresp.TagResponse, len(tags))
	for i, t := range tags {
		res[i] = &tagresp.TagResponse{
			ID:          strconv.FormatUint(t.ID, 10),
			CommunityID: strconv.FormatInt(t.CommunityID, 10),
			Name:        t.Name,
			PostCount:   t.PostCount,
			CreatedAt:   t.CreatedAt,
		}
	}
	return res, nil
}
