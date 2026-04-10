package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Vote 投票记录表
type Vote struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	PostID    int64     `gorm:"index;not null" json:"post_id"`
	UserID    int64     `gorm:"index;not null" json:"user_id"`
	Value     int8      `gorm:"not null"` // 1: 赞成, -1: 反对
	IsDeleted bool      `gorm:"default:false" json:"is_deleted"`
	CreateAt  time.Time `gorm:"not null" json:"create_at"`
	UpdateAt  time.Time `gorm:"not null" json:"update_at"`
}

func (Vote) TableName() string {
	return "votes"
}

// PostStat 帖子统计表
type PostStat struct {
	PostID    int64     `gorm:"primaryKey" json:"post_id"`
	UpVotes   int64     `gorm:"default:0" json:"up_votes"`
	DownVotes int64     `gorm:"default:0" json:"down_votes"`
	NetVotes  int64     `gorm:"default:0" json:"net_votes"` // 净投票数
	UpdateAt  time.Time `gorm:"not null" json:"update_at"`
}

func (PostStat) TableName() string {
	return "post_stats"
}

// VoteMessage RabbitMQ 消息结构
type VoteMessage struct {
	PostID   int64 `json:"post_id"`
	UserID   int64 `json:"user_id"`
	Value    int8  `json:"value"`     // 1: 赞成, -1: 反对
	CreateAt int64 `json:"create_at"` // 时间戳
}

// VoteConsumer 投票消费者
type VoteConsumer struct {
	db *gorm.DB
}

func NewVoteConsumer(db *gorm.DB) *VoteConsumer {
	return &VoteConsumer{db: db}
}

// Start 启动消费者
func (c *VoteConsumer) Start(ctx context.Context) error {
	// 这里应该对接 RabbitMQ消费者
	// 为简化，展示核心落盘逻辑
	return c.consumeLoop(ctx)
}

// consumeLoop 模拟消费循环
func (c *VoteConsumer) consumeLoop(ctx context.Context) error {
	// 实际项目中这里是从 RabbitMQ 获取消息
	// 为演示用，假设收到消息
	msg := VoteMessage{
		PostID:   123,
		UserID:   456,
		Value:    1,
		CreateAt: time.Now().Unix(),
	}
	return c.handleVote(ctx, msg)
}

// HandleVote 处理投票消息
func (c *VoteConsumer) HandleVote(ctx context.Context, msg VoteMessage) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 写入投票记录
		vote := Vote{
			PostID:    msg.PostID,
			UserID:    msg.UserID,
			Value:     msg.Value,
			IsDeleted: false,
			CreateAt:  time.Unix(msg.CreateAt, 0),
			UpdateAt:  time.Now(),
		}

		// 判断是否已存在投票记录（防止重复）
		var existing Vote
		err := tx.Where("post_id = ? AND user_id = ? AND is_deleted = ?",
			msg.PostID, msg.UserID, false).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// 新增投票
			if err := tx.Create(&vote).Error; err != nil {
				return fmt.Errorf("create vote failed: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("query vote failed: %w", err)
		} else {
			// 已存在，更新（投票修改场景）
			updateValue := existing.Value
			diff := msg.Value - updateValue // 差值
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"value":     msg.Value,
				"update_at": time.Now(),
			}).Error; err != nil {
				return fmt.Errorf("update vote failed: %w", err)
			}
			// 更新帖子统计（差量更新）
			return c.updatePostStatWithDiff(tx, msg.PostID, diff)
		}

		// 2. 更新帖子统计（全量更新）
		return c.updatePostStat(tx, msg.PostID)
	})
}

// updatePostStat 更新帖子统计（全量）
func (c *VoteConsumer) updatePostStat(tx *gorm.DB, postID int64) error {
	// 统计赞成票
	var upVotes int64
	tx.Model(&Vote{}).Where("post_id = ? AND value = 1 AND is_deleted = ?", postID, false).
		Count(&upVotes)

	// 统计反对票
	var downVotes int64
	tx.Model(&Vote{}).Where("post_id = ? AND value = -1 AND is_deleted = ?", postID, false).
		Count(&downVotes)

	netVotes := upVotes - downVotes

	// 使用 upsert 语法
	stats := PostStat{
		PostID:    postID,
		UpVotes:   upVotes,
		DownVotes: downVotes,
		NetVotes:  netVotes,
		UpdateAt:  time.Now(),
	}

	return tx.Clauses(
		"ON CONFLICT (post_id) DO UPDATE SET up_votes = EXCLUDED.up_votes, " +
			"down_votes = EXCLUDED.down_votes, net_votes = EXCLUDED.net_votes, " +
			"update_at = EXCLUDED.update_at",
	).Create(&stats).Error
}

// updatePostStatWithDiff 差量更新（性能更好）
func (c *VoteConsumer) updatePostStatWithDiff(tx *gorm.DB, postID int64, diff int8) error {
	// diff 为 1: 新增赞成 或 反对->赞成 +1
	// diff 为 -1: 新增反对 或 赞成->反对 -1
	// diff 为 0: 不变

	setters := make(map[string]interface{})
	if diff > 0 {
		setters["up_votes"] = gorm.Expr("up_votes + ?", diff)
		// 反对变赞成，净票数 +2
		if diff == 2 {
			setters["down_votes"] = gorm.Expr("down_votes - 1")
			setters["net_votes"] = gorm.Expr("net_votes + 2")
		}
	} else if diff < 0 {
		setters["down_votes"] = gorm.Expr("down_votes + ?", -diff)
		// 赞成变反对，净票数 -2
		if diff == -2 {
			setters["up_votes"] = gorm.Expr("up_votes - 1")
			setters["net_votes"] = gorm.Expr("net_votes - 2")
		}
	}

	setters["net_votes"] = gorm.Expr("net_votes + ?", diff)
	setters["update_at"] = time.Now()

	return tx.Model(&PostStat{}).Where("post_id = ?", postID).Updates(setters).Error
}

// ParseMessage 解析 RabbitMQ 消息
func ParseMessage(data []byte) (VoteMessage, error) {
	var msg VoteMessage
	err := json.Unmarshal(data, &msg)
	return msg, err
}
