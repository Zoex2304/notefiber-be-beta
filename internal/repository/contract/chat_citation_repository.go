package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"

	"github.com/google/uuid"
)

type ChatCitationRepository interface {
	Create(ctx context.Context, citation *entity.ChatCitation) error
	CreateBulk(ctx context.Context, citations []*entity.ChatCitation) error
	FindAllByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatCitation, error)
	FindCitationsByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatCitation, error)
	DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error
	DeleteAllCitationsByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error
}
