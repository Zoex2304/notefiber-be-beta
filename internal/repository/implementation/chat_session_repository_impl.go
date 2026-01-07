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
	"gorm.io/gorm"
)

type ChatSessionRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.ChatMapper
}

func NewChatSessionRepository(db *gorm.DB) contract.ChatSessionRepository {
	return &ChatSessionRepositoryImpl{
		db:     db,
		mapper: mapper.NewChatMapper(),
	}
}

func (r *ChatSessionRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *ChatSessionRepositoryImpl) Create(ctx context.Context, session *entity.ChatSession) error {
	m := r.mapper.ChatSessionToModel(session)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*session = *r.mapper.ChatSessionToEntity(m)
	return nil
}

func (r *ChatSessionRepositoryImpl) Update(ctx context.Context, session *entity.ChatSession) error {
	m := r.mapper.ChatSessionToModel(session)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*session = *r.mapper.ChatSessionToEntity(m)
	return nil
}

func (r *ChatSessionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ChatSession{}, id).Error
}

func (r *ChatSessionRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.ChatSession{}).Error
}

func (r *ChatSessionRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.ChatSession, error) {
	var m model.ChatSession
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ChatSessionToEntity(&m), nil
}

func (r *ChatSessionRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.ChatSession, error) {
	var models []*model.ChatSession
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	// Manual slice mapping needed since ChatMapper doesn't have ToEntities equivalent for session
	entities := make([]*entity.ChatSession, len(models))
	for i, m := range models {
		entities[i] = r.mapper.ChatSessionToEntity(m)
	}
	return entities, nil
}

func (r *ChatSessionRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.ChatSession{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
