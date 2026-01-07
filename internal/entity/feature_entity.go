// FILE: internal/entity/feature_entity.go
// Domain entity for features
package entity

import (
	"time"

	"github.com/google/uuid"
)

// Feature represents a feature in the master catalog
type Feature struct {
	Id          uuid.UUID
	Key         string // Unique key: ai_chat, semantic_search, etc.
	Name        string // Display name: "AI Chat Assistant"
	Description string // Full description
	Category    string // Category: ai, storage, support, export
	IsActive    bool   // Global enable/disable
	SortOrder   int    // Display order
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
