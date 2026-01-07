package executor

import (
	"context"
	"fmt"
	"strings"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/rag/response"
	"ai-notetaking-be/pkg/rag/search"
	"ai-notetaking-be/pkg/rag/state"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// ActionExecutor executes actions determined by the planner
type ActionExecutor struct {
	searchOrchestrator *search.Orchestrator
	stateManager       *state.Manager
	responseGenerator  *response.Generator
	uow                unitofwork.UnitOfWork
	session            *store.Session
	history            []llm.Message
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(
	searchOrchestrator *search.Orchestrator,
	stateManager *state.Manager,
	responseGenerator *response.Generator,
	uow unitofwork.UnitOfWork,
	session *store.Session,
	history []llm.Message,
) *ActionExecutor {
	return &ActionExecutor{
		searchOrchestrator: searchOrchestrator,
		stateManager:       stateManager,
		responseGenerator:  responseGenerator,
		uow:                uow,
		session:            session,
		history:            history,
	}
}

// Execute performs the action determined by semantic intent analysis
func (e *ActionExecutor) Execute(
	ctx context.Context,
	userId uuid.UUID,
	plan *dto.AIActionPlan,
	query string,
) (string, error) {

	switch plan.Action {
	case dto.PlannerActionSearch:
		return e.executeSearch(ctx, userId, plan.SearchQuery, query)

	case dto.PlannerActionSelect:
		return e.executeSelect(ctx, plan.TargetIndex, query)

	case dto.PlannerActionSwitch:
		return e.executeSwitch(ctx, plan.TargetIndex, query)

	case dto.PlannerActionAnswerCurrent:
		return e.executeAnswerCurrent(ctx, query)

	case dto.PlannerActionAnswerAll:
		return e.executeAnswerAll(ctx, query)

	case dto.PlannerActionClarify:
		return e.executeClarify()

	default:
		return e.executeAnswerCurrent(ctx, query)
	}
}

func (e *ActionExecutor) executeSearch(ctx context.Context, userId uuid.UUID, searchQuery, originalQuery string) (string, error) {
	query := searchQuery
	if query == "" {
		query = originalQuery
	}

	config := search.DefaultConfig()
	candidates, err := e.searchOrchestrator.Execute(ctx, e.uow, userId, query, config)
	if err != nil {
		return "", err
	}

	// No results
	if len(candidates) == 0 {
		return e.responseGenerator.GenerateNotFoundMessage(), nil
	}

	// Single result: auto-focus
	if len(candidates) == 1 {
		e.stateManager.TransitionToFocused(e.session, candidates[0])
		return e.responseGenerator.GenerateAnswer(ctx, e.session, query, e.history), nil
	}

	// Multiple results: enter browsing mode
	e.stateManager.TransitionToBrowsing(e.session, candidates)
	return e.responseGenerator.GenerateBrowsingMessage(candidates), nil
}

func (e *ActionExecutor) executeSelect(ctx context.Context, targetIndex int, query string) (string, error) {
	if targetIndex < 0 || targetIndex >= len(e.session.Candidates) {
		return e.responseGenerator.GenerateInvalidSelectionMessage(), nil
	}

	selected := e.session.Candidates[targetIndex]

	// Hydrate with full content
	if nid, err := uuid.Parse(selected.ID); err == nil {
		if note, err := e.uow.NoteRepository().FindOne(ctx, specification.ByID{ID: nid}); err == nil && note != nil {
			selected.Content = note.Content
		}
	}

	e.stateManager.TransitionToFocused(e.session, selected)
	return e.responseGenerator.GenerateAnswer(ctx, e.session, query, e.history), nil
}

func (e *ActionExecutor) executeSwitch(ctx context.Context, targetIndex int, query string) (string, error) {
	return e.executeSelect(ctx, targetIndex, query)
}

func (e *ActionExecutor) executeAnswerCurrent(ctx context.Context, query string) (string, error) {
	if e.session.FocusedNote == nil {
		return "Maaf, saya kehilangan konteks catatan yang sedang dibahas. Bisa tolong cari ulang?", nil
	}

	return e.responseGenerator.GenerateAnswer(ctx, e.session, query, e.history), nil
}

func (e *ActionExecutor) executeAnswerAll(ctx context.Context, query string) (string, error) {
	if len(e.session.Candidates) == 0 {
		return "Tidak ada catatan yang tersedia untuk dijawab.", nil
	}

	// Aggregate all candidates
	var aggregatedContent strings.Builder
	for i, cand := range e.session.Candidates {
		if nid, err := uuid.Parse(cand.ID); err == nil {
			if note, err := e.uow.NoteRepository().FindOne(ctx, specification.ByID{ID: nid}); err == nil && note != nil {
				aggregatedContent.WriteString(fmt.Sprintf("--- Note %d: %s ---\n", i+1, cand.Title))
				aggregatedContent.WriteString(note.Content)
				aggregatedContent.WriteString("\n\n")
			}
		}
	}

	e.stateManager.TransitionToAggregated(e.session, e.session.Candidates, aggregatedContent.String())
	return e.responseGenerator.GenerateAnswer(ctx, e.session, query, e.history), nil
}

func (e *ActionExecutor) executeClarify() (string, error) {
	if len(e.session.Candidates) > 0 {
		return e.responseGenerator.GenerateBrowsingMessage(e.session.Candidates), nil
	}
	return "Bisa jelaskan lebih lanjut apa yang ingin Anda cari?", nil
}
