package notifresp

import "time"

// NotificationItem 单条通知展示结构
type NotificationItem struct {
	ID         string                 `json:"id"`
	ActorID    string                 `json:"actor_id"`
	ActorName  string                 `json:"actor_name"`
	ActionType int8                   `json:"action_type"`
	EntityType int8                   `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	ExtraInfo  map[string]interface{} `json:"extra_info,omitempty"`
	IsRead     bool                   `json:"is_read"`
	CreatedAt  time.Time              `json:"created_at"`
}

// UnreadCountResponse 未读红点计数统计
type UnreadCountResponse struct {
	Total  int64 `json:"total"`
	Reply  int64 `json:"reply"`
	Like   int64 `json:"like"`
	Follow int64 `json:"follow"`
	System int64 `json:"system"`
}

// NotificationListResponse 通知列表及未读数
type NotificationListResponse struct {
	Total         int64               `json:"total"`
	UnreadCount   int64               `json:"unread_count"`
	Notifications []*NotificationItem `json:"notifications"`
}
