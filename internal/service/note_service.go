// FILE: internal/service/note_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/events"
	"ai-notetaking-be/pkg/lexical"
	pktNats "ai-notetaking-be/pkg/nats"
	"ai-notetaking-be/pkg/rag/access"
	pkgSearch "ai-notetaking-be/pkg/search" // Fixed import

	"github.com/google/uuid"
)

type INoteService interface {
	Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error)
	Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNoteResponse, error)
	Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error)
	Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error
	MoveNote(ctx context.Context, userId uuid.UUID, req *dto.MoveNoteRequest) (*dto.MoveNoteResponse, error)
	SemanticSearch(ctx context.Context, userId uuid.UUID, search string) ([]*dto.SemanticSearchResponse, error)
}

type noteService struct {
	uowFactory        unitofwork.RepositoryFactory
	publisherService  IPublisherService
	embeddingProvider embedding.EmbeddingProvider
	eventPublisher    *pktNats.Publisher
	accessVerifier    *access.Verifier
}

func NewNoteService(
	uowFactory unitofwork.RepositoryFactory,
	publisherService IPublisherService,
	embeddingProvider embedding.EmbeddingProvider,
	eventPublisher *pktNats.Publisher,
) INoteService {
	return &noteService{
		uowFactory:        uowFactory,
		publisherService:  publisherService,
		embeddingProvider: embeddingProvider,
		eventPublisher:    eventPublisher,
		accessVerifier:    access.NewVerifier(),
	}
}

func (c *noteService) Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	note := entity.Note{
		Id:         uuid.New(),
		Title:      req.Title,
		Content:    req.Content,
		NotebookId: req.NotebookId,
		UserId:     userId,
		CreatedAt:  time.Now(),
	}

	err := uow.NoteRepository().Create(ctx, &note)
	if err != nil {
		return nil, err
	}

	msgPayload := dto.PublishEmbedNoteMessage{
		NoteId: note.Id,
	}
	msgJson, err := json.Marshal(msgPayload)
	if err != nil {
		return nil, err
	}

	err = c.publisherService.Publish(ctx, msgJson)
	if err != nil {
		return nil, err
	}

	// Publish Event for Notification System
	if c.eventPublisher != nil {
		evt := events.BaseEvent{
			Type: "NOTE_CREATED",
			Data: map[string]interface{}{
				"title":   note.Title, // Template uses {title}
				"note_id": note.Id,
				"user_id": userId,
			},
			OccurredAt: time.Now(),
		}
		// We log error but don't fail the request as notification is auxiliary
		if err := c.eventPublisher.Publish(ctx, evt); err != nil {
			fmt.Printf("[WARN] Failed to publish NOTE_CREATED event: %v\n", err)
		}
	}

	return &dto.CreateNoteResponse{
		Id: note.Id,
	}, nil
}

func (c *noteService) Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	note, err := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, nil // Not found
	}

	// Build breadcrumb: traverse notebook ancestry from note's parent to root
	breadcrumb, err := c.buildBreadcrumb(ctx, uow, note.NotebookId, userId)
	if err != nil {
		return nil, err
	}

	res := dto.ShowNoteResponse{
		Id:         note.Id,
		Title:      note.Title,
		Content:    note.Content,
		NotebookId: note.NotebookId,
		Breadcrumb: breadcrumb,
		CreatedAt:  note.CreatedAt,
		UpdatedAt:  note.UpdatedAt,
	}

	return &res, nil
}

// buildBreadcrumb traverses notebook parent_id chain to build ancestry path from root to parent.
// This enables deep linking: frontend can display breadcrumbs and auto-expand sidebar tree.
func (c *noteService) buildBreadcrumb(ctx context.Context, uow unitofwork.UnitOfWork, notebookId uuid.UUID, userId uuid.UUID) ([]dto.BreadcrumbItem, error) {
	var breadcrumb []dto.BreadcrumbItem
	currentId := &notebookId

	for currentId != nil {
		notebook, err := uow.NotebookRepository().FindOne(ctx,
			specification.ByID{ID: *currentId},
			specification.UserOwnedBy{UserID: userId},
		)
		if err != nil {
			return nil, err
		}
		if notebook == nil {
			break // Safety: orphaned reference or ownership mismatch
		}

		// Prepend to build root-first order
		breadcrumb = append([]dto.BreadcrumbItem{{
			Id:   notebook.Id,
			Name: notebook.Name,
		}}, breadcrumb...)

		currentId = notebook.ParentId
	}

	return breadcrumb, nil
}

