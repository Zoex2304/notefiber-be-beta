// FILE: internal/dto/cancellation_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- User-Side Cancellation Request ---

// UserCancellationRequest for user submitting a cancellation
type UserCancellationRequest struct {
	SubscriptionId uuid.UUID `json:"subscription_id" validate:"required"`
	Reason         string    `json:"reason" validate:"required,min=10"`
}

// UserCancellationResponse after cancellation request submitted
type UserCancellationResponse struct {
	CancellationId string `json:"cancellation_id"`
	Status         string `json:"status"`
	Message        string `json:"message"`
}

// --- User's Cancellation List ---

// UserCancellationListResponse for user's cancellation history
type UserCancellationListResponse struct {
	Id             uuid.UUID `json:"id"`
	SubscriptionId uuid.UUID `json:"subscription_id"`
	PlanName       string    `json:"plan_name"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	EffectiveDate  time.Time `json:"effective_date"`
	CreatedAt      time.Time `json:"created_at"`
}

// --- Admin-Side Cancellation Management ---

// AdminCancellationListResponse for admin view of cancellations
type AdminCancellationListResponse struct {
	Id            uuid.UUID                         `json:"id"`
	User          AdminCancellationUserInfo         `json:"user"`
	Subscription  AdminCancellationSubscriptionInfo `json:"subscription"`
	Reason        string                            `json:"reason"`
	Status        string                            `json:"status"`
	AdminNotes    string                            `json:"admin_notes,omitempty"`
	EffectiveDate time.Time                         `json:"effective_date"`
	CreatedAt     time.Time                         `json:"created_at"`
	ProcessedAt   *time.Time                        `json:"processed_at,omitempty"`
}

// AdminCancellationUserInfo embedded user info for admin view
type AdminCancellationUserInfo struct {
	Id       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
}

// AdminCancellationSubscriptionInfo embedded subscription info
type AdminCancellationSubscriptionInfo struct {
	Id               uuid.UUID `json:"id"`
	PlanName         string    `json:"plan_name"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
}

// AdminProcessCancellationRequest for admin processing a cancellation
type AdminProcessCancellationRequest struct {
	Action     string `json:"action" validate:"required,oneof=approve reject"`
	AdminNotes string `json:"admin_notes,omitempty"`
}

// AdminProcessCancellationResponse after admin processes cancellation
type AdminProcessCancellationResponse struct {
	CancellationId string    `json:"cancellation_id"`
	Status         string    `json:"status"`
	EffectiveDate  time.Time `json:"effective_date"`
	ProcessedAt    time.Time `json:"processed_at"`
}

// --- Subscription Validation (Expiration Check) ---

// SubscriptionValidationResponse for checking subscription validity
type SubscriptionValidationResponse struct {
	IsValid          bool       `json:"is_valid"`
	Status           string     `json:"status"` // active, grace_period, expired
	RenewalRequired  bool       `json:"renewal_required"`
	CurrentPeriodEnd time.Time  `json:"current_period_end,omitempty"`
	DaysRemaining    int        `json:"days_remaining,omitempty"`
	GracePeriodEnd   *time.Time `json:"grace_period_end,omitempty"`
	PlanName         string     `json:"plan_name,omitempty"`
}
