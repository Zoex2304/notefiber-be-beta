package router

import (
	"context"
	"strings"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/lexical"

	"github.com/google/uuid"
)

// ResolvedReference represents a parsed reference that has been resolved to an actual note
type ResolvedReference struct {
	NoteId    uuid.UUID `json:"note_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"` // Plain text content
	Found     bool      `json:"resolved"`
	Error     string    `json:"error,omitempty"` // If resolution failed
	Reference ParsedReference
}

// ReferenceResolver resolves parsed references to note entities
type ReferenceResolver struct{}

// NewReferenceResolver creates a new reference resolver
func NewReferenceResolver() *ReferenceResolver {
	return &ReferenceResolver{}
}

// Resolve takes parsed references and resolves them to note entities.
// It enforces ownership (userId) and deduplicates results.
// Returns resolved references in the same order as input.
func (r *ReferenceResolver) Resolve(
	ctx context.Context,
	userId uuid.UUID,
	refs []ParsedReference,
	uow unitofwork.UnitOfWork,
) ([]ResolvedReference, error) {
	if len(refs) == 0 {
		return []ResolvedReference{}, nil
	}

	resolved := make([]ResolvedReference, 0, len(refs))
	seen := make(map[uuid.UUID]bool) // Deduplicate

	for _, ref := range refs {
		var note *entity.Note
		var err error

		switch ref.Type {
		case ReferenceTypeUUID:
			note, err = r.resolveByUUID(ctx, userId, ref.Value, uow)
		case ReferenceTypeTitle:
			note, err = r.resolveByTitle(ctx, userId, ref.Value, uow)
		case ReferenceTypePartial:
			note, err = r.resolveByPartial(ctx, userId, ref.Value, uow)
		}

		result := ResolvedReference{
			Reference: ref,
		}

		if err != nil {
			result.Error = err.Error()
			result.Found = false
		} else if note == nil {
			result.Error = "note not found"
			result.Found = false
		} else {
			// Skip duplicates
			if seen[note.Id] {
				continue
			}
			seen[note.Id] = true

			result.NoteId = note.Id
			result.Title = note.Title
			result.Content = lexical.ParseContent(note.Content) // Parse lexical JSON to plain text
			result.Found = true
		}

		resolved = append(resolved, result)
	}

	return resolved, nil
}

// resolveByUUID looks up a note by its UUID
func (r *ReferenceResolver) resolveByUUID(
	ctx context.Context,
	userId uuid.UUID,
	uuidStr string,
	uow unitofwork.UnitOfWork,
) (*entity.Note, error) {
	noteId, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, err
	}

	return uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: noteId},
		specification.UserOwnedBy{UserID: userId},
	)
}

// resolveByTitle looks up a note by exact or close title match
func (r *ReferenceResolver) resolveByTitle(
	ctx context.Context,
	userId uuid.UUID,
	title string,
	uow unitofwork.UnitOfWork,
) (*entity.Note, error) {
	// Try exact match first
	notes, err := uow.NoteRepository().FindAll(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.ByNoteTitle{Title: title},
	)
	if err != nil {
		return nil, err
	}

	if len(notes) > 0 {
		return notes[0], nil // Return first match
	}

	// Fall back to partial/ILIKE match
	return r.resolveByPartial(ctx, userId, title, uow)
}

// resolveByPartial looks up notes using partial text match
func (r *ReferenceResolver) resolveByPartial(
	ctx context.Context,
	userId uuid.UUID,
	query string,
	uow unitofwork.UnitOfWork,
) (*entity.Note, error) {
	notes, err := uow.NoteRepository().FindAll(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.NoteSearchQuery{Query: query},
	)
	if err != nil {
		return nil, err
	}

	if len(notes) == 0 {
		return nil, nil
	}

	// Return best match (first result, assuming title matches are prioritized)
	return notes[0], nil
}

// FilterResolved returns only successfully resolved references
func FilterResolved(refs []ResolvedReference) []ResolvedReference {
	result := make([]ResolvedReference, 0)
	for _, ref := range refs {
		if ref.Found {
			result = append(result, ref)
		}
	}
	return result
}

// ToNoteContext converts resolved references to a format suitable for RAG context
func ToNoteContext(refs []ResolvedReference) []NoteContext {
	result := make([]NoteContext, 0, len(refs))
	for _, ref := range refs {
		if ref.Found {
			result = append(result, NoteContext{
				ID:      ref.NoteId.String(),
				Title:   ref.Title,
				Content: ref.Content,
			})
		}
	}
	return result
}

// NoteContext represents a note ready for RAG context injection
type NoteContext struct {
	ID      string
	Title   string
	Content string
}

// HasAnyResolved checks if at least one reference was resolved
func HasAnyResolved(refs []ResolvedReference) bool {
	for _, ref := range refs {
		if ref.Found {
			return true
		}
	}
	return false
}

// GetResolvedIDs extracts note IDs from resolved references
func GetResolvedIDs(refs []ResolvedReference) []uuid.UUID {
	ids := make([]uuid.UUID, 0)
	for _, ref := range refs {
		if ref.Found {
			ids = append(ids, ref.NoteId)
		}
	}
	return ids
}

// SummarizeUnresolved returns a human-readable summary of unresolved refs
func SummarizeUnresolved(refs []ResolvedReference) string {
	unresolved := make([]string, 0)
	for _, ref := range refs {
		if !ref.Found {
			unresolved = append(unresolved, ref.Reference.Value)
		}
	}
	if len(unresolved) == 0 {
		return ""
	}
	return "Could not find: " + strings.Join(unresolved, ", ")
}