func (c *noteService) Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	note, err := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: req.Id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, nil
	}

	now := time.Now()

	note.Title = req.Title
	note.Content = req.Content
	note.UpdatedAt = &now

	err = uow.NoteRepository().Update(ctx, note)
	if err != nil {
		return nil, err
	}

	payload := dto.PublishEmbedNoteMessage{
		NoteId: note.Id,
	}
	payloadJson, _ := json.Marshal(payload)
	err = c.publisherService.Publish(ctx, payloadJson)
	if err != nil {
		return nil, err
	}

	return &dto.UpdateNoteResponse{
		Id: note.Id,
	}, nil
}

func (c *noteService) Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	note, err := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return err
	}
	if note == nil {
		return nil
	}

	if err := uow.Begin(ctx); err != nil {
		return err
	}
	defer uow.Rollback()

	if err := uow.NoteRepository().Delete(ctx, id); err != nil {
		return err
	}

	if err := uow.NoteEmbeddingRepository().DeleteByNoteId(ctx, id); err != nil {
		return err
	}

	return uow.Commit()
}

func (c *noteService) MoveNote(ctx context.Context, userId uuid.UUID, req *dto.MoveNoteRequest) (*dto.MoveNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	note, err := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: req.Id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, nil
	}

	now := time.Now()
	note.NotebookId = req.NotebookId
	note.UpdatedAt = &now

	err = uow.NoteRepository().Update(ctx, note)
	if err != nil {
		return nil, err
	}

	payload := dto.PublishEmbedNoteMessage{
		NoteId: note.Id,
	}
	payloadJson, _ := json.Marshal(payload)
	err = c.publisherService.Publish(ctx, payloadJson)
	if err != nil {
		return nil, err
	}

	return &dto.MoveNoteResponse{
		Id: note.Id,
	}, nil
}

// getSemanticSearchThreshold reads threshold from ai_configurations table.
// Falls back to 0.35 (original default) if not configured.
func (c *noteService) getSemanticSearchThreshold(ctx context.Context, uow unitofwork.UnitOfWork) float64 {
	const defaultThreshold = 0.35 // Original default - balanced for recall

	config, err := uow.AiConfigRepository().FindConfigurationByKey(ctx, "rag_similarity_threshold")
	if err != nil || config == nil {
		return defaultThreshold
	}

	val, err := strconv.ParseFloat(config.Value, 64)
	if err != nil {
		return defaultThreshold
	}

	return val // Respect configured value without floor
}

