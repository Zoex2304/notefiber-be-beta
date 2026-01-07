// FILE: internal/service/plan_service.go
// Service for plan management and usage limit checking
package service

import (
	"context"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

type PlanService interface {
	// Public
	GetAllActivePlansWithFeatures(ctx context.Context) ([]*dto.PlanWithFeaturesResponse, error)

	// User
	GetUserUsageStatus(ctx context.Context, userId uuid.UUID) (*dto.UsageStatusResponse, error)
	CheckCanCreateNotebook(ctx context.Context, userId uuid.UUID) error
	CheckCanCreateNote(ctx context.Context, userId uuid.UUID, notebookId uuid.UUID) error
}

type planService struct {
	uowFactory unitofwork.RepositoryFactory
}

func NewPlanService(uowFactory unitofwork.RepositoryFactory) PlanService {
	return &planService{
		uowFactory: uowFactory,
	}
}

// GetAllActivePlansWithFeatures returns all active plans with their features for pricing modal
func (s *planService) GetAllActivePlansWithFeatures(ctx context.Context) ([]*dto.PlanWithFeaturesResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Get all plans (we'll filter active ones in code for now)
	plans, err := uow.SubscriptionRepository().FindAllPlans(ctx)
	if err != nil {
		return nil, err
	}

	var result []*dto.PlanWithFeaturesResponse
	for _, plan := range plans {
		if !plan.IsActive {
			continue
		}

		// Features are already preloaded by repository
		featureDTOs := make([]dto.FeatureDTO, 0, len(plan.Features))
		for _, f := range plan.Features {
			featureDTOs = append(featureDTOs, dto.FeatureDTO{
				Key:       f.Key,
				Text:      f.Name, // Using Name as DisplayText
				IsEnabled: true,   // Existence implies enabled
			})
		}

		result = append(result, &dto.PlanWithFeaturesResponse{
			Id:            plan.Id,
			Name:          plan.Name,
			Slug:          plan.Slug,
			Tagline:       plan.Tagline,
			Price:         plan.Price,
			BillingPeriod: string(plan.BillingPeriod),
			IsMostPopular: plan.IsMostPopular,
			Limits: dto.PlanLimitsDTO{
				MaxNotebooks:        plan.MaxNotebooks,
				MaxNotesPerNotebook: plan.MaxNotesPerNotebook,
				AiChatDaily:         plan.AiChatDailyLimit,
				SemanticSearchDaily: plan.SemanticSearchDailyLimit,
			},
			Features: featureDTOs,
		})
	}

	return result, nil
}

// GetUserUsageStatus returns current usage vs limits for a user
func (s *planService) GetUserUsageStatus(ctx context.Context, userId uuid.UUID) (*dto.UsageStatusResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Get user
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil {
		return nil, err
	}

	// Get user's active plan
	plan, err := s.getUserPlan(ctx, uow, userId)
	if err != nil {
		return nil, err
	}

	// Count current usage
	notebookCount, err := uow.NotebookRepository().Count(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		return nil, err
	}

	// Count total notes across all notebooks for the user
	noteCount, err := uow.NoteRepository().Count(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		return nil, err
	}

	// Check and reset daily usage if needed
	if err := s.checkAndResetDailyUsage(ctx, uow, user); err != nil {
		return nil, err
	}

	// Calculate reset time (next midnight)
	now := time.Now()
	resetTime := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	response := &dto.UsageStatusResponse{
		Plan: dto.PlanInfo{
			Id:   plan.Id,
			Name: plan.Name,
			Slug: plan.Slug,
		},
		Storage: dto.StorageLimits{
			Notebooks: dto.UsageLimit{
				Used:   int(notebookCount),
				Limit:  plan.MaxNotebooks,
				CanUse: plan.MaxNotebooks < 0 || int(notebookCount) < plan.MaxNotebooks,
			},
			Notes: dto.UsageLimit{
				Used:   int(noteCount),
				Limit:  plan.MaxNotesPerNotebook,
				CanUse: plan.MaxNotesPerNotebook < 0 || int(noteCount) < plan.MaxNotesPerNotebook,
			},
		},
		Daily: dto.DailyLimits{
			AiChat: dto.UsageLimit{
				Used:     user.AiDailyUsage,
				Limit:    s.getEffectiveLimit(plan.AiChatDailyLimit, user.AiDailyLimitOverride),
				CanUse:   s.canUseLimit(user.AiDailyUsage, s.getEffectiveLimit(plan.AiChatDailyLimit, user.AiDailyLimitOverride)),
				ResetsAt: &resetTime,
			},
			SemanticSearch: dto.UsageLimit{
				Used:     user.SemanticSearchDailyUsage,
				Limit:    plan.SemanticSearchDailyLimit,
				CanUse:   s.canUseLimit(user.SemanticSearchDailyUsage, plan.SemanticSearchDailyLimit),
				ResetsAt: &resetTime,
			},
		},
		UpgradeAvailable: plan.Slug == "free",
	}

	return response, nil
}

// checkAndResetDailyUsage checks if the daily usage needs to be reset based on calendar day
func (s *planService) checkAndResetDailyUsage(ctx context.Context, uow unitofwork.UnitOfWork, user *entity.User) error {
	now := time.Now()
	lastReset := user.AiDailyUsageLastReset

	// Check if the last reset was on a different calendar day
	// We compare Year, Month, and Day. If any differ, it's a new day.
	if now.Year() != lastReset.Year() || now.Month() != lastReset.Month() || now.Day() != lastReset.Day() {
		// If last reset was properly initialized (not zero time), and today is different
		// Or if it's zero time (never reset), we treat it as reset needed if usage > 0
		// But usually we just want to ensure it counts for *today*.

		// Logic: If the stored "last reset" timestamp is NOT today, then the usage stored
		// belongs to a previous day. So we reset it.

		user.AiDailyUsage = 0
		user.AiDailyUsageLastReset = now

		if err := uow.UserRepository().Update(ctx, user); err != nil {
			return err
		}
	}
	return nil
}

// CheckCanCreateNotebook checks if user can create a new notebook
func (s *planService) CheckCanCreateNotebook(ctx context.Context, userId uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	plan, err := s.getUserPlan(ctx, uow, userId)
	if err != nil {
		return err
	}

	// -1 means unlimited
	if plan.MaxNotebooks < 0 {
		return nil
	}

	count, err := uow.NotebookRepository().Count(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		return err
	}

	if int(count) >= plan.MaxNotebooks {
		return &dto.LimitExceededError{
			Limit: plan.MaxNotebooks,
			Used:  int(count),
		}
	}

	return nil
}

// CheckCanCreateNote checks if user can create a note in a notebook
func (s *planService) CheckCanCreateNote(ctx context.Context, userId uuid.UUID, notebookId uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	plan, err := s.getUserPlan(ctx, uow, userId)
	if err != nil {
		return err
	}

	// -1 means unlimited
	if plan.MaxNotesPerNotebook < 0 {
		return nil
	}

	count, err := uow.NoteRepository().Count(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.ByNotebookID{NotebookID: notebookId},
	)
	if err != nil {
		return err
	}

	if int(count) >= plan.MaxNotesPerNotebook {
		return &dto.LimitExceededError{
			Limit: plan.MaxNotesPerNotebook,
			Used:  int(count),
		}
	}

	return nil
}

// getUserPlan gets the user's current plan or returns default free plan
func (s *planService) getUserPlan(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) (*entity.SubscriptionPlan, error) {
	// Get all subscriptions for the user, ordered by creation (newest first)
	subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	var activeSub *entity.UserSubscription
	// Find the most recent active or paid subscription
	for _, sub := range subs {
		// Priority 1: Active
		if sub.Status == entity.SubscriptionStatusActive && sub.CurrentPeriodEnd.After(time.Now()) {
			activeSub = sub
			break
		}
		// Priority 2: Canceled but still within billing period (access retained)
		if sub.Status == entity.SubscriptionStatusCanceled && sub.CurrentPeriodEnd.After(time.Now()) {
			activeSub = sub
			break
		}
		// Priority 3: Just paid (fallback)
		if sub.PaymentStatus == entity.PaymentStatusPaid && sub.CurrentPeriodEnd.After(time.Now()) {
			activeSub = sub
			break
		}
	}

	if activeSub != nil {
		// Get the plan
		plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: activeSub.PlanId})
		if err != nil {
			return nil, err
		}
		if plan != nil {
			return plan, nil
		}
	}

	// Return default free plan limits
	return &entity.SubscriptionPlan{
		Name:                     "Free Plan",
		Slug:                     "free",
		MaxNotebooks:             3,
		MaxNotesPerNotebook:      10,
		AiChatDailyLimit:         0,
		SemanticSearchDailyLimit: 0,
		AiChatEnabled:            false,
		SemanticSearchEnabled:    false,
	}, nil
}

// Helper to get effective limit (override takes precedence)
func (s *planService) getEffectiveLimit(planLimit int, override *int) int {
	if override != nil {
		return *override
	}
	return planLimit
}

// Helper to check if usage is within limit
func (s *planService) canUseLimit(used int, limit int) bool {
	if limit < 0 {
		return true // Unlimited
	}
	return used < limit
}
