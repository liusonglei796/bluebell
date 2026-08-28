package mysql

import (
	"context"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// NotificationDao 通知数据访问对象
type NotificationDao struct {
	db *gorm.DB
}

// NewNotificationDao 创建通知 DAO 实例
func NewNotificationDao(db *gorm.DB) *NotificationDao {
	return &NotificationDao{db: db}
}

// CreateNotificationInTx 在事务中创建通知记录
func (d *NotificationDao) CreateNotificationInTx(ctx context.Context, tx *gorm.DB, notif *model.UserNotification) error {
	db := d.db
	if tx != nil {
		db = tx
	}
	if err := db.WithContext(ctx).Create(notif).Error; err != nil {
		return fmt.Errorf("create notification failed: %w", err)
	}
	return nil
}

// CreateNotification 创建通知记录
func (d *NotificationDao) CreateNotification(ctx context.Context, notif *model.UserNotification) error {
	return d.CreateNotificationInTx(ctx, nil, notif)
}

// GetNotifications 分页获取用户的通知列表
func (d *NotificationDao) GetNotifications(ctx context.Context, recipientID int64, actionType int8, page, size int64) ([]*model.UserNotification, int64, error) {
	var notifs []*model.UserNotification
	var total int64

	db := d.db.WithContext(ctx).Model(&model.UserNotification{}).
		Where("recipient_id = ? AND is_deleted = 0", recipientID)

	if actionType > 0 {
		db = db.Where("action_type = ?", actionType)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at DESC").Find(&notifs).Error; err != nil {
		return nil, 0, err
	}

	return notifs, total, nil
}

// GetUnreadCount 获取未读总数
func (d *NotificationDao) GetUnreadCount(ctx context.Context, recipientID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.UserNotification{}).
		Where("recipient_id = ? AND is_read = 0 AND is_deleted = 0", recipientID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// MarkAsRead 标记通知为已读
func (d *NotificationDao) MarkAsRead(ctx context.Context, recipientID int64, notifIDs []int64, actionType int8, all bool) error {
	db := d.db.WithContext(ctx).Model(&model.UserNotification{}).
		Where("recipient_id = ? AND is_read = 0", recipientID)

	if all {
		// 全部已读
		return db.Update("is_read", 1).Error
	}

	if actionType > 0 {
		return db.Where("action_type = ?", actionType).Update("is_read", 1).Error
	}

	if len(notifIDs) > 0 {
		return db.Where("id IN ?", notifIDs).Update("is_read", 1).Error
	}

	return nil
}
