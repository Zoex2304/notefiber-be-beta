package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

// IAiConfigRepository defines AI configuration repository operations
type IAiConfigRepository interface {
	// Configuration methods
	FindAllConfigurations(ctx context.Context, specs ...specification.Specification) ([]*entity.AiConfiguration, error)
	FindConfigurationByKey(ctx context.Context, key string) (*entity.AiConfiguration, error)
	UpdateConfiguration(ctx context.Context, config *entity.AiConfiguration) error
	CreateConfiguration(ctx context.Context, config *entity.AiConfiguration) error

	// Nuance methods
	FindAllNuances(ctx context.Context, specs ...specification.Specification) ([]*entity.AiNuance, error)
	FindNuanceByKey(ctx context.Context, key string) (*entity.AiNuance, error)
	FindNuanceById(ctx context.Context, id uuid.UUID) (*entity.AiNuance, error)
	CreateNuance(ctx context.Context, nuance *entity.AiNuance) error
	UpdateNuance(ctx context.Context, nuance *entity.AiNuance) error
	DeleteNuance(ctx context.Context, id uuid.UUID) error
}
