package feature

import (
	"context"
	"fmt"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

// Manager handles feature catalog operations
type Manager struct{}

// NewManager creates a new feature manager
func NewManager() *Manager {
	return &Manager{}
}

// GetAll retrieves all features from the master catalog
func (m *Manager) GetAll(ctx context.Context, uow unitofwork.UnitOfWork) ([]*entity.Feature, error) {
	return uow.FeatureRepository().FindAll(ctx)
}

// Create creates a new feature in the master catalog
func (m *Manager) Create(ctx context.Context, uow unitofwork.UnitOfWork, req dto.CreateFeatureRequest) (*entity.Feature, error) {
	// Check for duplicate key
	existing, err := uow.FeatureRepository().FindByKey(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("feature with key '%s' already exists", req.Key)
	}

	feature := &entity.Feature{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		IsActive:    req.IsActive,
		SortOrder:   req.SortOrder,
	}

	if err := uow.FeatureRepository().Create(ctx, feature); err != nil {
		return nil, err
	}

	return feature, nil
}

// Update updates a feature in the master catalog
func (m *Manager) Update(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID, req dto.UpdateFeatureRequest) (*entity.Feature, error) {
	feature, err := uow.FeatureRepository().FindOne(ctx, specification.ByID{ID: id})
	if err != nil {
		return nil, err
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found")
	}

	if req.Name != nil {
		feature.Name = *req.Name
	}
	if req.Description != nil {
		feature.Description = *req.Description
	}
	if req.Category != nil {
		feature.Category = *req.Category
	}
	if req.IsActive != nil {
		feature.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		feature.SortOrder = *req.SortOrder
	}

	if err := uow.FeatureRepository().Update(ctx, feature); err != nil {
		return nil, err
	}

	return feature, nil
}

// Delete removes a feature from the master catalog
func (m *Manager) Delete(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID) error {
	feature, err := uow.FeatureRepository().FindOne(ctx, specification.ByID{ID: id})
	if err != nil {
		return err
	}
	if feature == nil {
		return fmt.Errorf("feature not found")
	}

	return uow.FeatureRepository().Delete(ctx, id)
}
