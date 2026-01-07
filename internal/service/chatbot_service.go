package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/ai/pipeline"
	"ai-notetaking-be/pkg/ai/router"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/lexical"
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/rag/access"
	"ai-notetaking-be/pkg/rag/executor"
	"ai-notetaking-be/pkg/rag/history"
	"ai-notetaking-be/pkg/rag/message"
	"ai-notetaking-be/pkg/rag/response"
	"ai-notetaking-be/pkg/rag/search"
	"ai-notetaking-be/pkg/rag/session"
	"ai-notetaking-be/pkg/rag/state"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// IChatbotService defines the chatbot service interface
type IChatbotService interface {
	CreateSession(ctx context.Context, userId uuid.UUID) (*dto.CreateSessionResponse, error)
	GetAllSessions(ctx context.Context, userId uuid.UUID) ([]*dto.GetAllSessionsResponse, error)
	GetChatHistory(ctx context.Context, userId uuid.UUID, sessionId uuid.UUID) ([]*dto.GetChatHistoryResponse, error)
	SendChat(ctx context.Context, userId uuid.UUID, request *dto.SendChatRequest) (*dto.SendChatResponse, error)
	DeleteSession(ctx context.Context, userId uuid.UUID, request *dto.DeleteSessionRequest) error
	GetAvailableNuances(ctx context.Context) ([]*dto.AvailableNuanceResponse, error)
}

// chatbotService coordinates domain components
type chatbotService struct {
	uowFactory  unitofwork.RepositoryFactory
	llmProvider llm.LLMProvider
	sessionRepo *memory.SessionRepository
	llmLogger   *log.Logger

	// Domain components
	// Deprecated components will be removed after migration
	searchOrchestrator *search.Orchestrator
	stateManager       *state.Manager
	responseGenerator  *response.Generator
	messageFactory     *message.Factory
	accessVerifier     *access.Verifier
	historyLoader      *history.Loader
	sessionManager     *session.Manager
	pipelineExecutor   *executor.PipelineExecutor
	pipelineRouter     *router.Router             // Routes between RAG and Bypass pipelines
	explicitExecutor   *executor.ExplicitExecutor // For pre-resolved references
	refResolver        *router.ReferenceResolver  // Resolves note references
}

// NewChatbotService creates a new chatbot service with all domain components
func NewChatbotService(
	uowFactory unitofwork.RepositoryFactory,
	embeddingProvider embedding.EmbeddingProvider,
	llmProvider llm.LLMProvider,
	sessionRepo *memory.SessionRepository,
) IChatbotService {

	llmLogger := initLLMLogger()

	searchOrchestrator := search.NewOrchestrator(embeddingProvider, llmLogger)
	pipelineExecutor := executor.NewPipelineExecutor(llmProvider, searchOrchestrator, sessionRepo, llmLogger)

	// Initialize pipeline router (new routing layer)
	ragPipeline := pipeline.NewRAGPipeline(pipelineExecutor)
	bypassPipeline := pipeline.NewBypassPipeline(llmProvider, llmLogger)
	nuanceResolver := router.NewNuanceResolver()
	pipelineRouter := router.NewRouter(ragPipeline, bypassPipeline, nuanceResolver, llmLogger)

	return &chatbotService{
		uowFactory:  uowFactory,
		llmProvider: llmProvider,
		sessionRepo: sessionRepo,
		llmLogger:   llmLogger,

		// Initialize all domain components
		searchOrchestrator: searchOrchestrator,
		stateManager:       state.NewManager(llmLogger),
		responseGenerator:  response.NewGenerator(llmProvider, llmLogger),
		messageFactory:     message.NewFactory(),
		accessVerifier:     access.NewVerifier(),
		historyLoader:      history.NewLoader(uowFactory, sessionRepo),
		sessionManager:     session.NewManager(sessionRepo),
		pipelineExecutor:   pipelineExecutor,
		pipelineRouter:     pipelineRouter,
		explicitExecutor:   executor.NewExplicitExecutor(llmProvider, llmLogger),
		refResolver:        router.NewReferenceResolver(),
	}
}

