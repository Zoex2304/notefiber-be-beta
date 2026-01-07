package implementation

import (
	"context"
	"errors"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/mapper"
	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.SubscriptionMapper
}

func NewSubscriptionRepository(db *gorm.DB) contract.SubscriptionRepository {
	return &SubscriptionRepositoryImpl{
		db:     db,
		mapper: mapper.NewSubscriptionMapper(),
	}
}

func (r *SubscriptionRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

// Plan Implementation

func (r *SubscriptionRepositoryImpl) CreatePlan(ctx context.Context, plan *entity.SubscriptionPlan) error {
	m := r.mapper.PlanToModel(plan)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*plan = *r.mapper.PlanToEntity(m)
	return nil
}

func (r *SubscriptionRepositoryImpl) UpdatePlan(ctx context.Context, plan *entity.SubscriptionPlan) error {
	m := r.mapper.PlanToModel(plan)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*plan = *r.mapper.PlanToEntity(m)
	return nil
}

func (r *SubscriptionRepositoryImpl) DeletePlan(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.SubscriptionPlan{}, id).Error
}

func (r *SubscriptionRepositoryImpl) FindOnePlan(ctx context.Context, specs ...specification.Specification) (*entity.SubscriptionPlan, error) {
	var m model.SubscriptionPlan
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	// Preload Features
	query = query.Preload("Features")
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.PlanToEntity(&m), nil
}

func (r *SubscriptionRepositoryImpl) FindAllPlans(ctx context.Context, specs ...specification.Specification) ([]*entity.SubscriptionPlan, error) {
	var models []*model.SubscriptionPlan
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	// Preload Features
	query = query.Preload("Features")
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	entities := make([]*entity.SubscriptionPlan, len(models))
	for i, m := range models {
		entities[i] = r.mapper.PlanToEntity(m)
	}
	return entities, nil
}

// Subscription Implementation

func (r *SubscriptionRepositoryImpl) CreateSubscription(ctx context.Context, subscription *entity.UserSubscription) error {
	m := r.mapper.UserSubscriptionToModel(subscription)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*subscription = *r.mapper.UserSubscriptionToEntity(m)
	return nil
}

func (r *SubscriptionRepositoryImpl) UpdateSubscription(ctx context.Context, subscription *entity.UserSubscription) error {
	m := r.mapper.UserSubscriptionToModel(subscription)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*subscription = *r.mapper.UserSubscriptionToEntity(m)
	return nil
}

func (r *SubscriptionRepositoryImpl) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.UserSubscription{}, id).Error
}

func (r *SubscriptionRepositoryImpl) DeleteAllSubscriptionsByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.UserSubscription{}).Error
}

func (r *SubscriptionRepositoryImpl) FindOneSubscription(ctx context.Context, specs ...specification.Specification) (*entity.UserSubscription, error) {
	var m model.UserSubscription
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.UserSubscriptionToEntity(&m), nil
}

func (r *SubscriptionRepositoryImpl) FindAllSubscriptions(ctx context.Context, specs ...specification.Specification) ([]*entity.UserSubscription, error) {
	var models []*model.UserSubscription
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	entities := make([]*entity.UserSubscription, len(models))
	for i, m := range models {
		entities[i] = r.mapper.UserSubscriptionToEntity(m)
	}
	return entities, nil
}

// Dashboard / Admin Stats implementation

func (r *SubscriptionRepositoryImpl) GetTotalRevenue(ctx context.Context) (float64, error) {
	var total float64
	// Sum of Plan.Price where subscription is PAID
	// Assuming "payment_status" = 'success' in user_subscriptions (as per entity constant)
	err := r.db.WithContext(ctx).Table("user_subscriptions").
		Joins("JOIN subscription_plans ON user_subscriptions.plan_id = subscription_plans.id").
		Where("user_subscriptions.payment_status = ?", "success").
		Select("COALESCE(SUM(subscription_plans.price), 0)").
		Scan(&total).Error
	return total, err
}

func (r *SubscriptionRepositoryImpl) CountActiveSubscribers(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserSubscription{}).
		Where("status = ?", "active").
		Count(&count).Error
	return int(count), err
}

func (r *SubscriptionRepositoryImpl) GetTransactions(ctx context.Context, status string, limit, offset int) ([]*entity.SubscriptionTransaction, error) {
	var results []*entity.SubscriptionTransaction

	// Join UserSubscription, User, Plan
	query := r.db.WithContext(ctx).Table("user_subscriptions").
		Select(`
			user_subscriptions.id,
			user_subscriptions.user_id,
			users.email as user_email,
			subscription_plans.name as plan_name,
			subscription_plans.price as amount,
			user_subscriptions.status,
			user_subscriptions.payment_status,
			user_subscriptions.created_at,
			user_subscriptions.midtrans_transaction_id as midtrans_order_id
		`).
		Joins("JOIN users ON user_subscriptions.user_id = users.id").
		Joins("JOIN subscription_plans ON user_subscriptions.plan_id = subscription_plans.id")

	if status != "" {
		query = query.Where("user_subscriptions.payment_status = ?", status)
	}

	err := query.Order("user_subscriptions.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// Feature Management Implementation

func (r *SubscriptionRepositoryImpl) AddFeatureToPlan(ctx context.Context, planId uuid.UUID, featureId uuid.UUID) error {
	// Use GORM Association to add many-to-many link
	plan := &model.SubscriptionPlan{Id: planId}
	feature := &model.Feature{Id: featureId}
	return r.db.WithContext(ctx).Model(plan).Association("Features").Append(feature)
}

func (r *SubscriptionRepositoryImpl) RemoveFeatureFromPlan(ctx context.Context, planId uuid.UUID, featureId uuid.UUID) error {
	// Use GORM Association to remove many-to-many link
	plan := &model.SubscriptionPlan{Id: planId}
	feature := &model.Feature{Id: featureId}
	return r.db.WithContext(ctx).Model(plan).Association("Features").Delete(feature)
}
