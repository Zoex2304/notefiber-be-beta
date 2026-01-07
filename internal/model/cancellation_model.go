// FILE: internal/model/cancellation_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Cancellation GORM model for subscription cancellation requests
type Cancellation struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SubscriptionID uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"`
	Reason         string    `gorm:"type:text"`
	Status         string    `gorm:"type:varchar(50);default:'pending';index"` // pending, approved, rejected
	AdminNotes     string    `gorm:"type:text"`
	EffectiveDate  time.Time `gorm:"not null"` // When the cancellation takes effect (usually current_period_end)
	ProcessedAt    *time.Time
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	// Relations
	Subscription UserSubscription `gorm:"foreignKey:SubscriptionID"`
	User         User             `gorm:"foreignKey:UserID"`
}

func (Cancellation) TableName() string {
	return "cancellations"
}