func initLLMLogger() *log.Logger {
	logPath := filepath.Join(".", "logs", "llm_rag.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
	}
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return log.New(os.Stdout, "[LLM-RAG] ", log.LstdFlags)
	}
	return log.New(file, "", log.LstdFlags)
}

// CreateSession creates a new chat session
func (cs *chatbotService) CreateSession(ctx context.Context, userId uuid.UUID) (*dto.CreateSessionResponse, error) {
	uow := cs.uowFactory.NewUnitOfWork(ctx)
	now := time.Now()

	chatSession := entity.ChatSession{
		Id:        uuid.New(),
		UserId:    userId,
		Title:     "Unnamed session",
		CreatedAt: now,
	}

	chatMessage := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          "Hi, how can I help you ?",
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now,
	}

	chatMessageRawUser := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          constant.ChatMessageRawInitialUserPromptV1,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now,
	}

	chatMessageRawModel := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          constant.ChatMessageRawInitialModelPromptV1,
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now.Add(1 * time.Second),
	}

	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	if err := uow.ChatSessionRepository().Create(ctx, &chatSession); err != nil {
		return nil, err
	}
	if err := uow.ChatMessageRepository().Create(ctx, &chatMessage); err != nil {
		return nil, err
	}
	if err := uow.ChatMessageRawRepository().Create(ctx, &chatMessageRawUser); err != nil {
		return nil, err
	}
	if err := uow.ChatMessageRawRepository().Create(ctx, &chatMessageRawModel); err != nil {
		return nil, err
	}

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	return &dto.CreateSessionResponse{Id: chatSession.Id}, nil
}

