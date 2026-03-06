package serviceinterface

import (
	"bluebell/internal/dto/request"
	"bluebell/internal/dto/response"
	"context"
)

// PostService 帖子业务逻辑服务接口
type PostService interface {
	// CreatePost 创建帖子，返回新创建的帖子ID
	CreatePost(ctx context.Context, p *request.CreatePostRequest, authorID int64) (postID int64, err error)

	// GetPostByID 查询单个帖子详情
	GetPostByID(ctx context.Context, pid int64) (*response.PostDetailResponse, error)

	// GetPostList 获取帖子列表
	GetPostList(ctx context.Context, p *request.PostListRequest) ([]*response.PostDetailResponse, error)

	// GetCommunityPostList 根据社区ID获取帖子列表
	GetCommunityPostList(ctx context.Context, p *request.PostListRequest) ([]*response.PostDetailResponse, error)

	// DeletePost 删除帖子（软删除）
	DeletePost(ctx context.Context, postID int64, userID int64) error
}
