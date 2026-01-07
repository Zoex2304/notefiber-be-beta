// FILE: internal/entity/cancellation_entity.go
package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CancellationStatus represents the status of a cancellation request
type CancellationStatus string

const (
	CancellationStatusPending  CancellationStatus = "pending"
	CancellationStatusApproved CancellationStatus = "approved"
	CancellationStatusRejected CancellationStatus = "rejected"
)

// Cancellation represents a subscription cancellation request
type Cancellation struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SubscriptionID uuid.UUID `gorm:"type:uuid;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	Reason         string    `gorm:"type:text"`
	Status         string    `gorm:"type:varchar(50);default:'pending'"`
	AdminNotes     string    `gorm:"type:text"`
	EffectiveDate  time.Time `gorm:"not null"` // When the cancellation takes effect
	ProcessedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt   `gorm:"index"`
	Subscription   UserSubscription `gorm:"foreignKey:SubscriptionID"`
	User           User             `gorm:"foreignKey:UserID"`
}
