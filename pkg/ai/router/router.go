package router

import (
	"context"
	"log"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/ai/pipeline"
	"ai-notetaking-be/pkg/llm"

	"github.com/google/uuid"
)

// ExecuteResult is the unified result from any pipeline execution
type ExecuteResult struct {
	Reply     string
	Citations []dto.CitationDTO
	Mode      Mode
	NuanceKey string // If nuance was applied, this is the key
}

// NuanceResolver resolves nuance configurations from the database
type NuanceResolver interface {
	GetNuanceByKey(ctx context.Context, uow unitofwork.UnitOfWork, key string) (*entity.AiNuance, error)
}

// Router handles pipeline selection based on prompt analysis
type Router struct {
	ragPipeline    *pipeline.RAGPipeline
	bypassPipeline *pipeline.BypassPipeline
	nuanceResolver NuanceResolver
	logger         *log.Logger
}

// NewRouter creates a new pipeline router
func NewRouter(
	ragPipeline *pipeline.RAGPipeline,
	bypassPipeline *pipeline.BypassPipeline,
	nuanceResolver NuanceResolver,
	logger *log.Logger,
) *Router {
	return &Router{
		ragPipeline:    ragPipeline,
		bypassPipeline: bypassPipeline,
		nuanceResolver: nuanceResolver,
		logger:         logger,
	}
}

// Execute routes and executes the appropriate pipeline based on prompt prefix
// sessionMode: The current mode stored in the session ("" if not set, or "BYPASS"/"NUANCE"/"RAG")
// Returns: ExecuteResult with the resolved Mode that should be persisted to session
func (r *Router) Execute(
	ctx context.Context,
	userId uuid.UUID,
	sessionId uuid.UUID,
	prompt string,
	history []llm.Message,
	uow unitofwork.UnitOfWork,
	sessionMode string, // Existing session mode (empty string if not set)
) (*ExecuteResult, error) {

	// 1. Parse prompt for routing directives
	parsed := Parse(prompt)

	// 2. Determine effective mode (session mode takes precedence if already set)
	effectiveMode := parsed.Mode
	if sessionMode == string(ModeBypass) {
		// Session is locked to bypass mode - override parsed mode
		effectiveMode = ModeBypass
		r.logger.Printf("[ROUTER] Session locked to BYPASS mode, using clean prompt")
		// Use un-prefixed version if user included prefix again
		if parsed.Mode == ModeBypass || parsed.Mode == ModeBypassNuance {
			// Already parsed, use clean prompt
		} else {
			// User didn't include prefix, treat entire prompt as query
			parsed.CleanPrompt = prompt
		}
	} else if sessionMode == string(ModeBypassNuance) {
		effectiveMode = ModeBypassNuance
		r.logger.Printf("[ROUTER] Session locked to BYPASS_NUANCE mode")
		if parsed.Mode != ModeBypassNuance {
			parsed.CleanPrompt = prompt
		}
	} else if sessionMode == string(ModeRAGNuance) {
		effectiveMode = ModeRAGNuance
		r.logger.Printf("[ROUTER] Session locked to RAG_NUANCE mode")
		if parsed.Mode != ModeRAGNuance {
			parsed.CleanPrompt = prompt
		}
	}

	r.logger.Printf("[ROUTER] Mode: %s (session: %s), CleanPrompt: %s",
		effectiveMode, sessionMode, truncateLog(parsed.CleanPrompt, 50))

	// Handle empty prompt after prefix extraction
	if parsed.IsEmpty() && effectiveMode != ModeRAG {
		r.logger.Printf("[ROUTER] Empty prompt after prefix extraction, treating as help request")
		helpMsg := r.getHelpMessage(effectiveMode, parsed.NuanceKey)
		return &ExecuteResult{
			Reply:     helpMsg,
			Citations: nil,
			Mode:      effectiveMode,
		}, nil
	}

	// 3. Resolve nuance if requested
	var nuanceConfig *pipeline.NuanceConfig
	if (effectiveMode == ModeBypassNuance || effectiveMode == ModeRAGNuance) && parsed.NuanceKey != "" {
		nuance, err := r.resolveNuance(ctx, uow, parsed.NuanceKey)
		if err != nil {
			r.logger.Printf("[ROUTER] Failed to resolve nuance '%s': %v", parsed.NuanceKey, err)
			// Fall back to mode without nuance
		} else if nuance != nil {
			nuanceConfig = nuance
			r.logger.Printf("[ROUTER] Nuance '%s' resolved: %s", parsed.NuanceKey, nuance.Name)
		} else {
			r.logger.Printf("[ROUTER] Nuance '%s' not found, proceeding without nuance", parsed.NuanceKey)
		}
	}

	// 4. Route based on effective mode
	switch effectiveMode {
	case ModeBypass:
		return r.executeBypass(ctx, parsed.CleanPrompt, history, nil)

	case ModeBypassNuance:
		// Bypass + Nuance: Use bypass pipeline with nuance injection
		result, err := r.executeBypass(ctx, parsed.CleanPrompt, history, nuanceConfig)
		if err != nil {
			return nil, err
		}
		result.Mode = ModeBypassNuance
		result.NuanceKey = parsed.NuanceKey
		return result, nil

	case ModeRAGNuance:
		// RAG + Nuance: Use RAG pipeline with nuance context
		// TODO: Inject nuance into RAG response generation
		result, err := r.executeRAG(ctx, userId, sessionId, parsed.CleanPrompt, history, uow, ModeRAGNuance)
		if err != nil {
			return nil, err
		}
		result.NuanceKey = parsed.NuanceKey
		return result, nil

	default: // ModeRAG
		return r.executeRAG(ctx, userId, sessionId, parsed.CleanPrompt, history, uow, ModeRAG)
	}
}

