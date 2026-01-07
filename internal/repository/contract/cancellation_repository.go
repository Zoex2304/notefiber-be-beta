// FILE: internal/repository/contract/cancellation_repository.go
package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

// CancellationRepository defines operations for subscription cancellation requests
type CancellationRepository interface {
	Create(ctx context.Context, cancellation *entity.Cancellation) error
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Cancellation, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Cancellation, error)
	FindAllWithDetails(ctx context.Context, specs ...specification.Specification) ([]*entity.Cancellation, error)
	Update(ctx context.Context, cancellation *entity.Cancellation) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete all
}
