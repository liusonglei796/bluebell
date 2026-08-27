package model

import "time"

// ProcessedEvent 消费幂等去重记录
type ProcessedEvent struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	EventID       string    `gorm:"column:event_id;size:64;not null;uniqueIndex:uk_event_consumer,priority:1" json:"event_id"`
	ConsumerGroup string    `gorm:"column:consumer_group;size:64;not null;uniqueIndex:uk_event_consumer,priority:2" json:"consumer_group"`
	EventType     string    `gorm:"column:event_type;size:128;not null" json:"event_type"`
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_created_at" json:"created_at"`
}

func (ProcessedEvent) TableName() string {
	return "processed_events"
}
