package implementation

import (
	"context"
	"errors"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/mapper"
	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type NoteEmbeddingRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.NoteEmbeddingMapper
}

func NewNoteEmbeddingRepository(db *gorm.DB) contract.NoteEmbeddingRepository {
	return &NoteEmbeddingRepositoryImpl{
		db:     db,
		mapper: mapper.NewNoteEmbeddingMapper(),
	}
}

func (r *NoteEmbeddingRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *NoteEmbeddingRepositoryImpl) Create(ctx context.Context, embedding *entity.NoteEmbedding) error {
	m := r.mapper.ToModel(embedding)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*embedding = *r.mapper.ToEntity(m)
	return nil
}

func (r *NoteEmbeddingRepositoryImpl) Update(ctx context.Context, embedding *entity.NoteEmbedding) error {
	m := r.mapper.ToModel(embedding)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*embedding = *r.mapper.ToEntity(m)
	return nil
}

func (r *NoteEmbeddingRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.NoteEmbedding{}, id).Error
}

func (r *NoteEmbeddingRepositoryImpl) DeleteByNoteId(ctx context.Context, noteId uuid.UUID) error {
	return r.db.WithContext(ctx).Where("note_id = ?", noteId).Delete(&model.NoteEmbedding{}).Error
}

func (r *NoteEmbeddingRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	// Subquery to find note IDs for the user
	subQuery := r.db.Table("notes").Select("id").Where("user_id = ?", userId)
	return r.db.WithContext(ctx).Unscoped().Where("note_id IN (?)", subQuery).Delete(&model.NoteEmbedding{}).Error
}

func (r *NoteEmbeddingRepositoryImpl) DeleteByNotebookId(ctx context.Context, notebookId uuid.UUID) error {
	subQuery := r.db.Table("notes").Select("id").Where("notebook_id = ?", notebookId)
	return r.db.WithContext(ctx).Where("note_id IN (?)", subQuery).Delete(&model.NoteEmbedding{}).Error
}

func (r *NoteEmbeddingRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.NoteEmbedding, error) {
	var m model.NoteEmbedding
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}

func (r *NoteEmbeddingRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.NoteEmbedding, error) {
	var models []*model.NoteEmbedding
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	entities := make([]*entity.NoteEmbedding, len(models))
	for i, m := range models {
		entities[i] = r.mapper.ToEntity(m)
	}
	return entities, nil
}

func (r *NoteEmbeddingRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	err := query.Model(&model.NoteEmbedding{}).Count(&count).Error
	return count, err
}

func (r *NoteEmbeddingRepositoryImpl) SearchSimilar(ctx context.Context, embedding []float32, limit int, userId uuid.UUID) ([]*entity.NoteEmbedding, error) {
	if limit <= 0 {
		limit = 5
	}
	var models []*model.NoteEmbedding

	// Using pgvector cosine distance: embedding_value <=> vector
	// We MUST join with 'notes' to filter by user_id
	// CRITICAL: Filter out soft-deleted embeddings AND notes
	err := r.db.WithContext(ctx).
		Joins("JOIN notes ON notes.id = note_embeddings.note_id").
		Where("notes.user_id = ?", userId).
		Where("note_embeddings.deleted_at IS NULL").
		Where("notes.deleted_at IS NULL").
		Order(gorm.Expr("embedding_value <=> ?", pgvector.NewVector(embedding))).
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	entities := make([]*entity.NoteEmbedding, len(models))
	for i, m := range models {
		entities[i] = r.mapper.ToEntity(m)
	}
	return entities, nil
}

// SearchSimilarWithScore returns embeddings with similarity scores, filtered by threshold
func (r *NoteEmbeddingRepositoryImpl) SearchSimilarWithScore(ctx context.Context, embedding []float32, limit int, userId uuid.UUID, threshold float64) ([]*contract.ScoredNoteEmbedding, error) {
	if limit <= 0 {
		limit = 5
	}

	// Raw query to get similarity score along with embeddings
	// Cosine distance in pgvector is: 1 - cosine_similarity
	// So we compute: 1 - (embedding_value <=> query_vector) = cosine_similarity
	type result struct {
		model.NoteEmbedding
		Similarity float64
	}
	var results []result

	queryVector := pgvector.NewVector(embedding)

	err := r.db.WithContext(ctx).
		Table("note_embeddings").
		Select("note_embeddings.*, 1 - (embedding_value <=> ?) as similarity", queryVector).
		Joins("JOIN notes ON notes.id = note_embeddings.note_id").
		Where("notes.user_id = ?", userId).
		Where("note_embeddings.deleted_at IS NULL").
		Where("notes.deleted_at IS NULL").
		Where("1 - (embedding_value <=> ?) >= ?", queryVector, threshold).
		Order("similarity DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	scoredEmbeddings := make([]*contract.ScoredNoteEmbedding, len(results))
	for i, res := range results {
		entity := r.mapper.ToEntity(&res.NoteEmbedding)
		scoredEmbeddings[i] = &contract.ScoredNoteEmbedding{
			Embedding:  entity,
			Similarity: res.Similarity,
		}
	}
	return scoredEmbeddings, nil
}

func (r *NoteEmbeddingRepositoryImpl) CreateBulk(ctx context.Context, embeddings []*entity.NoteEmbedding) error {
	models := make([]*model.NoteEmbedding, len(embeddings))
	for i, e := range embeddings {
		models[i] = r.mapper.ToModel(e)
	}

	if err := r.db.WithContext(ctx).Create(models).Error; err != nil {
		return err
	}

	// Update IDs back to entities
	for i, m := range models {
		*embeddings[i] = *r.mapper.ToEntity(m)
	}
	return nil
}
