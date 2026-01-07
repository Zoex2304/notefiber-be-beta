// FILE: internal/repository/contract/feature_repository.go
// Repository interface for Feature (master catalog)
package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type FeatureRepository interface {
	Create(ctx context.Context, feature *entity.Feature) error
	Update(ctx context.Context, feature *entity.Feature) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Feature, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Feature, error)
	FindByKey(ctx context.Context, key string) (*entity.Feature, error)
}