// resolveNuance loads nuance configuration from the database
func (r *Router) resolveNuance(ctx context.Context, uow unitofwork.UnitOfWork, key string) (*pipeline.NuanceConfig, error) {
	if r.nuanceResolver == nil {
		return nil, nil
	}

	nuance, err := r.nuanceResolver.GetNuanceByKey(ctx, uow, key)
	if err != nil {
		return nil, err
	}
	if nuance == nil {
		return nil, nil
	}

	return &pipeline.NuanceConfig{
		Key:           nuance.Key,
		Name:          nuance.Name,
		SystemPrompt:  nuance.SystemPrompt,
		ModelOverride: nuance.ModelOverride,
	}, nil
}

// executeBypass runs the bypass pipeline (pure LLM)
func (r *Router) executeBypass(
	ctx context.Context,
	query string,
	history []llm.Message,
	nuance *pipeline.NuanceConfig,
) (*ExecuteResult, error) {
	if nuance != nil {
		r.logger.Printf("[ROUTER] Executing BYPASS pipeline with nuance: %s", nuance.Key)
	} else {
		r.logger.Printf("[ROUTER] Executing BYPASS pipeline")
	}

	result, err := r.bypassPipeline.Execute(ctx, query, history, nuance)
	if err != nil {
		r.logger.Printf("[ROUTER] Bypass pipeline error: %v", err)
		return nil, err
	}

	mode := ModeBypass
	nuanceKey := ""
	if nuance != nil {
		mode = ModeBypassNuance
		nuanceKey = nuance.Key
	}

	return &ExecuteResult{
		Reply:     result.Reply,
		Citations: nil, // No citations in bypass mode
		Mode:      mode,
		NuanceKey: nuanceKey,
	}, nil
}

// executeRAG runs the RAG pipeline
func (r *Router) executeRAG(
	ctx context.Context,
	userId uuid.UUID,
	sessionId uuid.UUID,
	query string,
	history []llm.Message,
	uow unitofwork.UnitOfWork,
	mode Mode,
) (*ExecuteResult, error) {
	r.logger.Printf("[ROUTER] Executing RAG pipeline")

	result, err := r.ragPipeline.Execute(ctx, userId, sessionId, query, history, uow)
	if err != nil {
		r.logger.Printf("[ROUTER] RAG pipeline error: %v", err)
		return nil, err
	}

	return &ExecuteResult{
		Reply:     result.Reply,
		Citations: result.Citations,
		Mode:      mode,
	}, nil
}

// getHelpMessage returns a helpful message when user sends empty prompt with prefix
func (r *Router) getHelpMessage(mode Mode, nuanceKey string) string {
	switch mode {
	case ModeBypass:
		return "Mode bypass diaktifkan. Silakan ketik pertanyaan Anda setelah /bypass untuk mengobrol langsung dengan AI tanpa mengakses catatan Anda.\n\nContoh: /bypass Apa itu machine learning?"
	case ModeBypassNuance:
		return "Mode bypass+nuance '" + nuanceKey + "' diaktifkan. AI akan merespons dengan gaya khusus tanpa mengakses catatan.\n\nContoh: /bypass/nuance:engineering Jelaskan konsep ini."
	case ModeRAGNuance:
		return "Mode RAG+nuance '" + nuanceKey + "' diaktifkan. AI akan mencari catatan dan merespons dengan gaya khusus.\n\nContoh: /nuance:teacher Jelaskan materi ini."
	default:
		return "Silakan ketik pertanyaan Anda."
	}
}

// truncateLog truncates string for logging
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
