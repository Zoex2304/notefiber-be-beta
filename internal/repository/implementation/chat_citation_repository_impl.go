package implementation

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/contract"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatCitationRepositoryImpl struct {
	db *gorm.DB
}

func NewChatCitationRepository(db *gorm.DB) contract.ChatCitationRepository {
	return &ChatCitationRepositoryImpl{
		db: db,
	}
}

func (r *ChatCitationRepositoryImpl) Create(ctx context.Context, citation *entity.ChatCitation) error {
	return r.db.WithContext(ctx).Create(citation).Error
}

func (r *ChatCitationRepositoryImpl) CreateBulk(ctx context.Context, citations []*entity.ChatCitation) error {
	if len(citations) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&citations).Error
}

func (r *ChatCitationRepositoryImpl) FindAllByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatCitation, error) {
	if len(messageIds) == 0 {
		return []*entity.ChatCitation{}, nil
	}
	var citations []*entity.ChatCitation
	// Preload Note to get Title
	err := r.db.WithContext(ctx).
		Preload("Note").
		Where("chat_message_id IN ?", messageIds).
		Find(&citations).Error
	if err != nil {
		return nil, err
	}
	return citations, nil
}

func (r *ChatCitationRepositoryImpl) FindCitationsByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatCitation, error) {
	return r.FindAllByMessageIds(ctx, messageIds)
}

func (r *ChatCitationRepositoryImpl) DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error {
	// Subquery delete strategy
	return r.db.WithContext(ctx).
		Where("chat_message_id IN (?)", r.db.Table("chat_messages").Select("id").Where("chat_session_id = ?", sessionId)).
		Delete(&entity.ChatCitation{}).Error
}

func (r *ChatCitationRepositoryImpl) DeleteAllCitationsByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	// Delete citations linked to messages linked to sessions owned by user
	subQuery := r.db.Table("chat_messages").
		Select("chat_messages.id").
		Joins("JOIN chat_sessions ON chat_messages.chat_session_id = chat_sessions.id").
		Where("chat_sessions.user_id = ?", userId)

	return r.db.WithContext(ctx).Unscoped().
		Where("chat_message_id IN (?)", subQuery).
		Delete(&entity.ChatCitation{}).Error
}
