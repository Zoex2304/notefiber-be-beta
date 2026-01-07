// FILE: internal/dto/usage_dto.go
// DTOs for usage limits and status checking
package dto

import (
	"time"

	"github.com/google/uuid"
)

// UsageLimit represents a single limit status
type UsageLimit struct {
	Used     int        `json:"used"`
	Limit    int        `json:"limit"` // -1 = unlimited, 0 = disabled
	CanUse   bool       `json:"can_use"`
	ResetsAt *time.Time `json:"resets_at,omitempty"` // For daily limits
}

// StorageLimits for cumulative resources (notebooks, notes)
type StorageLimits struct {
	Notebooks UsageLimit `json:"notebooks"`
	Notes     UsageLimit `json:"notes"` // Per notebook
}

// DailyLimits for usage that resets daily
type DailyLimits struct {
	AiChat         UsageLimit `json:"ai_chat"`
	SemanticSearch UsageLimit `json:"semantic_search"`
}

// UsageStatusResponse is returned by GET /api/user/usage-status
type UsageStatusResponse struct {
	Plan             PlanInfo      `json:"plan"`
	Storage          StorageLimits `json:"storage"`
	Daily            DailyLimits   `json:"daily"`
	UpgradeAvailable bool          `json:"upgrade_available"`
}

type PlanInfo struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// PlanWithFeaturesResponse is returned by GET /api/plans (public)
type PlanWithFeaturesResponse struct {
	Id            uuid.UUID     `json:"id"`
	Name          string        `json:"name"`
	Slug          string        `json:"slug"`
	Tagline       string        `json:"tagline"`
	Price         float64       `json:"price"`
	BillingPeriod string        `json:"billing_period"`
	IsMostPopular bool          `json:"is_most_popular"`
	Limits        PlanLimitsDTO `json:"limits"`
	Features      []FeatureDTO  `json:"features"`
}

type PlanLimitsDTO struct {
	MaxNotebooks        int `json:"max_notebooks"`
	MaxNotesPerNotebook int `json:"max_notes_per_notebook"`
	AiChatDaily         int `json:"ai_chat_daily"`
	SemanticSearchDaily int `json:"semantic_search_daily"`
}

type FeatureDTO struct {
	Key       string `json:"key"`
	Text      string `json:"text"`
	IsEnabled bool   `json:"is_enabled"`
}

// LimitType constants for error handling
const (
	LimitTypeNotebooks      = "notebooks"
	LimitTypeNotes          = "notes"
	LimitTypeAiChat         = "ai_chat"
	LimitTypeSemanticSearch = "semantic_search"
)
