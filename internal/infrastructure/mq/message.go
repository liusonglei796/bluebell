package mq

// VoteMessage 投票异步计数消息
// 通过 RabbitMQ 异步消费后更新 Redis 中的投票计数
type VoteMessage struct {
	PostID string `json:"post_id"`
	UserID string `json:"user_id"`
	Action int    `json:"action"` // 1=upvote, -1=downvote
}

// AuditMessage 审核消息体
// 通过 RabbitMQ 发送到 audit.exchange 进行 AI 内容审核
type AuditMessage struct {
	PostID   string `json:"post_id"`   // 帖子审核时填写
	RemarkID uint   `json:"remark_id"` // 评论审核时填写（gorm.Model 的 ID）
	Title    string `json:"title"`
	Content  string `json:"content"`
	Type     string `json:"type"` // "post" or "remark"
	AuthorID int64  `json:"author_id"`
}

// SyncMessage ES 搜索同步消息
// 通过 RabbitMQ 消费后同步帖子数据到 Elasticsearch
type SyncMessage struct {
	PostID      string `json:"post_id"`
	AuthorID    int64  `json:"author_id"`
	CommunityID int64  `json:"community_id"`
	PostTitle   string `json:"post_title"`
	Content     string `json:"content"`
	Status      int8   `json:"status"`
	CreatedAt   string `json:"created_at"`
	Action      string `json:"action"` // "index" or "delete"
}
