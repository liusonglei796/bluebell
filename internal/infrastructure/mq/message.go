package mq

// VoteMessage 投票异步计数消息
// 通过 RabbitMQ 异步消费后更新 Redis 中的投票计数
type VoteMessage struct {
	MsgID  string `json:"msg_id"`  // 消息唯一 ID (如 Snowflake ID)
	PostID string `json:"post_id"`
	UserID string `json:"user_id"`
	Action int    `json:"action"` // 1=upvote, -1=downvote
}

// SyncMessage ES 搜索同步消息
// 通过 RabbitMQ 消费后同步帖子数据到 Elasticsearch
type SyncMessage struct {
	PostID      string `json:"post_id"`
	AuthorID    int64  `json:"author_id"`
	CommunityID int64  `json:"community_id"`
	PostTitle   string `json:"post_title"`
	Content     string `json:"content"`
	Status      int8   `json:"status"` // post status: 1=published
	CreatedAt   string `json:"created_at"`
	Action      string `json:"action"` // "index" or "delete"
}

// ActivityMessage 用户动态消息
type ActivityMessage struct {
	UserID     int64  `json:"user_id"`
	Type       string `json:"type"` // "post_created", "vote_up", "vote_down", "follow", etc.
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name"`
	Timestamp  int64  `json:"timestamp"`
}
