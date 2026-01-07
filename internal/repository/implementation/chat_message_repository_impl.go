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

type ChatMessageRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.ChatMapper
}

func NewChatMessageRepository(db *gorm.DB) contract.ChatMessageRepository {
	return &ChatMessageRepositoryImpl{
		db:     db,
		mapper: mapper.NewChatMapper(),
	}
}

func (r *ChatMessageRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *ChatMessageRepositoryImpl) Create(ctx context.Context, message *entity.ChatMessage) error {
	m := r.mapper.ChatMessageToModel(message)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*message = *r.mapper.ChatMessageToEntity(m)
	return nil
}

func (r *ChatMessageRepositoryImpl) Update(ctx context.Context, message *entity.ChatMessage) error {
	m := r.mapper.ChatMessageToModel(message)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*message = *r.mapper.ChatMessageToEntity(m)
	return nil
}

func (r *ChatMessageRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ChatMessage{}, id).Error
}

func (r *ChatMessageRepositoryImpl) DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error {
	return r.db.WithContext(ctx).Where("chat_session_id = ?", sessionId).Delete(&model.ChatMessage{}).Error
}

func (r *ChatMessageRepositoryImpl) DeleteUnscoped(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.ChatMessage{}, id).Error
}

func (r *ChatMessageRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	// Subquery to find session IDs for the user
	subQuery := r.db.Table("chat_sessions").Select("id").Where("user_id = ?", userId)
	return r.db.WithContext(ctx).Unscoped().Where("chat_session_id IN (?)", subQuery).Delete(&model.ChatMessage{}).Error
}

func (r *ChatMessageRepositoryImpl) DeleteAllCitationsByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	// Subquery to find message IDs for the user (via sessions)
	subQuerySessions := r.db.Table("chat_sessions").Select("id").Where("user_id = ?", userId)
	subQueryMessages := r.db.Table("chat_messages").Select("id").Where("chat_session_id IN (?)", subQuerySessions)

	// Delete citations linked to those messages
	return r.db.WithContext(ctx).Unscoped().Where("chat_message_id IN (?)", subQueryMessages).Delete(&model.ChatCitation{}).Error
}

func (r *ChatMessageRepositoryImpl) CreateCitations(ctx context.Context, citations []*entity.ChatCitation) error {
	// Mapper if needed, but model alias is direct
	models := make([]*model.ChatCitation, len(citations))
	for i, c := range citations {
		// Assuming simple cast/assignment since types match via alias
		m := model.ChatCitation(*c)
		models[i] = &m
	}
	return r.db.WithContext(ctx).Create(models).Error
}

func (r *ChatMessageRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.ChatMessage, error) {
	var m model.ChatMessage
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ChatMessageToEntity(&m), nil
}

func (r *ChatMessageRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.ChatMessage, error) {
	var models []*model.ChatMessage
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	entities := make([]*entity.ChatMessage, len(models))
	for i, m := range models {
		entities[i] = r.mapper.ChatMessageToEntity(m)
	}
	return entities, nil
}

func (r *ChatMessageRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.ChatMessage{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindCitationsByMessageIds finds all citations for the given message IDs with note titles
func (r *ChatMessageRepositoryImpl) FindCitationsByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatCitation, error) {
	if len(messageIds) == 0 {
		return []*entity.ChatCitation{}, nil
	}

	var models []*model.ChatCitation
	err := r.db.WithContext(ctx).
		Preload("Note").
		Where("chat_message_id IN ?", messageIds).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	entities := make([]*entity.ChatCitation, len(models))
	for i, m := range models {
		e := entity.ChatCitation(*m)
		entities[i] = &e
	}
	return entities, nil
}
