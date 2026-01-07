package access

import (
	"context"
	"fmt"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

// Verifier handles access control and usage limits
type Verifier struct{}

// NewVerifier creates a new access verifier
func NewVerifier() *Verifier {
	return &Verifier{}
}

// VerifyAccessAndLimits checks user subscription and daily limits
func (v *Verifier) VerifyAccessAndLimits(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) error {
	// 1. Fetch User First (to check for override)
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	// 2. Fetch Subscription & Plan
	// 2. Fetch Subscription & Plan
	subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return err
	}
	var activeSub *entity.UserSubscription
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

	// Default to Free Plan logic if no active sub, but we need Plan object for feature flags unless overridden
	// If activeSub is nil, we can't get Plan from DB easily (unless we fetch "free" plan by slug).
	// But if Override is set, we might not need Plan?
	// Let's assume we proceed to check Plan only if no override or if we need base config.

	var aiLimit int = 0
	var aiEnabled bool = false

	if activeSub != nil {
		plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: activeSub.PlanId})
		if err == nil && plan != nil {
			aiLimit = plan.AiChatDailyLimit
			aiEnabled = plan.AiChatEnabled
		}
	}

	// 3. Apply Override
	if user.AiDailyLimitOverride != nil {
		aiLimit = *user.AiDailyLimitOverride
		aiEnabled = true // implied enabled if specific limit set
	}

	if !aiEnabled {
		return fmt.Errorf("feature requires pro plan")
	}

	// 4. Check Usage
	now := time.Now()
	// Reset logic
	// Reset logic
	// Check if the last reset was on a different calendar day
	// We compare Year, Month, and Day. If any differ, it's a new day.
	if now.Year() != user.AiDailyUsageLastReset.Year() || now.Month() != user.AiDailyUsageLastReset.Month() || now.Day() != user.AiDailyUsageLastReset.Day() {
		user.AiDailyUsage = 0
		user.AiDailyUsageLastReset = now
		if err := uow.UserRepository().Update(ctx, user); err != nil {
			return err
		}
	}

	// Check Limit (Limit < 0 means unlimited)
	if aiLimit >= 0 && user.AiDailyUsage >= aiLimit {
		resetTime := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		return &dto.LimitExceededError{
			Limit:      aiLimit,
			Used:       user.AiDailyUsage,
			ResetAfter: resetTime,
		}
	}

	return nil
}

// IncrementUserUsage increments daily AI usage counter
func (v *Verifier) IncrementUserUsage(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) error {
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	user.AiDailyUsage++
	return uow.UserRepository().Update(ctx, user)
}

// VerifySemanticSearchAccess checks user subscription and daily limits for semantic search
func (v *Verifier) VerifySemanticSearchAccess(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) error {
	// 1. Fetch User First
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	// 2. Fetch Subscription & Plan
	// 2. Fetch Subscription & Plan
	subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return err
	}
	var activeSub *entity.UserSubscription
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

	var searchLimit int = 0
	var searchEnabled bool = false

	if activeSub != nil {
		plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: activeSub.PlanId})
		if err == nil && plan != nil {
			searchLimit = plan.SemanticSearchDailyLimit
			searchEnabled = plan.SemanticSearchEnabled
		}
	}

	// 3. Apply Override (Currently using same override field or separate?
	// Assuming AiDailyLimitOverride might apply to Chat only, but let's check requirement.
	// For now, let's assume no specific override for search, or stick to plan limits.)
	// Wait, standardizing: If AI Limit override exists, does it apply to Search?
	// Usually separate. Let's assume NO override for search in User entity for now (it wasn't in the struct).

	if !searchEnabled {
		return fmt.Errorf("feature requires pro plan")
	}

	// 4. Check Usage & Reset Logic
	now := time.Now()
	if now.Year() != user.SemanticSearchDailyUsageLastReset.Year() || now.Month() != user.SemanticSearchDailyUsageLastReset.Month() || now.Day() != user.SemanticSearchDailyUsageLastReset.Day() {
		user.SemanticSearchDailyUsage = 0
		user.SemanticSearchDailyUsageLastReset = now
		if err := uow.UserRepository().Update(ctx, user); err != nil {
			return err
		}
	}

	// Check Limit (Limit < 0 means unlimited)
	if searchLimit >= 0 && user.SemanticSearchDailyUsage >= searchLimit {
		resetTime := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		return &dto.LimitExceededError{
			Limit:      searchLimit,
			Used:       user.SemanticSearchDailyUsage,
			ResetAfter: resetTime,
		}
	}

	return nil
}

// IncrementSemanticSearchUsage increments daily semantic search usage counter
func (v *Verifier) IncrementSemanticSearchUsage(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) error {
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	user.SemanticSearchDailyUsage++
	return uow.UserRepository().Update(ctx, user)
}
