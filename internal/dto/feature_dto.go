// FILE: internal/dto/feature_dto.go
// DTOs for Feature catalog CRUD
package dto

import "github.com/google/uuid"

// --- Feature Catalog DTOs ---

// CreateFeatureRequest is used to add a new feature to the catalog
type CreateFeatureRequest struct {
	Key         string `json:"key" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"` // ai, storage, support, export
	IsActive    bool   `json:"is_active"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateFeatureRequest is used to update a feature in the catalog
type UpdateFeatureRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// FeatureResponse is returned when getting feature(s) from the catalog
type FeatureResponse struct {
	Id          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	IsActive    bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
}

// --- Updated Plan Feature DTOs (using FeatureId) ---

// AssignFeatureRequest is used to assign a feature from the catalog to a plan
type AssignFeatureRequest struct {
	FeatureId uuid.UUID `json:"feature_id" validate:"required"`
	IsEnabled bool      `json:"is_enabled"`
	SortOrder int       `json:"sort_order"`
}

// PlanFeatureDetailResponse includes the full feature details from the catalog
type PlanFeatureDetailResponse struct {
	Id        uuid.UUID        `json:"id"`
	PlanId    uuid.UUID        `json:"plan_id"`
	FeatureId *uuid.UUID       `json:"feature_id,omitempty"`
	Feature   *FeatureResponse `json:"feature,omitempty"` // Expanded from catalog
	IsEnabled bool             `json:"is_enabled"`
	SortOrder int              `json:"sort_order"`
	// Legacy fields (for backward compatibility during migration)
	FeatureKey  *string `json:"feature_key,omitempty"`
	DisplayText *string `json:"display_text,omitempty"`
}
