package executor

import (
	"context"
	"log"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/pkg/llm"
	ragcontext "ai-notetaking-be/pkg/rag/context"
	"ai-notetaking-be/pkg/rag/intent"
	"ai-notetaking-be/pkg/rag/response"

	"github.com/google/uuid"
)

// ExplicitContext represents pre-resolved note context for explicit RAG
type ExplicitContext struct {
	ID      string
	Title   string
	Content string
}

// ExplicitExecutor handles RAG execution with pre-resolved context.
// This bypasses the Intent Resolution and Context Grounding phases,
// going directly to Generation with user-specified notes.
type ExplicitExecutor struct {
	generator *response.Generator
	logger    *log.Logger
}

// NewExplicitExecutor creates a new explicit RAG executor
func NewExplicitExecutor(llmProvider llm.LLMProvider, logger *log.Logger) *ExplicitExecutor {
	return &ExplicitExecutor{
		generator: response.NewGenerator(llmProvider, logger),
		logger:    logger,
	}
}

// ExecuteWithContext runs RAG with pre-resolved notes.
// This is the "Explicit RAG" mode - user explicitly referenced notes,
// so we skip search and go straight to generation.
func (e *ExplicitExecutor) ExecuteWithContext(
	ctx context.Context,
	userId uuid.UUID,
	sessionId uuid.UUID,
	query string,
	notes []ExplicitContext,
	history []llm.Message,
) (*ExecutionResult, error) {

	e.logger.Printf("[EXPLICIT] Executing with %d pre-resolved notes", len(notes))

	if len(notes) == 0 {
		e.logger.Printf("[EXPLICIT] No notes provided, returning error message")
		return &ExecutionResult{
			Reply:     "No valid notes were found for the references you provided.",
			Citations: []dto.CitationDTO{},
		}, nil
	}

	// Convert ExplicitContext to GroundedContext format
	groundedNotes := make([]ragcontext.NoteContent, len(notes))
	noteIDs := make([]string, len(notes))
	for i, note := range notes {
		groundedNotes[i] = ragcontext.NoteContent{
			ID:      note.ID,
			Title:   note.Title,
			Content: note.Content,
		}
		noteIDs[i] = note.ID
		e.logger.Printf("[EXPLICIT] Note %d: '%s' (%d chars)", i+1, note.Title, len(note.Content))
	}

	// Determine scope based on note count
	scope := intent.ScopeSingle
	if len(notes) > 1 {
		scope = intent.ScopeAll
	}

	// Build grounded context
	groundedContext := &ragcontext.GroundedContext{
		Notes: groundedNotes,
		Scope: scope,
		IDs:   noteIDs,
	}

	// If single note, set focus
	if len(notes) == 1 {
		groundedContext.FocusedID = notes[0].ID
	}

	// PHASE 3: GENERATION (Skip Phase 1 Intent + Phase 2 Search)
	e.logger.Printf("[EXPLICIT] Generating response from explicit context (Scope: %s)", scope)

	answer := e.generator.GenerateFromGroundedContext(ctx, query, groundedContext, history)

	// Build citations
	citations := make([]dto.CitationDTO, len(notes))
	for i, note := range notes {
		noteId, _ := uuid.Parse(note.ID)
		citations[i] = dto.CitationDTO{
			NoteId: noteId,
			Title:  note.Title,
		}
	}

	e.logger.Printf("[EXPLICIT] Response generated with %d citations", len(citations))

	return &ExecutionResult{
		Reply:     answer,
		Citations: citations,
	}, nil
}
