package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type SubscriptionRepository interface {
	// Plans
	CreatePlan(ctx context.Context, plan *entity.SubscriptionPlan) error
	UpdatePlan(ctx context.Context, plan *entity.SubscriptionPlan) error
	DeletePlan(ctx context.Context, id uuid.UUID) error
	FindOnePlan(ctx context.Context, specs ...specification.Specification) (*entity.SubscriptionPlan, error)
	FindAllPlans(ctx context.Context, specs ...specification.Specification) ([]*entity.SubscriptionPlan, error)

	// User Subscription Implementation
	CreateSubscription(ctx context.Context, subscription *entity.UserSubscription) error
	UpdateSubscription(ctx context.Context, subscription *entity.UserSubscription) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error
	DeleteAllSubscriptionsByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error // Hard delete all
	FindOneSubscription(ctx context.Context, specs ...specification.Specification) (*entity.UserSubscription, error)
	FindAllSubscriptions(ctx context.Context, specs ...specification.Specification) ([]*entity.UserSubscription, error)

	// Dashboard / Admin Stats
	GetTotalRevenue(ctx context.Context) (float64, error)
	CountActiveSubscribers(ctx context.Context) (int, error)

	GetTransactions(ctx context.Context, status string, limit, offset int) ([]*entity.SubscriptionTransaction, error)

	// Feature Management
	AddFeatureToPlan(ctx context.Context, planId uuid.UUID, featureId uuid.UUID) error
	RemoveFeatureFromPlan(ctx context.Context, planId uuid.UUID, featureId uuid.UUID) error
}
