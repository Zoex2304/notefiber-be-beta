package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateSessionResponse struct {
	Id uuid.UUID `json:"id"`
}

type GetAllSessionsResponse struct {
	Id        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type GetChatHistoryResponse struct {
	Id         uuid.UUID              `json:"id"`
	Role       string                 `json:"role"`
	Chat       string                 `json:"chat"`
	CreatedAt  time.Time              `json:"created_at"`
	Citations  []CitationDTO          `json:"citations,omitempty"`
	References []ResolvedReferenceDTO `json:"references,omitempty"`
}

type CitationDTO struct {
	NoteId uuid.UUID `json:"note_id"`
	Title  string    `json:"title"`
}

type SendChatRequest struct {
	ChatSessionId uuid.UUID          `json:"chat_session_id" validate:"required"`
	Chat          string             `json:"chat" validate:"required"`
	References    []NoteReferenceDTO `json:"references,omitempty" validate:"max=5"` // Pre-resolved note references
}

// NoteReferenceDTO represents a note reference in the chat request
type NoteReferenceDTO struct {
	NoteId     uuid.UUID `json:"note_id" validate:"required"`
	SourceType string    `json:"source_type,omitempty"` // "export" | "inline" | "autocomplete"
}

type SendChatResponseChat struct {
	Id         uuid.UUID              `json:"id"`
	Chat       string                 `json:"chat"`
	Role       string                 `json:"role"`
	CreatedAt  time.Time              `json:"created_at"`
	Citations  []CitationDTO          `json:"citations,omitempty"`
	References []ResolvedReferenceDTO `json:"references,omitempty"`
}

type SendChatResponse struct {
	ChatSessionId      uuid.UUID              `json:"chat_session_id"`
	ChatSessionTitle   string                 `json:"title"`
	Sent               *SendChatResponseChat  `json:"sent"`
	Reply              *SendChatResponseChat  `json:"reply"`
	Mode               string                 `json:"mode,omitempty"` // "rag" | "explicit_rag" | "bypass"
	ResolvedReferences []ResolvedReferenceDTO `json:"resolved_references,omitempty"`
}

// ResolvedReferenceDTO shows which references were resolved
type ResolvedReferenceDTO struct {
	NoteId   uuid.UUID `json:"note_id"`
	Title    string    `json:"title"`
	Resolved bool      `json:"resolved"`
}

type DeleteSessionRequest struct {
	ChatSessionId uuid.UUID `json:"chat_session_id"`
}

// --- Limit Exceeded Error Types ---

// LimitExceededError is a custom error that carries usage details
type LimitExceededError struct {
	Limit      int       `json:"limit"`
	Used       int       `json:"used"`
	ResetAfter time.Time `json:"reset_after"`
}

func (e *LimitExceededError) Error() string {
	return "daily AI usage limit exceeded"
}

// LimitExceededData is the data payload for 429 responses
type LimitExceededData struct {
	Limit            int       `json:"limit"`
	Used             int       `json:"used"`
	ResetAfter       time.Time `json:"reset_after"`
	ShowModalPricing bool      `json:"show_modal_pricing"`
}

// LimitExceededResponse is the full 429 response structure
type LimitExceededResponse struct {
	Success   bool              `json:"success"`
	Code      int               `json:"code"`
	Message   string            `json:"message"`
	ErrorType string            `json:"error_type"`
	Data      LimitExceededData `json:"data"`
}