// GetAllSessions retrieves all chat sessions
func (cs *chatbotService) GetAllSessions(ctx context.Context, userId uuid.UUID) ([]*dto.GetAllSessionsResponse, error) {
	uow := cs.uowFactory.NewUnitOfWork(ctx)

	chatSessions, err := uow.ChatSessionRepository().FindAll(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.GetAllSessionsResponse, 0, len(chatSessions))
	for _, s := range chatSessions {
		response = append(response, &dto.GetAllSessionsResponse{
			Id:        s.Id,
			Title:     s.Title,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}

	return response, nil
}

// GetChatHistory retrieves chat history for a session
func (cs *chatbotService) GetChatHistory(ctx context.Context, userId uuid.UUID, sessionId uuid.UUID) ([]*dto.GetChatHistoryResponse, error) {
	uow := cs.uowFactory.NewUnitOfWork(ctx)

	sess, err := uow.ChatSessionRepository().FindOne(ctx,
		specification.ByID{ID: sessionId},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, fmt.Errorf("session not found or access denied")
	}

	chatMessages, err := uow.ChatMessageRepository().FindAll(ctx,
		specification.ByChatSessionID{ChatSessionID: sessionId},
		specification.OrderBy{Field: "created_at", Desc: false},
	)
	if err != nil {
		return nil, err
	}

	messageIds := make([]uuid.UUID, len(chatMessages))
	for i, msg := range chatMessages {
		messageIds[i] = msg.Id
	}

	citations, err := uow.ChatMessageRepository().FindCitationsByMessageIds(ctx, messageIds)
	if err != nil {
		return nil, err
	}

	citationsByMsgId := make(map[uuid.UUID][]dto.CitationDTO)
	for _, c := range citations {
		if c.Note != nil {
			citationsByMsgId[c.ChatMessageId] = append(citationsByMsgId[c.ChatMessageId], dto.CitationDTO{
				NoteId: c.NoteId,
				Title:  c.Note.Title,
			})
		}
	}

	// Fetch references for these messages
	rawRefs, err := uow.ChatMessageReferenceRepository().FindAllByMessageIds(ctx, messageIds)
	if err != nil {
		return nil, err
	}
	refsByMsgId := make(map[uuid.UUID][]dto.ResolvedReferenceDTO)
	for _, r := range rawRefs {
		if r.Note.Id != uuid.Nil {
			refsByMsgId[r.ChatMessageId] = append(refsByMsgId[r.ChatMessageId], dto.ResolvedReferenceDTO{
				NoteId:   r.NoteId,
				Title:    r.Note.Title,
				Resolved: true,
			})
		}
	}

	resp := make([]*dto.GetChatHistoryResponse, 0, len(chatMessages))
	for _, msg := range chatMessages {
		resp = append(resp, &dto.GetChatHistoryResponse{
			Id:         msg.Id,
			Role:       msg.Role,
			Chat:       msg.Chat,
			CreatedAt:  msg.CreatedAt,
			Citations:  citationsByMsgId[msg.Id],
			References: refsByMsgId[msg.Id], // Attach references
		})
	}

	return resp, nil
}

// SendChat processes user message and returns AI response
func (cs *chatbotService) SendChat(ctx context.Context, userId uuid.UUID, request *dto.SendChatRequest) (*dto.SendChatResponse, error) {
	uow := cs.uowFactory.NewUnitOfWork(ctx)

	// Verify access using domain component
	if err := cs.accessVerifier.VerifyAccessAndLimits(ctx, uow, userId); err != nil {
		return nil, err
	}

	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	// Verify session using domain component
	chatSession, err := cs.sessionManager.VerifyChatSession(ctx, uow, userId, request.ChatSessionId)
	if err != nil {
		return nil, err
	}

	existingRawChats, err := uow.ChatMessageRawRepository().FindAll(ctx,
		specification.ByChatSessionID{ChatSessionID: request.ChatSessionId},
		specification.OrderBy{Field: "created_at", Desc: false},
	)
	if err != nil {
		return nil, err
	}

	updateSessionTitle := len(existingRawChats) == 2
	now := time.Now()

	// Create and save user message using domain component
	userMessage := cs.messageFactory.CreateUserMessage(request, now)
	if err := cs.messageFactory.SaveUserMessage(ctx, uow, userMessage); err != nil {
		return nil, err
	}

	// Execute RAG flow (3-phase pipeline)
	pipelineResult, err := cs.executePipeline(ctx, uow, userId, request)
	if err != nil {
		return nil, err
	}

	// Create and save model message using domain component
	modelMessage := cs.messageFactory.CreateModelMessage(request.ChatSessionId, pipelineResult.Reply, now)
	if err := cs.messageFactory.SaveModelMessage(ctx, uow, modelMessage, pipelineResult.Citations); err != nil {
		return nil, err
	}

	// Update session title if needed
	if updateSessionTitle {
		if err := cs.sessionManager.UpdateTitle(ctx, uow, chatSession, request.Chat, now); err != nil {
			return nil, err
		}
	}

	// Increment usage using domain component
	if err := cs.accessVerifier.IncrementUserUsage(ctx, uow, userId); err != nil {
		return nil, err
	}

	// Collect only resolved references for the Sent object (to match History behavior)
	var persistedReferences []dto.ResolvedReferenceDTO
	if len(pipelineResult.ResolvedReferences) > 0 {
		var referencesToSave []*entity.ChatMessageReference
		for _, ref := range pipelineResult.ResolvedReferences {
			// Only save resolved references
			if ref.Resolved {
				referencesToSave = append(referencesToSave, &entity.ChatMessageReference{
					Id:            uuid.New(),
					ChatMessageId: userMessage.Id, // Link to USER message
					NoteId:        ref.NoteId,
					CreatedAt:     now,
				})
				persistedReferences = append(persistedReferences, ref)
			}
		}
		if len(referencesToSave) > 0 {
			if err := uow.ChatMessageReferenceRepository().CreateBulk(ctx, referencesToSave); err != nil {
				return nil, err
			}
		}
	}

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	return &dto.SendChatResponse{
		ChatSessionId:      chatSession.Id,
		ChatSessionTitle:   chatSession.Title,
		Mode:               pipelineResult.Mode,
		ResolvedReferences: pipelineResult.ResolvedReferences,
		Sent: &dto.SendChatResponseChat{
			Id:         userMessage.Id,
			Chat:       userMessage.Chat,
			Role:       userMessage.Role,
			CreatedAt:  userMessage.CreatedAt,
			References: persistedReferences,
		},
		Reply: &dto.SendChatResponseChat{
			Id:        modelMessage.Id,
			Chat:      modelMessage.Chat,
			Role:      modelMessage.Role,
			CreatedAt: modelMessage.CreatedAt,
			Citations: pipelineResult.Citations,
		},
	}, nil
}

// DeleteSession removes a chat session
func (cs *chatbotService) DeleteSession(ctx context.Context, userId uuid.UUID, request *dto.DeleteSessionRequest) error {
	uow := cs.uowFactory.NewUnitOfWork(ctx)

	sess, err := uow.ChatSessionRepository().FindOne(ctx,
		specification.ByID{ID: request.ChatSessionId},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return err
	}
	if sess == nil {
		return fmt.Errorf("session not found or access denied")
	}

	if err := uow.Begin(ctx); err != nil {
		return err
	}
	defer uow.Rollback()

	if err := uow.ChatSessionRepository().Delete(ctx, request.ChatSessionId); err != nil {
		return err
	}
	if err := uow.ChatMessageRepository().DeleteByChatSessionId(ctx, request.ChatSessionId); err != nil {
		return err
	}
	if err := uow.ChatMessageRawRepository().DeleteByChatSessionId(ctx, request.ChatSessionId); err != nil {
		return err
	}

	cs.sessionRepo.Delete(request.ChatSessionId.String())

	return uow.Commit()
}

// executePipeline routes to the appropriate pipeline based on prompt prefix
func (cs *chatbotService) executePipeline(
	ctx context.Context,
	uow unitofwork.UnitOfWork,
	userId uuid.UUID,
	request *dto.SendChatRequest,
) (*executor.ExecutionResult, error) {

	sessionIdStr := request.ChatSessionId.String()

	// Get existing session mode (if any)
	var sessionMode string
	if sess, found := cs.sessionRepo.Get(sessionIdStr); found {
		sessionMode = sess.Mode
	}

	// Parse prompt FIRST to detect mode (BYPASS, NUANCE, or RAG)
	parsed := router.Parse(request.Chat)

	// Determine effective mode for history loading
	effectiveMode := string(parsed.Mode)
	if sessionMode == "BYPASS" {
		effectiveMode = "BYPASS"
	} else if sessionMode == "NUANCE" {
		effectiveMode = "NUANCE"
	}

	// Load history with mode-aware filtering
	// BYPASS mode: RAG system prompts are filtered out
	hist, err := cs.historyLoader.LoadConversationHistory(ctx, userId, request.ChatSessionId, effectiveMode)
	if err != nil {
		cs.llmLogger.Printf("[WARN] Failed to load history: %v", err)
		hist = []llm.Message{}
	}

	// === CHECK FOR EXPLICIT REFERENCES ===
	// Priority 1: DTO references (from export flow)
	// Priority 2: Inline references (@notes:, [[]])
	var explicitNotes []executor.ExplicitContext
	var resolvedRefs []router.ResolvedReference

	if len(request.References) > 0 {
		// References from DTO (export from semantic search)
		cs.llmLogger.Printf("[EXPLICIT] Found %d DTO references", len(request.References))
		for _, ref := range request.References {
			note, err := uow.NoteRepository().FindOne(ctx,
				specification.ByID{ID: ref.NoteId},
				specification.UserOwnedBy{UserID: userId},
			)
			if err != nil || note == nil {
				resolvedRefs = append(resolvedRefs, router.ResolvedReference{
					NoteId: ref.NoteId,
					Found:  false,
				})
				continue
			}
			explicitNotes = append(explicitNotes, executor.ExplicitContext{
				ID:      note.Id.String(),
				Title:   note.Title,
				Content: lexical.ParseContent(note.Content),
			})
			resolvedRefs = append(resolvedRefs, router.ResolvedReference{
				NoteId: note.Id,
				Title:  note.Title,
				Found:  true,
			})
		}
	} else {
		// Check for inline references in prompt
		parsedRefs := router.ParseReferences(request.Chat)
		if parsedRefs.HasRefs {
			cs.llmLogger.Printf("[EXPLICIT] Found %d inline references", len(parsedRefs.References))
			resolvedRefs, _ = cs.refResolver.Resolve(ctx, userId, parsedRefs.References, uow)
			for _, r := range resolvedRefs {
				if r.Found {
					explicitNotes = append(explicitNotes, executor.ExplicitContext{
						ID:      r.NoteId.String(),
						Title:   r.Title,
						Content: r.Content,
					})
				}
			}
			// Use clean prompt (references removed) for generation
			request.Chat = parsedRefs.CleanPrompt
		}
	}

	// If we have explicit notes, use explicit RAG pipeline
	if len(explicitNotes) > 0 {
		cs.llmLogger.Printf("[EXPLICIT] Executing explicit RAG with %d notes", len(explicitNotes))
		explicitResult, err := cs.explicitExecutor.ExecuteWithContext(
			ctx, userId, request.ChatSessionId, request.Chat, explicitNotes, hist,
		)
		if err != nil {
			return nil, err
		}

		// Convert resolved refs to DTO
		resolvedDTOs := make([]dto.ResolvedReferenceDTO, len(resolvedRefs))
		for i, r := range resolvedRefs {
			resolvedDTOs[i] = dto.ResolvedReferenceDTO{
				NoteId:   r.NoteId,
				Title:    r.Title,
				Resolved: r.Found,
			}
		}

		return &executor.ExecutionResult{
			Reply:              explicitResult.Reply,
			Citations:          explicitResult.Citations,
			Mode:               "explicit_rag",
			ResolvedReferences: resolvedDTOs,
		}, nil
	}

	// EXECUTE PIPELINE VIA ROUTER
	// Router handles /bypass, /nuance:X, or default RAG mode
	result, err := cs.pipelineRouter.Execute(
		ctx,
		userId,
		request.ChatSessionId,
		request.Chat,
		hist,
		uow,
		sessionMode, // Pass existing session mode
	)
	if err != nil {
		return nil, err
	}

	// PERSIST session mode if a new mode was set
	// This makes bypass/nuance mode "sticky" for the entire session
	if result.Mode == router.ModeBypass || result.Mode == router.ModeBypassNuance || result.Mode == router.ModeRAGNuance {
		newMode := string(result.Mode)
		if sessionMode != newMode {
			cs.llmLogger.Printf("[SERVICE] Session mode changed: %s -> %s", sessionMode, newMode)
			// Get or create session and update mode
			sess, found := cs.sessionRepo.Get(sessionIdStr)
			if !found {
				sess = &store.Session{
					ID:     sessionIdStr,
					UserID: userId.String(),
					State:  store.StateBrowsing,
				}
			}
			sess.Mode = newMode
			cs.sessionRepo.Save(sess)
		}
	}

	return &executor.ExecutionResult{
		Reply:     result.Reply,
		Citations: result.Citations,
		Mode:      string(result.Mode),
	}, nil
}

// GetAvailableNuances returns all active nuances for public consumption
func (cs *chatbotService) GetAvailableNuances(ctx context.Context) ([]*dto.AvailableNuanceResponse, error) {
	uow := cs.uowFactory.NewUnitOfWork(ctx)

	nuances, err := uow.AiConfigRepository().FindAllNuances(ctx)
	if err != nil {
		return nil, err
	}

	var result []*dto.AvailableNuanceResponse
	for _, n := range nuances {
		if n.IsActive {
			result = append(result, &dto.AvailableNuanceResponse{
				Key:         n.Key,
				Name:        n.Name,
				Description: n.Description,
			})
		}
	}

	return result, nil
}
