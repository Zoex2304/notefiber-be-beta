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

type ChatMessageRawRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.ChatMapper
}

func NewChatMessageRawRepository(db *gorm.DB) contract.ChatMessageRawRepository {
	return &ChatMessageRawRepositoryImpl{
		db:     db,
		mapper: mapper.NewChatMapper(),
	}
}

func (r *ChatMessageRawRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *ChatMessageRawRepositoryImpl) Create(ctx context.Context, message *entity.ChatMessageRaw) error {
	m := r.mapper.ChatMessageRawToModel(message)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*message = *r.mapper.ChatMessageRawToEntity(m)
	return nil
}

func (r *ChatMessageRawRepositoryImpl) Update(ctx context.Context, message *entity.ChatMessageRaw) error {
	m := r.mapper.ChatMessageRawToModel(message)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*message = *r.mapper.ChatMessageRawToEntity(m)
	return nil
}

func (r *ChatMessageRawRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ChatMessageRaw{}, id).Error
}

func (r *ChatMessageRawRepositoryImpl) DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error {
	return r.db.WithContext(ctx).Where("chat_session_id = ?", sessionId).Delete(&model.ChatMessageRaw{}).Error
}

func (r *ChatMessageRawRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.ChatMessageRaw, error) {
	var m model.ChatMessageRaw
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ChatMessageRawToEntity(&m), nil
}

func (r *ChatMessageRawRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.ChatMessageRaw, error) {
	var models []*model.ChatMessageRaw
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	entities := make([]*entity.ChatMessageRaw, len(models))
	for i, m := range models {
		entities[i] = r.mapper.ChatMessageRawToEntity(m)
	}
	return entities, nil
}

func (r *ChatMessageRawRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.ChatMessageRaw{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
