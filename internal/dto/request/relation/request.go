package relationreq

// FollowRequest 关注/取关用户请求
type FollowRequest struct {
	TargetUserID int64 `json:"target_user_id" binding:"required"`
	Action       int8  `json:"action" binding:"oneof=0 1"` // 1:关注, 0:取关
}

// RelationListRequest 关注/粉丝列表分页请求
type RelationListRequest struct {
	UserID int64 `form:"user_id" binding:"required"`
	Page   int64 `form:"page"`
	Size   int64 `form:"size"`
}
