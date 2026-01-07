// FILE: internal/repository/implementation/cancellation_repository_impl.go
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

type cancellationRepositoryImpl struct {
	db *gorm.DB
}

// NewCancellationRepository creates a new cancellation repository
func NewCancellationRepository(db *gorm.DB) contract.CancellationRepository {
	return &cancellationRepositoryImpl{db: db}
}

func (r *cancellationRepositoryImpl) Create(ctx context.Context, cancellation *entity.Cancellation) error {
	modelCancellation := &model.Cancellation{
		ID:             cancellation.ID,
		SubscriptionID: cancellation.SubscriptionID,
		UserID:         cancellation.UserID,
		Reason:         cancellation.Reason,
		Status:         cancellation.Status,
		AdminNotes:     cancellation.AdminNotes,
		EffectiveDate:  cancellation.EffectiveDate,
		ProcessedAt:    cancellation.ProcessedAt,
	}
	return r.db.WithContext(ctx).Create(modelCancellation).Error
}

func (r *cancellationRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Cancellation, error) {
	var modelCancellation model.Cancellation
	query := r.db.WithContext(ctx)

	for _, spec := range specs {
		query = spec.Apply(query)
	}

	if err := query.First(&modelCancellation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.mapToEntity(&modelCancellation), nil
}

func (r *cancellationRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Cancellation, error) {
	var modelCancellations []*model.Cancellation
	query := r.db.WithContext(ctx)

	for _, spec := range specs {
		query = spec.Apply(query)
	}

	if err := query.Find(&modelCancellations).Error; err != nil {
		return nil, err
	}

	var cancellations []*entity.Cancellation
	for _, mc := range modelCancellations {
		cancellations = append(cancellations, r.mapToEntity(mc))
	}

	return cancellations, nil
}

// FindAllWithDetails returns cancellations with preloaded User and Subscription relations
func (r *cancellationRepositoryImpl) FindAllWithDetails(ctx context.Context, specs ...specification.Specification) ([]*entity.Cancellation, error) {
	var modelCancellations []*model.Cancellation
	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("Subscription")

	for _, spec := range specs {
		query = spec.Apply(query)
	}

	if err := query.Find(&modelCancellations).Error; err != nil {
		return nil, err
	}

	var cancellations []*entity.Cancellation
	for _, mc := range modelCancellations {
		cancellation := r.mapToEntity(mc)
		// Map User
		cancellation.User = entity.User{
			Id:       mc.User.Id,
			Email:    mc.User.Email,
			FullName: mc.User.FullName,
		}
		// Map Subscription
		cancellation.Subscription = entity.UserSubscription{
			Id:               mc.Subscription.Id,
			PlanId:           mc.Subscription.PlanId,
			CurrentPeriodEnd: mc.Subscription.CurrentPeriodEnd,
		}
		cancellations = append(cancellations, cancellation)
	}

	return cancellations, nil
}

func (r *cancellationRepositoryImpl) Update(ctx context.Context, cancellation *entity.Cancellation) error {
	return r.db.WithContext(ctx).Model(&model.Cancellation{}).
		Where("id = ?", cancellation.ID).
		Updates(map[string]interface{}{
			"reason":         cancellation.Reason,
			"status":         cancellation.Status,
			"admin_notes":    cancellation.AdminNotes,
			"effective_date": cancellation.EffectiveDate,
			"processed_at":   cancellation.ProcessedAt,
		}).Error
}

func (r *cancellationRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Cancellation{}, id).Error
}

func (r *cancellationRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.Cancellation{}).Error
}

// mapToEntity converts model.Cancellation to entity.Cancellation
func (r *cancellationRepositoryImpl) mapToEntity(mc *model.Cancellation) *entity.Cancellation {
	return &entity.Cancellation{
		ID:             mc.ID,
		SubscriptionID: mc.SubscriptionID,
		UserID:         mc.UserID,
		Reason:         mc.Reason,
		Status:         mc.Status,
		AdminNotes:     mc.AdminNotes,
		EffectiveDate:  mc.EffectiveDate,
		ProcessedAt:    mc.ProcessedAt,
		CreatedAt:      mc.CreatedAt,
		UpdatedAt:      mc.UpdatedAt,
	}
}
