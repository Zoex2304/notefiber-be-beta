package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateNoteRequest struct {
	Title      string    `json:"title" validate:"required"`
	Content    string    `json:"content"`
	NotebookId uuid.UUID `json:"notebook_id" validate:"required"`
}

type CreateNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

// BreadcrumbItem represents a single notebook in the ancestry path
// Used for deep linking: allows frontend to display breadcrumbs and auto-expand sidebar
type BreadcrumbItem struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ShowNoteResponse struct {
	Id         uuid.UUID        `json:"id"`
	Title      string           `json:"title"`
	Content    string           `json:"content"`
	NotebookId uuid.UUID        `json:"notebook_id"`
	Breadcrumb []BreadcrumbItem `json:"breadcrumb"` // Notebook ancestry path from root to parent
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  *time.Time       `json:"updated_at"`
}

type UpdateNoteRequest struct {
	Id      uuid.UUID
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

type UpdateNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

type MoveNoteRequest struct {
	Id         uuid.UUID
	NotebookId uuid.UUID `json:"notebook_id" validate:"required"`
}

type MoveNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

type SemanticSearchResponse struct {
	Id             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	NotebookId     uuid.UUID  `json:"notebook_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	SearchType     string     `json:"search_type,omitempty"`     // "literal_filter" | "literal" | "semantic"
	RelevanceScore *float64   `json:"relevance_score,omitempty"` // 0.0-1.0, only for semantic search
}
