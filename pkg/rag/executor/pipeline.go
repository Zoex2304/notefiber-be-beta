package executor

import (
	"context"
	"log"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/llm"
	ragcontext "ai-notetaking-be/pkg/rag/context"
	"ai-notetaking-be/pkg/rag/intent"
	"ai-notetaking-be/pkg/rag/response"
	"ai-notetaking-be/pkg/rag/search"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// PipelineExecutor orchestrates the three-phase RAG pipeline
// Phase 1: Intent Resolution → Phase 2: Context Grounding → Phase 3: Generation
type PipelineExecutor struct {
	intentResolver *intent.Resolver
	grounder       *ragcontext.Grounder
	generator      *response.Generator
	sessionRepo    *memory.SessionRepository
	logger         *log.Logger
}

// NewPipelineExecutor creates a new three-phase pipeline executor
func NewPipelineExecutor(
	llmProvider llm.LLMProvider,
	searchOrchestrator *search.Orchestrator,
	sessionRepo *memory.SessionRepository,
	logger *log.Logger,
) *PipelineExecutor {
	return &PipelineExecutor{
		intentResolver: intent.NewResolver(llmProvider, logger),
		grounder:       ragcontext.NewGrounder(searchOrchestrator, nil, llmProvider, logger),
		generator:      response.NewGenerator(llmProvider, logger),
		sessionRepo:    sessionRepo,
		logger:         logger,
	}
}

// ExecutionResult contains the result of pipeline execution
type ExecutionResult struct {
	Reply              string
	Citations          []dto.CitationDTO
	SessionState       string
	Mode               string
	ResolvedReferences []dto.ResolvedReferenceDTO
}

// Execute runs the complete three-phase pipeline
func (p *PipelineExecutor) Execute(
	ctx context.Context,
	userId uuid.UUID,
	sessionId uuid.UUID,
	query string,
	history []llm.Message,
	uow unitofwork.UnitOfWork,
) (*ExecutionResult, error) {

	// Load or create session
	session, found := p.sessionRepo.Get(sessionId.String())
	if !found {
		session = &store.Session{
			ID:     sessionId.String(),
			UserID: userId.String(),
			State:  store.StateBrowsing,
		}
	}

	p.logger.Printf("[PIPELINE] Starting three-phase execution for query: %s", truncate(query, 50))

	// ═══════════════════════════════════════════════════════════════
	// PHASE 1: INTENT RESOLUTION (Pure LLM - No RAG)
	// ═══════════════════════════════════════════════════════════════
	p.logger.Printf("[PHASE 1] Resolving intent...")

	resolvedIntent, err := p.intentResolver.Resolve(ctx, query, history, session)
	if err != nil {
		p.logger.Printf("[ERROR] Intent resolution failed: %v", err)
		return &ExecutionResult{
			Reply: "Sorry, an error occurred while understanding your question.",
		}, nil
	}

	p.logger.Printf("[PHASE 1] Intent: %s (Target: %d, Scope: %s)",
		resolvedIntent.Action, resolvedIntent.Target, resolvedIntent.Scope)

	// ═══════════════════════════════════════════════════════════════
	// PHASE 2: CONTEXT GROUNDING (Selective RAG)
	// ═══════════════════════════════════════════════════════════════
	p.logger.Printf("[PHASE 2] Grounding context...")

	groundingResult, err := p.grounder.Ground(ctx, resolvedIntent, session, uow, userId, history)
	if err != nil {
		p.logger.Printf("[ERROR] Context grounding failed: %v", err)
		return &ExecutionResult{
			Reply: "Sorry, an error occurred while loading the notes.",
		}, nil
	}

	// Save session state
	p.sessionRepo.Save(groundingResult.Session)

	// If not ready to answer (browse mode, etc.), return early
	if !groundingResult.ShouldAnswer {
		citations := p.buildCitations(groundingResult)
		p.logger.Printf("[PHASE 2] Not answering - returning browse message")

		return &ExecutionResult{
			Reply:        groundingResult.BrowseMessage,
			Citations:    citations,
			SessionState: groundingResult.Session.State,
		}, nil
	}

	p.logger.Printf("[PHASE 2] Context grounded: %d notes (Scope: %s)",
		len(groundingResult.Context.Notes), groundingResult.Context.Scope)

	// ═══════════════════════════════════════════════════════════════
	// PHASE 3: GENERATION (Answer from grounded context)
	// ═══════════════════════════════════════════════════════════════
	p.logger.Printf("[PHASE 3] Generating answer from grounded context...")

	answer := p.generator.GenerateFromGroundedContext(
		ctx,
		query,
		groundingResult.Context,
		history,
	)

	// Build citations from grounded context
	citations := p.buildCitations(groundingResult)

	p.logger.Printf("[PHASE 3] Answer generated, %d citations", len(citations))

	return &ExecutionResult{
		Reply:        answer,
		Citations:    citations,
		SessionState: groundingResult.Session.State,
	}, nil
}

func (p *PipelineExecutor) buildCitations(result *ragcontext.GroundingResult) []dto.CitationDTO {
	var citations []dto.CitationDTO

	if result.Context == nil {
		// No grounded context - use candidates for browse mode
		for _, c := range result.Session.Candidates {
			if nid, err := uuid.Parse(c.ID); err == nil {
				citations = append(citations, dto.CitationDTO{
					NoteId: nid,
					Title:  c.Title,
				})
			}
		}
		return citations
	}

	// Use grounded context IDs
	for _, note := range result.Context.Notes {
		if nid, err := uuid.Parse(note.ID); err == nil {
			citations = append(citations, dto.CitationDTO{
				NoteId: nid,
				Title:  note.Title,
			})
		}
	}

	return citations
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
