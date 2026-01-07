package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// NotificationType serves as a registry for event-to-notification mapping.
type NotificationType struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string         `gorm:"type:varchar(50);unique;not null" json:"code"`
	DisplayName string         `gorm:"type:varchar(100);not null" json:"display_name"`
	Template    string         `gorm:"type:text;not null" json:"template"`
	TargetType  string         `gorm:"type:varchar(20);not null" json:"target_type"` // e.g. "SELF", "ADMIN", "ROLE"
	TargetRole  string         `gorm:"type:varchar(50)" json:"target_role,omitempty"`
	Priority    string         `gorm:"type:varchar(10);default:'MEDIUM'" json:"priority"`
	Channels    datatypes.JSON `gorm:"type:jsonb;default:'[\"web\"]'" json:"channels"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// Notification stores the actual notification history.
type Notification struct {
	ID         uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID     uuid.UUID        `gorm:"type:uuid;not null;index:idx_notifications_user_created,priority:1;index:idx_notifications_user_unread,priority:1" json:"user_id"`
	ActorID    *uuid.UUID       `gorm:"type:uuid" json:"actor_id,omitempty"`
	TypeCode   string           `gorm:"type:varchar(50);not null;index:idx_notifications_type" json:"type_code"`
	Type       NotificationType `gorm:"foreignKey:TypeCode;references:Code;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	EntityType string           `gorm:"type:varchar(50);index:idx_notifications_entity,priority:1" json:"entity_type,omitempty"`
	EntityID   *uuid.UUID       `gorm:"type:uuid;index:idx_notifications_entity,priority:2" json:"entity_id,omitempty"`
	Title      string           `gorm:"type:varchar(200);not null" json:"title"`
	Message    string           `gorm:"type:text;not null" json:"message"`
	Metadata   datatypes.JSON   `gorm:"type:jsonb" json:"metadata,omitempty"`
	IsRead     bool             `gorm:"default:false;index:idx_notifications_user_unread,priority:2" json:"is_read"`
	ReadAt     *time.Time       `json:"read_at,omitempty"`
	CreatedAt  time.Time        `gorm:"default:CURRENT_TIMESTAMP;index:idx_notifications_user_created,priority:2" json:"created_at"`
}

// UserNotificationPreference stores user settings.
type UserNotificationPreference struct {
	UserID       uuid.UUID                   `gorm:"type:uuid;primaryKey" json:"user_id"`
	MutedTypes   datatypes.JSONSlice[string] `gorm:"type:text[]" json:"muted_types"`
	EmailEnabled bool                        `gorm:"default:true" json:"email_enabled"`
	PushEnabled  bool                        `gorm:"default:true" json:"push_enabled"`
	UpdatedAt    time.Time                   `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}
