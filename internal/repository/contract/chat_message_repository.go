package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type ChatMessageRepository interface {
	Create(ctx context.Context, message *entity.ChatMessage) error
	Update(ctx context.Context, message *entity.ChatMessage) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteUnscoped(ctx context.Context, id uuid.UUID) error // Hard delete
	DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error
	DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error          // Hard delete messages
	DeleteAllCitationsByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete citations
	CreateCitations(ctx context.Context, citations []*entity.ChatCitation) error
	FindCitationsByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatCitation, error)
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.ChatMessage, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.ChatMessage, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
}
