// FILE: internal/entity/subscription_entity.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string
type PaymentStatus string
type BillingPeriod string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"

	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "success" // CHANGED: Must match DB Enum 'success'
	PaymentStatusFailed  PaymentStatus = "failed"

	BillingPeriodMonthly BillingPeriod = "monthly"
	BillingPeriodYearly  BillingPeriod = "yearly"
)

type SubscriptionPlan struct {
	Id            uuid.UUID
	Name          string
	Slug          string
	Description   string
	Tagline       string // Subtitle for pricing modal (e.g., "Unlock AI Chat and Semantic Search")
	Price         float64
	TaxRate       float64
	BillingPeriod BillingPeriod
	// Storage Limits (cumulative)
	MaxNotebooks        int // Max folders allowed, -1 = unlimited
	MaxNotesPerNotebook int // Max notes per folder, -1 = unlimited
	// Daily Usage Limits (reset daily)
	AiChatDailyLimit         int // Max AI chat messages per day, 0 = disabled, -1 = unlimited
	SemanticSearchDailyLimit int // Max semantic searches per day, 0 = disabled, -1 = unlimited
	// Feature Flags (kept for backward compatibility)
	SemanticSearchEnabled bool
	AiChatEnabled         bool
	// Display Settings
	IsMostPopular bool // Show "Most Popular" badge
	IsActive      bool // Show in pricing modal
	SortOrder     int  // Display order

	// Relations
	Features []Feature
}

type UserSubscription struct {
	Id                    uuid.UUID
	UserId                uuid.UUID
	PlanId                uuid.UUID
	BillingAddressId      *uuid.UUID
	Status                SubscriptionStatus
	CurrentPeriodStart    time.Time
	CurrentPeriodEnd      time.Time
	PaymentStatus         PaymentStatus
	MidtransTransactionId *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// SubscriptionTransaction represents a view of a subscription transaction details
type SubscriptionTransaction struct {
	Id              uuid.UUID
	UserId          uuid.UUID
	UserEmail       string
	PlanName        string
	Amount          float64
	Status          SubscriptionStatus
	PaymentStatus   PaymentStatus
	CreatedAt       time.Time
	MidtransOrderId *string
}
