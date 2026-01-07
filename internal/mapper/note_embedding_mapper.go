package mapper

import (
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type NoteEmbeddingMapper struct{}

func NewNoteEmbeddingMapper() *NoteEmbeddingMapper {
	return &NoteEmbeddingMapper{}
}

func (m *NoteEmbeddingMapper) ToEntity(e *model.NoteEmbedding) *entity.NoteEmbedding {
	if e == nil {
		return nil
	}

	var deletedAt *time.Time
	if e.DeletedAt.Valid {
		t := e.DeletedAt.Time
		deletedAt = &t
	}

	var updatedAt *time.Time
	if !e.UpdatedAt.IsZero() {
		t := e.UpdatedAt
		updatedAt = &t
	}

	return &entity.NoteEmbedding{
		Id:             e.Id,
		Document:       e.Document,
		EmbeddingValue: e.EmbeddingValue.Slice(),
		NoteId:         e.NoteId,
		ChunkIndex:     e.ChunkIndex, // Added
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      updatedAt,
		DeletedAt:      deletedAt,
		IsDeleted:      e.DeletedAt.Valid,
	}
}

func (m *NoteEmbeddingMapper) ToModel(e *entity.NoteEmbedding) *model.NoteEmbedding {
	if e == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if e.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *e.DeletedAt, Valid: true}
	} else if e.IsDeleted {
		deletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}

	var updatedAt time.Time
	if e.UpdatedAt != nil {
		updatedAt = *e.UpdatedAt
	}

	return &model.NoteEmbedding{
		Id:             e.Id,
		Document:       e.Document,
		EmbeddingValue: pgvector.NewVector(e.EmbeddingValue),
		NoteId:         e.NoteId,
		ChunkIndex:     e.ChunkIndex, // Added
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      updatedAt,
		DeletedAt:      deletedAt,
	}
}

func (m *NoteEmbeddingMapper) ToEntities(embeddings []*model.NoteEmbedding) []*entity.NoteEmbedding {
	entities := make([]*entity.NoteEmbedding, len(embeddings))
	for i, e := range embeddings {
		entities[i] = m.ToEntity(e)
	}
	return entities
}

func (m *NoteEmbeddingMapper) ToModels(embeddings []*entity.NoteEmbedding) []*model.NoteEmbedding {
	models := make([]*model.NoteEmbedding, len(embeddings))
	for i, e := range embeddings {
		models[i] = m.ToModel(e)
	}
	return models
}
