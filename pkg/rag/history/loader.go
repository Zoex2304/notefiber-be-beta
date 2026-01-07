package history

import (
	"context"
	"strings"

	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/llm"

	"github.com/google/uuid"
)

// Loader handles conversation history and citations
type Loader struct {
	uowFactory  unitofwork.RepositoryFactory
	sessionRepo *memory.SessionRepository
}

// NewLoader creates a new history loader
func NewLoader(uowFactory unitofwork.RepositoryFactory, sessionRepo *memory.SessionRepository) *Loader {
	return &Loader{
		uowFactory:  uowFactory,
		sessionRepo: sessionRepo,
	}
}

// LoadConversationHistory loads recent chat history for LLM context.
// When mode is "BYPASS", RAG system prompts are filtered out to prevent contamination.
func (l *Loader) LoadConversationHistory(ctx context.Context, userId uuid.UUID, sessionId uuid.UUID, mode string) ([]llm.Message, error) {
	uow := l.uowFactory.NewUnitOfWork(ctx)

	rawChats, err := uow.ChatMessageRawRepository().FindAll(ctx,
		specification.ByChatSessionID{ChatSessionID: sessionId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	limit := 10
	if len(rawChats) > limit {
		rawChats = rawChats[:limit]
	}

	messages := make([]llm.Message, 0, len(rawChats))
	for i := len(rawChats) - 1; i >= 0; i-- {
		chat := rawChats[i]

		// BYPASS MODE or BYPASS_NUANCE MODE: Filter out RAG system prompts to prevent contamination
		if (mode == "BYPASS" || mode == "BYPASS_NUANCE") && isRAGSystemPrompt(chat.Chat) {
			continue
		}

		role := "user"
		if chat.Role == constant.ChatMessageRoleModel {
			role = "assistant"
		}
		messages = append(messages, llm.Message{
			Role:    role,
			Content: chat.Chat,
		})
	}

	return messages, nil
}

// isRAGSystemPrompt detects if a message is a RAG system prompt.
// These are the priming prompts seeded at session creation.
func isRAGSystemPrompt(content string) bool {
	// Check against known RAG prompt constants
	if content == constant.ChatMessageRawInitialUserPromptV1 ||
		content == constant.ChatMessageRawInitialModelPromptV1 {
		return true
	}

	// Fallback: detect by content fingerprints
	ragFingerprints := []string{
		"pattern-based logic",
		"QUERY PATTERN MATCHING",
		"According to [note_title]",
		"CITATION FORMAT",
		"NOTES DATABASE",
		"pattern matching internally",
		"cite sources naturally",
	}

	lowerContent := strings.ToLower(content)
	for _, fp := range ragFingerprints {
		if strings.Contains(lowerContent, strings.ToLower(fp)) {
			return true
		}
	}

	return false
}

// PrepareCitations builds CONTEXTUAL citation list based on session state
// Citations must reflect what was actually used to generate the answer
func (l *Loader) PrepareCitations(userId uuid.UUID, sessionId uuid.UUID) []dto.CitationDTO {
	session, found := l.sessionRepo.Get(sessionId.String())
	if !found {
		return []dto.CitationDTO{}
	}

	var citations []dto.CitationDTO

	// Priority 1: If a SPECIFIC note is focused (not aggregated), cite only that note
	if session.FocusedNote != nil && session.FocusedNote.ID != "aggregated" {
		if nid, err := uuid.Parse(session.FocusedNote.ID); err == nil {
			citations = append(citations, dto.CitationDTO{
				NoteId: nid,
				Title:  session.FocusedNote.Title,
			})
		}
		return citations // Return only the focused note
	}

	// Priority 2: If in AGGREGATED mode (all notes combined), cite all candidates
	if session.FocusedNote != nil && session.FocusedNote.ID == "aggregated" {
		for _, c := range session.Candidates {
			if nid, err := uuid.Parse(c.ID); err == nil {
				citations = append(citations, dto.CitationDTO{
					NoteId: nid,
					Title:  c.Title,
				})
			}
		}
		return citations
	}

	// Priority 3: BROWSING mode (no focus) - cite all candidates as potential sources
	if len(session.Candidates) > 0 {
		for _, c := range session.Candidates {
			if nid, err := uuid.Parse(c.ID); err == nil {
				citations = append(citations, dto.CitationDTO{
					NoteId: nid,
					Title:  c.Title,
				})
			}
		}
	}

	return citations
}
