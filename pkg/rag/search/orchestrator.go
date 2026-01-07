package search

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/lexical"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// Orchestrator handles vector search and candidate filtering
type Orchestrator struct {
	embeddingProvider embedding.EmbeddingProvider
	logger            *log.Logger
}

// NewOrchestrator creates a new search orchestrator
func NewOrchestrator(embeddingProvider embedding.EmbeddingProvider, logger *log.Logger) *Orchestrator {
	return &Orchestrator{
		embeddingProvider: embeddingProvider,
		logger:            logger,
	}
}

// Config encapsulates search parameters
type Config struct {
	DBThreshold    float64
	LogicThreshold float64
	TopK           int
}

// DefaultConfig returns default search configuration
func DefaultConfig() Config {
	return Config{
		DBThreshold:    0.0,
		LogicThreshold: 0.35,
		TopK:           10, // Increased from 5 to capture more relevant notes
	}
}

// Execute runs vector search and returns filtered candidates
func (o *Orchestrator) Execute(
	ctx context.Context,
	uow unitofwork.UnitOfWork,
	userId uuid.UUID,
	query string,
	config Config,
) ([]store.Document, error) {

	// Generate embedding
	embeddingRes, err := o.embeddingProvider.Generate(query, "RETRIEVAL_QUERY")
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", err)
	}

	// Execute vector search
	scoredResults, err := uow.NoteEmbeddingRepository().SearchSimilarWithScore(
		ctx,
		embeddingRes.Embedding.Values,
		config.TopK,
		userId,
		config.DBThreshold,
	)
	if err != nil {
		o.logger.Printf("[ERROR] Vector search failed: %v", err)
		return nil, err
	}

	o.logger.Printf("[DEBUG] Raw search results: %d documents", len(scoredResults))

	// Filter and deduplicate candidates
	candidates := o.filterAndDeduplicateCandidates(scoredResults, config.LogicThreshold)

	o.logger.Printf("[DEBUG] Filtered candidates: %d documents", len(candidates))

	// Hydrate with titles and content
	if err := o.hydrateCandidates(ctx, uow, candidates); err != nil {
		o.logger.Printf("[WARN] Failed to hydrate candidates: %v", err)
	}

	return candidates, nil
}

func (o *Orchestrator) filterAndDeduplicateCandidates(
	results []*contract.ScoredNoteEmbedding,
	threshold float64,
) []store.Document {

	var candidates []store.Document
	seen := make(map[string]bool)

	for i, res := range results {
		if res.Similarity >= threshold {
			noteId := res.Embedding.NoteId.String()

			if seen[noteId] {
				continue
			}

			candidates = append(candidates, store.Document{
				ID:      noteId,
				Content: ParseDocumentContent(res.Embedding.Document),
				Score:   float32(res.Similarity),
			})

			seen[noteId] = true

			status := "KEEP"
			o.logger.Printf("[DEBUG] Candidate %d: Score=%.4f [%s]", i+1, res.Similarity, status)
		} else {
			o.logger.Printf("[DEBUG] Candidate %d: Score=%.4f [FILTERED]", i+1, res.Similarity)
		}
	}

	return candidates
}

func (o *Orchestrator) hydrateCandidates(
	ctx context.Context,
	uow unitofwork.UnitOfWork,
	candidates []store.Document,
) error {

	if len(candidates) == 0 {
		return nil
	}

	noteIds := make([]uuid.UUID, len(candidates))
	for i, c := range candidates {
		noteIds[i] = uuid.MustParse(c.ID)
	}

	notes, err := uow.NoteRepository().FindAll(ctx, specification.ByIDs{IDs: noteIds})
	if err != nil {
		return err
	}

	// Build lookup maps
	titleMap := make(map[string]string)
	contentMap := make(map[string]string)
	for _, n := range notes {
		idStr := n.Id.String()
		titleMap[idStr] = n.Title
		contentMap[idStr] = n.Content
	}

	// Hydrate candidates
	for i := range candidates {
		if title, ok := titleMap[candidates[i].ID]; ok {
			candidates[i].Title = title
		} else {
			candidates[i].Title = "Untitled Note"
		}

		// If single candidate (auto-focus), include full content
		if len(candidates) == 1 {
			if content, ok := contentMap[candidates[i].ID]; ok {
				candidates[i].Content = content
			}
		}
	}

	return nil
}

// ParseDocumentContent extracts readable content from document format
func ParseDocumentContent(document string) string {
	parts := strings.SplitN(document, "\n\n", 3)
	if len(parts) < 3 {
		return lexical.ParseContent(document)
	}

	header := parts[0]
	content := parts[1]
	footer := parts[2]

	parsedContent := lexical.ParseContent(content)
	return fmt.Sprintf("%s\n\n%s\n\n%s", header, parsedContent, footer)
}
