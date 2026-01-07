package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"

	"github.com/google/uuid"
)

type ChatMessageReferenceRepository interface {
	Create(ctx context.Context, reference *entity.ChatMessageReference) error
	CreateBulk(ctx context.Context, references []*entity.ChatMessageReference) error
	FindAllByMessageIds(ctx context.Context, messageIds []uuid.UUID) ([]*entity.ChatMessageReference, error)
	DeleteByChatSessionId(ctx context.Context, sessionId uuid.UUID) error
}
