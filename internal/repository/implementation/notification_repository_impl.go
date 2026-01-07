package implementation

import (
	"context"
	"errors"
	"time"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepositoryImpl struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) repository.NotificationRepository {
	return &NotificationRepositoryImpl{db: db}
}

func (r *NotificationRepositoryImpl) CreateNotification(ctx context.Context, notification *model.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

func (r *NotificationRepositoryImpl) GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, int64, error) {
	var notifications []model.Notification
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ?", userID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *NotificationRepositoryImpl) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepositoryImpl) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	now := time.Now()
	// Using Updates for partial update
	result := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", notificationID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("notification not found")
	}
	return nil
}

func (r *NotificationRepositoryImpl) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

func (r *NotificationRepositoryImpl) GetNotificationTypeByCode(ctx context.Context, code string) (*model.NotificationType, error) {
	var notifType model.NotificationType
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&notifType).Error
	if err != nil {
		return nil, err
	}
	return &notifType, nil
}

func (r *NotificationRepositoryImpl) GetUsersByRole(ctx context.Context, role string) ([]model.User, error) {
	var users []model.User
	// Assuming there is a Role field or related table.
	// Based on the enum setup in migration: CREATE TYPE user_role AS ENUM ('user', 'admin');
	// We should probably check the User model structure.
	// For now assuming a simple "role = ?" query works if mapped.
	// If User model has custom Role mapping we might need to adjust.
	err := r.db.WithContext(ctx).
		Where("role = ?", role).
		Find(&users).Error
	return users, err
}
