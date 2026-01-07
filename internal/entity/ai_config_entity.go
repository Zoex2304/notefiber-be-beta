package entity

import (
	"time"

	"github.com/google/uuid"
)

// AiConfiguration stores AI behavior settings (key-value pairs)
type AiConfiguration struct {
	Id          uuid.UUID
	Key         string // e.g., "rag_similarity_threshold", "default_model"
	Value       string // JSON-encoded value
	ValueType   string // "string", "number", "boolean", "json"
	Description string // Human-readable description
	Category    string // "rag", "llm", "bypass", "nuance"
	IsSecret    bool   // If true, value is encrypted
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AiNuance stores reusable prompt templates for behavior modification
type AiNuance struct {
	Id            uuid.UUID
	Key           string  // e.g., "engineering", "creative", "formal"
	Name          string  // Display name
	Description   string  // Admin description
	SystemPrompt  string  // Injected system prompt
	ModelOverride *string // Optional: use different model for this nuance
	IsActive      bool
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Category constants for AiConfiguration
const (
	AiConfigCategoryRAG     = "rag"
	AiConfigCategoryLLM     = "llm"
	AiConfigCategoryBypass  = "bypass"
	AiConfigCategoryNuance  = "nuance"
	AiConfigCategoryGeneral = "general"
)

// ValueType constants for AiConfiguration
const (
	AiConfigValueTypeString  = "string"
	AiConfigValueTypeNumber  = "number"
	AiConfigValueTypeBoolean = "boolean"
	AiConfigValueTypeJSON    = "json"
)

// Default configuration keys
const (
	AiConfigKeyRAGSimilarityThreshold = "rag_similarity_threshold"
	AiConfigKeyRAGMaxResults          = "rag_max_results"
	AiConfigKeyLLMDefaultModel        = "llm_default_model"
	AiConfigKeyLLMTemperature         = "llm_temperature"
	AiConfigKeyBypassEnabled          = "bypass_enabled"
	AiConfigKeyNuanceEnabled          = "nuance_enabled"
)
