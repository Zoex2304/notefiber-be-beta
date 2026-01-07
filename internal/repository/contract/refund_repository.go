package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type RefundRepository interface {
	Create(ctx context.Context, refund *entity.Refund) error
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Refund, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Refund, error)
	FindAllWithDetails(ctx context.Context, specs ...specification.Specification) ([]*entity.Refund, error)
	Update(ctx context.Context, refund *entity.Refund) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete all
}
