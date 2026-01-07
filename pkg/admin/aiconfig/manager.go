package aiconfig

import (
	"context"
	"fmt"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

// Manager handles AI configuration operations
type Manager struct{}

// NewManager creates a new AI config manager
func NewManager() *Manager {
	return &Manager{}
}

// ============================================================================
// Configuration Methods
// ============================================================================

// GetAllConfigurations retrieves all AI configurations
func (m *Manager) GetAllConfigurations(ctx context.Context, uow unitofwork.UnitOfWork) ([]*dto.AiConfigurationResponse, error) {
	configs, err := uow.AiConfigRepository().FindAllConfigurations(ctx)
	if err != nil {
		return nil, err
	}

	var responses []*dto.AiConfigurationResponse
	for _, c := range configs {
		responses = append(responses, configToResponse(c))
	}

	return responses, nil
}

// GetConfigurationByKey retrieves a configuration by key
func (m *Manager) GetConfigurationByKey(ctx context.Context, uow unitofwork.UnitOfWork, key string) (*dto.AiConfigurationResponse, error) {
	config, err := uow.AiConfigRepository().FindConfigurationByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("configuration with key '%s' not found", key)
	}

	return configToResponse(config), nil
}

// UpdateConfiguration updates a configuration value
func (m *Manager) UpdateConfiguration(ctx context.Context, uow unitofwork.UnitOfWork, key string, req dto.UpdateAiConfigurationRequest) (*dto.AiConfigurationResponse, error) {
	config, err := uow.AiConfigRepository().FindConfigurationByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("configuration with key '%s' not found", key)
	}

	config.Value = req.Value

	if err := uow.AiConfigRepository().UpdateConfiguration(ctx, config); err != nil {
		return nil, err
	}

	return configToResponse(config), nil
}

// ============================================================================
// Nuance Methods
// ============================================================================

// GetAllNuances retrieves all AI nuances
func (m *Manager) GetAllNuances(ctx context.Context, uow unitofwork.UnitOfWork) ([]*dto.AiNuanceResponse, error) {
	nuances, err := uow.AiConfigRepository().FindAllNuances(ctx)
	if err != nil {
		return nil, err
	}

	var responses []*dto.AiNuanceResponse
	for _, n := range nuances {
		responses = append(responses, nuanceToResponse(n))
	}

	return responses, nil
}

// GetNuanceByKey retrieves a nuance by key (for router)
func (m *Manager) GetNuanceByKey(ctx context.Context, uow unitofwork.UnitOfWork, key string) (*entity.AiNuance, error) {
	return uow.AiConfigRepository().FindNuanceByKey(ctx, key)
}

// CreateNuance creates a new nuance
func (m *Manager) CreateNuance(ctx context.Context, uow unitofwork.UnitOfWork, req dto.CreateAiNuanceRequest) (*dto.AiNuanceResponse, error) {
	// Check for duplicate key
	existing, err := uow.AiConfigRepository().FindNuanceByKey(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("nuance with key '%s' already exists", req.Key)
	}

	nuance := &entity.AiNuance{
		Key:           req.Key,
		Name:          req.Name,
		Description:   req.Description,
		SystemPrompt:  req.SystemPrompt,
		ModelOverride: req.ModelOverride,
		IsActive:      true,
		SortOrder:     req.SortOrder,
	}

	if err := uow.AiConfigRepository().CreateNuance(ctx, nuance); err != nil {
		return nil, err
	}

	return nuanceToResponse(nuance), nil
}

// UpdateNuance updates an existing nuance
func (m *Manager) UpdateNuance(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID, req dto.UpdateAiNuanceRequest) (*dto.AiNuanceResponse, error) {
	nuance, err := uow.AiConfigRepository().FindNuanceById(ctx, id)
	if err != nil {
		return nil, err
	}
	if nuance == nil {
		return nil, fmt.Errorf("nuance not found")
	}

	if req.Name != nil {
		nuance.Name = *req.Name
	}
	if req.Description != nil {
		nuance.Description = *req.Description
	}
	if req.SystemPrompt != nil {
		nuance.SystemPrompt = *req.SystemPrompt
	}
	if req.ModelOverride != nil {
		nuance.ModelOverride = req.ModelOverride
	}
	if req.IsActive != nil {
		nuance.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		nuance.SortOrder = *req.SortOrder
	}

	if err := uow.AiConfigRepository().UpdateNuance(ctx, nuance); err != nil {
		return nil, err
	}

	return nuanceToResponse(nuance), nil
}

// DeleteNuance removes a nuance
func (m *Manager) DeleteNuance(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID) error {
	nuance, err := uow.AiConfigRepository().FindNuanceById(ctx, id)
	if err != nil {
		return err
	}
	if nuance == nil {
		return fmt.Errorf("nuance not found")
	}

	return uow.AiConfigRepository().DeleteNuance(ctx, id)
}

// ============================================================================
// Mappers
// ============================================================================

func configToResponse(c *entity.AiConfiguration) *dto.AiConfigurationResponse {
	return &dto.AiConfigurationResponse{
		Id:          c.Id,
		Key:         c.Key,
		Value:       c.Value,
		ValueType:   c.ValueType,
		Description: c.Description,
		Category:    c.Category,
		UpdatedAt:   c.UpdatedAt,
	}
}

func nuanceToResponse(n *entity.AiNuance) *dto.AiNuanceResponse {
	return &dto.AiNuanceResponse{
		Id:            n.Id,
		Key:           n.Key,
		Name:          n.Name,
		Description:   n.Description,
		SystemPrompt:  n.SystemPrompt,
		ModelOverride: n.ModelOverride,
		IsActive:      n.IsActive,
		SortOrder:     n.SortOrder,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}
