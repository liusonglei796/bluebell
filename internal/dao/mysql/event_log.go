package mysql

import (
	"context"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// EventLogDao 消费去重记录 DAO
type EventLogDao struct {
	db *gorm.DB
}

// NewEventLogDao 创建去重 DAO 实例
func NewEventLogDao(db *gorm.DB) *EventLogDao {
	return &EventLogDao{db: db}
}

// InsertProcessedEvent 在给定事务中插入消费记录
func (d *EventLogDao) InsertProcessedEvent(ctx context.Context, tx *gorm.DB, eventID, consumerGroup, eventType string) error {
	log := &model.ProcessedEvent{
		EventID:       eventID,
		ConsumerGroup: consumerGroup,
		EventType:     eventType,
	}

	db := d.db
	if tx != nil {
		db = tx
	}

	if err := db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("insert processed event failed: %w", err)
	}
	return nil
}
