// FILE: internal/dto/billing_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- Admin Billing Management ---

// AdminBillingListResponse represents a billing address for admin view
type AdminBillingListResponse struct {
	Id           uuid.UUID `json:"id"`
	UserId       uuid.UUID `json:"user_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	AddressLine1 string    `json:"address_line1"`
	AddressLine2 string    `json:"address_line2,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AdminBillingCreateRequest for admin creating billing address for a user
type AdminBillingCreateRequest struct {
	FirstName    string `json:"first_name" validate:"required"`
	LastName     string `json:"last_name" validate:"required"`
	Email        string `json:"email" validate:"required,email"`
	Phone        string `json:"phone,omitempty"`
	AddressLine1 string `json:"address_line1" validate:"required"`
	AddressLine2 string `json:"address_line2,omitempty"`
	City         string `json:"city" validate:"required"`
	State        string `json:"state" validate:"required"`
	PostalCode   string `json:"postal_code" validate:"required,max=10"`
	Country      string `json:"country" validate:"required"`
	IsDefault    bool   `json:"is_default"`
}

// AdminBillingUpdateRequest for admin updating a billing address
type AdminBillingUpdateRequest struct {
	FirstName    *string `json:"first_name,omitempty"`
	LastName     *string `json:"last_name,omitempty"`
	Email        *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone        *string `json:"phone,omitempty"`
	AddressLine1 *string `json:"address_line1,omitempty"`
	AddressLine2 *string `json:"address_line2,omitempty"`
	City         *string `json:"city,omitempty"`
	State        *string `json:"state,omitempty"`
	PostalCode   *string `json:"postal_code,omitempty" validate:"omitempty,max=10"`
	Country      *string `json:"country,omitempty"`
	IsDefault    *bool   `json:"is_default,omitempty"`
}

// --- User Billing (Settings Page) ---

// UserBillingResponse for user's billing info on Settings page
type UserBillingResponse struct {
	Id           uuid.UUID `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	AddressLine1 string    `json:"address_line1"`
	AddressLine2 string    `json:"address_line2,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"`
}

// UserBillingUpdateRequest for user updating their billing info
type UserBillingUpdateRequest struct {
	FirstName    string `json:"first_name" validate:"required"`
	LastName     string `json:"last_name" validate:"required"`
	Email        string `json:"email" validate:"required,email"`
	Phone        string `json:"phone,omitempty"`
	AddressLine1 string `json:"address_line1" validate:"required"`
	AddressLine2 string `json:"address_line2,omitempty"`
	City         string `json:"city" validate:"required"`
	State        string `json:"state" validate:"required"`
	PostalCode   string `json:"postal_code" validate:"required,max=10"`
	Country      string `json:"country" validate:"required"`
}
