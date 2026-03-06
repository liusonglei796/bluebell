package serviceinterface

import (
	"bluebell/internal/dto/response"
	"context"
)

// CommunityService 社区业务逻辑服务接口
type CommunityService interface {
	// GetCommunityList 获取社区列表
	GetCommunityList(ctx context.Context) ([]*response.CommunityResponse, error)

	// GetCommunityDetail 根据ID获取社区详情
	GetCommunityDetail(ctx context.Context, id int64) (*response.CommunityResponse, error)
}
