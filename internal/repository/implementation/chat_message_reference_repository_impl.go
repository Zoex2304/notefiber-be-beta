package implementation

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/contract"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatMessageReferenceRepositoryImpl struct {
	db *gorm.DB
}

func NewChatMessageReferenceRepository(db *gorm.DB) contract.ChatMessageReferenceRepository {
	return &ChatMessageReferenceRepositoryImpl{
		db: db,
	}
}

func (r *ChatMessageReferenceRepositoryImpl) Create(ctx context.Context, reference *entity.ChatMessageReference) error {
	return r.db.WithContext(ctx).Create(reference).Error
}

func (r *ChatMessageReferenceRepositoryImpl) CreateBulk(ctx context.Context, references []*entity.ChatMessageReference) error {
	if len(references) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&references).Error
}

func (r *ChatMessageReferenceRepositoryImpl) FindAllByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatMessageReference, error) {
	if len(messageIds) == 0 {
		return []*entity.ChatMessageReference{}, nil
	}
	var references []*entity.ChatMessageReference
	// Preload Note to get Title
	err := r.db.WithContext(ctx).
		Preload("Note").
		Where("chat_message_id IN ?", messageIds).
		Find(&references).Error
	if err != nil {
		return nil, err
	}
	return references, nil
}

// DeleteByChatSessionId deletes all references associated with messages in a given session
// Optimally, we can do a JOIN delete or subquery delete.
// DELETE FROM chat_message_references WHERE chat_message_id IN (SELECT id FROM chat_messages WHERE chat_session_id = ?)
func (r *ChatMessageReferenceRepositoryImpl) DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("chat_message_id IN (?)", r.db.Table("chat_messages").Select("id").Where("chat_session_id = ?", sessionId)).
		Delete(&entity.ChatMessageReference{}).Error
}
