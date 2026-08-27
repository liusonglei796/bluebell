package communityreq

// CommunityDetailRequest 用于绑定获取社区详情的 URI 参数
type CommunityDetailRequest struct {
	ID int64 `uri:"id" binding:"required"`
}

// CreateCommunityRequest 用于绑定创建社区的请求参数
type CreateCommunityRequest struct {
	Name         string `json:"name" binding:"required"`
	Introduction string `json:"introduction" binding:"required"`
}