func (c *noteService) SemanticSearch(ctx context.Context, userId uuid.UUID, search string) ([]*dto.SemanticSearchResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	// Verify Semantic Search Access and Limits
	if err := c.accessVerifier.VerifySemanticSearchAccess(ctx, uow, userId); err != nil {
		return nil, err
	}

	var err error // Fix undefined err

	var notes []*entity.Note
	var searchType string
	scoreMap := make(map[uuid.UUID]float64) // Track scores for semantic results

	// === SLASH COMMAND PARSING ===
	// Extract filters like /nb:, /note:
	filters := pkgSearch.ParseQuery(search)
	hasFilters := filters.NotebookName != "" || filters.NoteTitle != ""

	if hasFilters {
		// STRATEGY: LITERAL FILTER (Bypass AI)
		searchType = "literal_filter"

		specs := []specification.Specification{
			specification.NoteOwnedByUser{UserID: userId}, // Explicit table alias to avoid ambiguity
		}

		if filters.NotebookName != "" {
			specs = append(specs, specification.ByNotebookName{Name: filters.NotebookName})
		}
		if filters.NoteTitle != "" {
			specs = append(specs, specification.ByNoteTitle{Title: filters.NoteTitle})
		}
		if filters.SearchQuery != "" {
			// If there's text remaining, search it in Title OR Content
			specs = append(specs, specification.NoteSearchQuery{Query: filters.SearchQuery})
		}

		notes, err = uow.NoteRepository().FindAll(ctx, specs...)
		if err != nil {
			return nil, err
		}

	} else {
		// === SMART SEARCH STRATEGY ===
		// No manual filters -> decide between Literal or Semantic based on query
		strategy := pkgSearch.DetermineStrategy(search)

		if strategy == pkgSearch.StrategyLiteral {
			searchType = "literal"
			// Literal Search: SQL ILIKE
			notes, err = uow.NoteRepository().FindAll(ctx,
				specification.UserOwnedBy{UserID: userId},
				specification.NoteSearchQuery{Query: search},
			)
			if err != nil {
				return nil, err
			}
		} else {
			searchType = "semantic"
			// Semantic Search: Vector Embedding
			embeddingRes, err := c.embeddingProvider.Generate(
				search,
				"RETRIEVAL_QUERY",
			)
			if err != nil {
				return nil, err
			}

			// Get threshold from configuration (not hardcoded)
			threshold := c.getSemanticSearchThreshold(ctx, uow)

			// Search Similar WITH SCORE (Threshold filtering)
			scoredResults, err := uow.NoteEmbeddingRepository().SearchSimilarWithScore(ctx, embeddingRes.Embedding.Values, 10, userId, threshold)
			if err != nil {
				return nil, err
			}

			if len(scoredResults) == 0 {
				return []*dto.SemanticSearchResponse{}, nil
			}

			// Extract Note IDs and track scores (Deduplicated)
			ids := make([]uuid.UUID, 0)
			seen := make(map[uuid.UUID]bool)

			for _, sr := range scoredResults {
				if !seen[sr.Embedding.NoteId] {
					ids = append(ids, sr.Embedding.NoteId)
					seen[sr.Embedding.NoteId] = true
					scoreMap[sr.Embedding.NoteId] = sr.Similarity
				}
			}

			fetchedNotes, err := uow.NoteRepository().FindAll(ctx, specification.ByIDs{IDs: ids}, specification.UserOwnedBy{UserID: userId})
			if err != nil {
				return nil, err
			}

			// Preserve order of Scored Results (highly relevant first)
			notes = make([]*entity.Note, 0)

			// Re-use seen map to ensure we don't add the same note twice
			// (if multiple chunks of the same note are in top K)
			added := make(map[uuid.UUID]bool)

			for _, sr := range scoredResults {
				if added[sr.Embedding.NoteId] {
					continue
				}
				for _, note := range fetchedNotes {
					if sr.Embedding.NoteId == note.Id {
						notes = append(notes, note)
						added[note.Id] = true
						break
					}
				}
			}
		}
	}

	// === NORMALIZATION ===
	// Convert Raw Lexical JSON -> Plain Text for Frontend
	response := make([]*dto.SemanticSearchResponse, 0)
	for _, note := range notes {
		parsedContent := lexical.ParseContent(note.Content)

		resp := &dto.SemanticSearchResponse{
			Id:         note.Id,
			Title:      note.Title,
			Content:    parsedContent, // <-- SENT AS PLAIN TEXT
			NotebookId: note.NotebookId,
			CreatedAt:  note.CreatedAt,
			UpdatedAt:  note.UpdatedAt,
			SearchType: searchType, // <-- INJECTED INDICATOR
		}

		// Include relevance score for semantic search results
		if score, ok := scoreMap[note.Id]; ok {
			resp.RelevanceScore = &score
		}

		response = append(response, resp)
	}

	// Increment Usage after successful search
	// We do this asynchronously or synchronously? Safest to do sync to block spam.
	if err := c.accessVerifier.IncrementSemanticSearchUsage(ctx, uow, userId); err != nil {
		// Log error but don't fail the request as data is already retrieved
		// ideally logging it
		fmt.Printf("Error incrementing usage: %v\n", err)
	}

	return response, nil
}
