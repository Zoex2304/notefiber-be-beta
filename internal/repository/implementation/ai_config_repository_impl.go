package implementation

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type aiConfigRepository struct {
	db *gorm.DB
}

// NewAiConfigRepository creates a new AI config repository
func NewAiConfigRepository(db *gorm.DB) contract.IAiConfigRepository {
	return &aiConfigRepository{db: db}
}

// applySpecifications applies all specifications to the query
func (r *aiConfigRepository) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

// ============================================================================
// Configuration Methods
// ============================================================================

func (r *aiConfigRepository) FindAllConfigurations(ctx context.Context, specs ...specification.Specification) ([]*entity.AiConfiguration, error) {
	var models []model.AiConfiguration
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)

	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	entities := make([]*entity.AiConfiguration, len(models))
	for i, m := range models {
		entities[i] = configModelToEntity(&m)
	}

	return entities, nil
}

func (r *aiConfigRepository) FindConfigurationByKey(ctx context.Context, key string) (*entity.AiConfiguration, error) {
	var m model.AiConfiguration
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return configModelToEntity(&m), nil
}

func (r *aiConfigRepository) UpdateConfiguration(ctx context.Context, config *entity.AiConfiguration) error {
	m := configEntityToModel(config)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *aiConfigRepository) CreateConfiguration(ctx context.Context, config *entity.AiConfiguration) error {
	if config.Id == uuid.Nil {
		config.Id = uuid.New()
	}
	m := configEntityToModel(config)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	config.Id = m.Id
	return nil
}

// ============================================================================
// Nuance Methods
// ============================================================================

func (r *aiConfigRepository) FindAllNuances(ctx context.Context, specs ...specification.Specification) ([]*entity.AiNuance, error) {
	var models []model.AiNuance
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)

	// Default ordering by sort_order
	query = query.Order("sort_order ASC, created_at ASC")

	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	entities := make([]*entity.AiNuance, len(models))
	for i, m := range models {
		entities[i] = nuanceModelToEntity(&m)
	}

	return entities, nil
}

func (r *aiConfigRepository) FindNuanceByKey(ctx context.Context, key string) (*entity.AiNuance, error) {
	var m model.AiNuance
	if err := r.db.WithContext(ctx).Where("key = ? AND is_active = true", key).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return nuanceModelToEntity(&m), nil
}

func (r *aiConfigRepository) FindNuanceById(ctx context.Context, id uuid.UUID) (*entity.AiNuance, error) {
	var m model.AiNuance
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return nuanceModelToEntity(&m), nil
}

func (r *aiConfigRepository) CreateNuance(ctx context.Context, nuance *entity.AiNuance) error {
	if nuance.Id == uuid.Nil {
		nuance.Id = uuid.New()
	}
	m := nuanceEntityToModel(nuance)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	nuance.Id = m.Id
	return nil
}

func (r *aiConfigRepository) UpdateNuance(ctx context.Context, nuance *entity.AiNuance) error {
	m := nuanceEntityToModel(nuance)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *aiConfigRepository) DeleteNuance(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.AiNuance{}, "id = ?", id).Error
}

// ============================================================================
// Mappers
// ============================================================================

func configModelToEntity(m *model.AiConfiguration) *entity.AiConfiguration {
	return &entity.AiConfiguration{
		Id:          m.Id,
		Key:         m.Key,
		Value:       m.Value,
		ValueType:   m.ValueType,
		Description: m.Description,
		Category:    m.Category,
		IsSecret:    m.IsSecret,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func configEntityToModel(e *entity.AiConfiguration) *model.AiConfiguration {
	return &model.AiConfiguration{
		Id:          e.Id,
		Key:         e.Key,
		Value:       e.Value,
		ValueType:   e.ValueType,
		Description: e.Description,
		Category:    e.Category,
		IsSecret:    e.IsSecret,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func nuanceModelToEntity(m *model.AiNuance) *entity.AiNuance {
	return &entity.AiNuance{
		Id:            m.Id,
		Key:           m.Key,
		Name:          m.Name,
		Description:   m.Description,
		SystemPrompt:  m.SystemPrompt,
		ModelOverride: m.ModelOverride,
		IsActive:      m.IsActive,
		SortOrder:     m.SortOrder,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func nuanceEntityToModel(e *entity.AiNuance) *model.AiNuance {
	return &model.AiNuance{
		Id:            e.Id,
		Key:           e.Key,
		Name:          e.Name,
		Description:   e.Description,
		SystemPrompt:  e.SystemPrompt,
		ModelOverride: e.ModelOverride,
		IsActive:      e.IsActive,
		SortOrder:     e.SortOrder,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}
