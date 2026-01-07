package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type ChatMessageRawRepository interface {
	Create(ctx context.Context, message *entity.ChatMessageRaw) error
	Update(ctx context.Context, message *entity.ChatMessageRaw) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.ChatMessageRaw, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.ChatMessageRaw, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
}
