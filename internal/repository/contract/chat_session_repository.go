package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type ChatSessionRepository interface {
	Create(ctx context.Context, session *entity.ChatSession) error
	Update(ctx context.Context, session *entity.ChatSession) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete all
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.ChatSession, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.ChatSession, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
}
