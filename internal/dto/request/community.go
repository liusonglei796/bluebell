package request

// CommunityDetailRequest 用于绑定获取社区详情的 URI 参数
type CommunityDetailRequest struct {
	ID int64 `uri:"id" binding:"required"`
}
