package model

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionPlan struct {
	Id            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name          string    `gorm:"type:varchar(255);not null"`
	Slug          string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Description   string    `gorm:"type:text"`
	Tagline       string    `gorm:"type:text"` // Subtitle for pricing modal
	Price         float64   `gorm:"type:decimal(10,2);not null"`
	TaxRate       float64   `gorm:"type:decimal(5,4);default:0"`
	BillingPeriod string    `gorm:"type:billing_period;not null"`
	// Storage Limits
	MaxNotebooks        int `gorm:"default:3"`  // -1 = unlimited
	MaxNotesPerNotebook int `gorm:"default:10"` // -1 = unlimited
	// Daily Usage Limits
	AiChatDailyLimit         int `gorm:"default:0"` // 0 = disabled, -1 = unlimited
	SemanticSearchDailyLimit int `gorm:"default:0"` // 0 = disabled, -1 = unlimited
	// Feature Flags (backward compatibility)
	SemanticSearchEnabled bool `gorm:"default:false"`
	AiChatEnabled         bool `gorm:"default:false"`
	// Display Settings
	IsMostPopular bool `gorm:"default:false"`
	IsActive      bool `gorm:"default:true"`
	SortOrder     int  `gorm:"default:0"`

	// Relations
	Features []*Feature `gorm:"many2many:subscription_plan_features;joinForeignKey:plan_id;joinReferences:feature_id"`
}

func (SubscriptionPlan) TableName() string {
	return "subscription_plans"
}

type UserSubscription struct {
	Id                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId                uuid.UUID  `gorm:"type:uuid;not null;index"`
	PlanId                uuid.UUID  `gorm:"type:uuid;not null;index"`
	BillingAddressId      *uuid.UUID `gorm:"type:uuid;index"`
	Status                string     `gorm:"type:varchar(50);not null"`
	CurrentPeriodStart    time.Time  `gorm:"not null"`
	CurrentPeriodEnd      time.Time  `gorm:"not null"`
	PaymentStatus         string     `gorm:"type:varchar(50);not null"`
	MidtransTransactionId *string    `gorm:"type:varchar(255)"`
	CreatedAt             time.Time  `gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime"`
}

func (UserSubscription) TableName() string {
	return "user_subscriptions"
}
