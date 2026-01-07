package model

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionPlanFeature struct {
	PlanId    uuid.UUID `gorm:"type:uuid;primaryKey"`
	FeatureId uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time `gorm:"default:now()"`
}

func (SubscriptionPlanFeature) TableName() string {
	return "subscription_plan_features"
}
