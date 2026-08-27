package model

import "time"

// 通知行为类型常量
const (
	ActionTypeLikePost     int8 = 1 // 点赞了帖子
	ActionTypeLikeComment  int8 = 2 // 点赞了评论
	ActionTypeReplyPost    int8 = 3 // 评论了帖子
	ActionTypeReplyComment int8 = 4 // 回复了评论
	ActionTypeFollow       int8 = 5 // 关注了我
	ActionTypeSystem       int8 = 6 // 系统通知
)

// 实体类型常量
const (
	EntityTypePost    int8 = 1
	EntityTypeComment int8 = 2
	EntityTypeUser    int8 = 3
)

// UserNotification 用户通知模型
type UserNotification struct {
	ID          int64     `gorm:"primaryKey;column:id" json:"id"`
	RecipientID int64     `gorm:"column:recipient_id;not null;index:idx_recipient_list,priority:1;index:idx_recipient_type,priority:1;index:idx_recipient_unread,priority:1" json:"recipient_id"`
	ActorID     int64     `gorm:"column:actor_id;not null" json:"actor_id"`
	ActionType  int8      `gorm:"column:action_type;not null;index:idx_recipient_type,priority:2" json:"action_type"`
	EntityType  int8      `gorm:"column:entity_type;not null" json:"entity_type"`
	EntityID    int64     `gorm:"column:entity_id;not null" json:"entity_id"`
	ExtraInfo   string    `gorm:"column:extra_info;type:json" json:"extra_info"`
	IsRead      int8      `gorm:"column:is_read;not null;default:0;index:idx_recipient_unread,priority:2" json:"is_read"`
	IsDeleted   int8      `gorm:"column:is_deleted;not null;default:0;index:idx_recipient_list,priority:2;index:idx_recipient_type,priority:3;index:idx_recipient_unread,priority:3" json:"is_deleted"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_recipient_list,priority:3;index:idx_recipient_type,priority:4" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updated_at"`

	// 动态填充对象
	Actor *User `gorm:"-" json:"actor,omitempty"`
}

func (UserNotification) TableName() string {
	return "user_notification"
}
