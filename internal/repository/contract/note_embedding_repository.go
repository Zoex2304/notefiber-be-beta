package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

// ScoredNoteEmbedding wraps NoteEmbedding with its similarity score
type ScoredNoteEmbedding struct {
	Embedding  *entity.NoteEmbedding
	Similarity float64 // 0.0 to 1.0 (1.0 = identical)
}

type NoteEmbeddingRepository interface {
	Create(ctx context.Context, embedding *entity.NoteEmbedding) error
	CreateBulk(ctx context.Context, embeddings []*entity.NoteEmbedding) error
	Update(ctx context.Context, embedding *entity.NoteEmbedding) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete embeddings
	DeleteByNoteId(ctx context.Context, noteId uuid.UUID) error
	DeleteByNotebookId(ctx context.Context, notebookId uuid.UUID) error
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.NoteEmbedding, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.NoteEmbedding, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
	// Advanced
	SearchSimilar(ctx context.Context, embedding []float32, limit int, userId uuid.UUID) ([]*entity.NoteEmbedding, error)
	// SearchSimilarWithScore returns embeddings with their similarity scores, filtered by threshold
	SearchSimilarWithScore(ctx context.Context, embedding []float32, limit int, userId uuid.UUID, threshold float64) ([]*ScoredNoteEmbedding, error)
}
