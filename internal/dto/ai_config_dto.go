package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// AI Configuration DTOs
// ============================================================================

// AiConfigurationResponse represents an AI configuration entry
type AiConfigurationResponse struct {
	Id          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateAiConfigurationRequest for updating a configuration value
type UpdateAiConfigurationRequest struct {
	Value string `json:"value" validate:"required"`
}

// ============================================================================
// AI Nuance DTOs
// ============================================================================

// AiNuanceResponse represents a nuance entry
type AiNuanceResponse struct {
	Id            uuid.UUID `json:"id"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SystemPrompt  string    `json:"system_prompt"`
	ModelOverride *string   `json:"model_override,omitempty"`
	IsActive      bool      `json:"is_active"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateAiNuanceRequest for creating a new nuance
type CreateAiNuanceRequest struct {
	Key           string  `json:"key" validate:"required,max=100"`
	Name          string  `json:"name" validate:"required,max=200"`
	Description   string  `json:"description"`
	SystemPrompt  string  `json:"system_prompt" validate:"required"`
	ModelOverride *string `json:"model_override,omitempty"`
	SortOrder     int     `json:"sort_order"`
}

// UpdateAiNuanceRequest for updating a nuance
type UpdateAiNuanceRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	SystemPrompt  *string `json:"system_prompt,omitempty"`
	ModelOverride *string `json:"model_override,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
	SortOrder     *int    `json:"sort_order,omitempty"`
}

// AiNuanceListResponse for listing nuances (minimal fields)
type AiNuanceListResponse struct {
	Id       uuid.UUID `json:"id"`
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	IsActive bool      `json:"is_active"`
}

// AvailableNuanceResponse for public nuance listing (user-facing)
type AvailableNuanceResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
