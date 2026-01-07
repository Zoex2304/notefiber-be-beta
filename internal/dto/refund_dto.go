package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- User-Side Refund Request ---

type UserRefundRequest struct {
	SubscriptionId uuid.UUID `json:"subscription_id" validate:"required"`
	Reason         string    `json:"reason" validate:"required,min=10"`
}

type UserRefundResponse struct {
	RefundId string `json:"refund_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// --- User's Refund List ---

type UserRefundListResponse struct {
	Id             uuid.UUID `json:"id"`
	SubscriptionId uuid.UUID `json:"subscription_id"`
	PlanName       string    `json:"plan_name"`
	Amount         float64   `json:"amount"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// --- Admin-Side Refund Management ---

type AdminRefundListResponse struct {
	Id           uuid.UUID                   `json:"id"`
	User         AdminRefundUserInfo         `json:"user"`
	Subscription AdminRefundSubscriptionInfo `json:"subscription"`
	Amount       float64                     `json:"amount"`
	Reason       string                      `json:"reason"`
	Status       string                      `json:"status"`
	AdminNotes   string                      `json:"admin_notes,omitempty"`
	CreatedAt    time.Time                   `json:"created_at"`
	ProcessedAt  *time.Time                  `json:"processed_at,omitempty"`
}

type AdminRefundUserInfo struct {
	Id       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
}

type AdminRefundSubscriptionInfo struct {
	Id          uuid.UUID `json:"id"`
	PlanName    string    `json:"plan_name"`
	AmountPaid  float64   `json:"amount_paid"`
	PaymentDate time.Time `json:"payment_date"`
}

type AdminApproveRefundRequest struct {
	AdminNotes string `json:"admin_notes,omitempty"`
}

type AdminApproveRefundResponse struct {
	RefundId       string    `json:"refund_id"`
	Status         string    `json:"status"`
	RefundedAmount float64   `json:"refunded_amount"`
	ProcessedAt    time.Time `json:"processed_at"`
}

type AdminRejectRefundRequest struct {
	AdminNotes string `json:"admin_notes,omitempty"`
}

type AdminRejectRefundResponse struct {
	RefundId    string    `json:"refund_id"`
	Status      string    `json:"status"`
	ProcessedAt time.Time `json:"processed_at"`
}
