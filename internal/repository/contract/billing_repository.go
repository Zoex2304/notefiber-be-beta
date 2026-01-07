package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type BillingRepository interface {
	Create(ctx context.Context, billing *entity.BillingAddress) error
	Update(ctx context.Context, billing *entity.BillingAddress) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete all
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.BillingAddress, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.BillingAddress, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
}
