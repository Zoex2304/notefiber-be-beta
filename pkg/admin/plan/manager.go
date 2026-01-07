package plan

import (
	"context"
	"fmt"
	"strings"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

// Manager handles plan-related admin operations
type Manager struct{}

// NewManager creates a new plan manager
func NewManager() *Manager {
	return &Manager{}
}

// Create creates a new subscription plan
func (m *Manager) Create(ctx context.Context, uow unitofwork.UnitOfWork, req dto.AdminCreatePlanRequest) (*entity.SubscriptionPlan, error) {
	newPlan := &entity.SubscriptionPlan{
		Name:                     req.Name,
		Slug:                     req.Slug,
		Price:                    req.Price,
		TaxRate:                  req.TaxRate,
		BillingPeriod:            entity.BillingPeriod(req.BillingPeriod),
		MaxNotebooks:             req.Features.MaxNotebooks,
		MaxNotesPerNotebook:      req.Features.MaxNotesPerNotebook,
		SemanticSearchEnabled:    req.Features.SemanticSearch,
		AiChatEnabled:            req.Features.AiChat,
		AiChatDailyLimit:         req.Features.AiChatDailyLimit,
		SemanticSearchDailyLimit: req.Features.SemanticSearchDailyLimit,
		IsActive:                 true,
	}

	if err := uow.SubscriptionRepository().CreatePlan(ctx, newPlan); err != nil {
		return nil, err
	}

	return newPlan, nil
}

// Update updates a subscription plan
func (m *Manager) Update(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID, req dto.AdminUpdatePlanRequest) (*entity.SubscriptionPlan, error) {
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: id})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found")
	}

	// Basic Info
	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.Description != nil {
		plan.Description = *req.Description
	}
	if req.Tagline != nil {
		plan.Tagline = *req.Tagline
	}
	if req.Price != nil {
		plan.Price = *req.Price
	}
	if req.TaxRate != nil {
		plan.TaxRate = *req.TaxRate
	}

	// Display Settings
	if req.IsMostPopular != nil {
		plan.IsMostPopular = *req.IsMostPopular
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		plan.SortOrder = *req.SortOrder
	}

	// Features (limits and AI toggles)
	if req.Features != nil {
		plan.MaxNotebooks = req.Features.MaxNotebooks
		plan.MaxNotesPerNotebook = req.Features.MaxNotesPerNotebook
		plan.SemanticSearchEnabled = req.Features.SemanticSearch
		plan.AiChatEnabled = req.Features.AiChat
		plan.AiChatDailyLimit = req.Features.AiChatDailyLimit
		plan.SemanticSearchDailyLimit = req.Features.SemanticSearchDailyLimit
	}

	if err := uow.SubscriptionRepository().UpdatePlan(ctx, plan); err != nil {
		return nil, err
	}

	return plan, nil
}

// Delete removes a subscription plan
func (m *Manager) Delete(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID) error {
	err := uow.SubscriptionRepository().DeletePlan(ctx, id)
	if err != nil {
		// Check for FK violation (Postgres code 23503)
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "violates foreign key constraint") {
			return fmt.Errorf("cannot delete plan because it has active subscriptions. Please archive the plan instead")
		}
		return err
	}
	return nil
}

// FindAll retrieves all subscription plans
func (m *Manager) FindAll(ctx context.Context, uow unitofwork.UnitOfWork) ([]*entity.SubscriptionPlan, error) {
	return uow.SubscriptionRepository().FindAllPlans(ctx)
}

// FindOne retrieves a single plan by ID
func (m *Manager) FindOne(ctx context.Context, uow unitofwork.UnitOfWork, id uuid.UUID) (*entity.SubscriptionPlan, error) {
	return uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: id})
}

// GetFeatures retrieves features linked to a plan
func (m *Manager) GetFeatures(ctx context.Context, uow unitofwork.UnitOfWork, planId uuid.UUID) ([]entity.Feature, error) {
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: planId})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found")
	}

	return plan.Features, nil
}

// AddFeature links a feature to a plan
func (m *Manager) AddFeature(ctx context.Context, uow unitofwork.UnitOfWork, planId uuid.UUID, featureKey string) (*entity.Feature, error) {
	// Check if plan exists
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: planId})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found")
	}

	// Check if feature exists in master catalog
	feature, err := uow.FeatureRepository().FindByKey(ctx, featureKey)
	if err != nil {
		return nil, err
	}
	if feature == nil {
		return nil, fmt.Errorf("feature with key '%s' not found in catalog", featureKey)
	}

	// Link feature to plan
	if err := uow.SubscriptionRepository().AddFeatureToPlan(ctx, planId, feature.Id); err != nil {
		return nil, err
	}

	return feature, nil
}

// RemoveFeature unlinks a feature from a plan
func (m *Manager) RemoveFeature(ctx context.Context, uow unitofwork.UnitOfWork, planId uuid.UUID, featureId uuid.UUID) error {
	return uow.SubscriptionRepository().RemoveFeatureFromPlan(ctx, planId, featureId)
}
