package notifreq

// NotificationListRequest 通知列表请求参数
type NotificationListRequest struct {
	ActionType int8  `form:"action_type"` // 0为全部, >0为指定类型
	Page       int64 `form:"page"`
	Size       int64 `form:"size"`
}

// MarkReadRequest 标记已读请求
type MarkReadRequest struct {
	NotificationIDs []int64 `json:"notification_ids"`
	ActionType      int8    `json:"action_type"` // 若传 >0 则该分类全部标记已读
	All             bool    `json:"all"`         // 全局全部标记已读
}
