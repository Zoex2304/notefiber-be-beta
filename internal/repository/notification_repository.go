package repository

import (
	"context"

	"ai-notetaking-be/internal/model"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	// Notification Operations
	CreateNotification(ctx context.Context, notification *model.Notification) error
	GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

	// Registry Operations
	GetNotificationTypeByCode(ctx context.Context, code string) (*model.NotificationType, error)
	GetUsersByRole(ctx context.Context, role string) ([]model.User, error) // Helper to resolve targets
}
