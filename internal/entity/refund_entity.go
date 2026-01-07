package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefundStatus represents the status of a refund request
type RefundStatus string

const (
	RefundStatusPending  RefundStatus = "pending"
	RefundStatusApproved RefundStatus = "approved"
	RefundStatusRejected RefundStatus = "rejected"
)

type Refund struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SubscriptionID uuid.UUID `gorm:"type:uuid;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	Amount         float64   `gorm:"type:decimal(10,2);not null"`
	Reason         string    `gorm:"type:text"`
	Status         string    `gorm:"type:varchar(50);default:'pending'"`
	AdminNotes     string    `gorm:"type:text"`
	ProcessedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt   `gorm:"index"`
	Subscription   UserSubscription `gorm:"foreignKey:SubscriptionID"`
	User           User             `gorm:"foreignKey:UserID"`
}
