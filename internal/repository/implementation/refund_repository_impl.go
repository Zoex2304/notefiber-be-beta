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

type refundRepositoryImpl struct {
	db *gorm.DB
}

func NewRefundRepository(db *gorm.DB) contract.RefundRepository {
	return &refundRepositoryImpl{db: db}
}

func (r *refundRepositoryImpl) Create(ctx context.Context, refund *entity.Refund) error {
	modelRefund := &model.Refund{
		ID:             refund.ID,
		SubscriptionID: refund.SubscriptionID,
		UserID:         refund.UserID,
		Amount:         refund.Amount,
		Reason:         refund.Reason,
		Status:         refund.Status,
		AdminNotes:     refund.AdminNotes,
		ProcessedAt:    refund.ProcessedAt,
	}
	return r.db.WithContext(ctx).Create(modelRefund).Error
}

func (r *refundRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Refund, error) {
	var modelRefund model.Refund
	query := r.db.WithContext(ctx)

	for _, spec := range specs {
		query = spec.Apply(query)
	}

	if err := query.First(&modelRefund).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.mapToEntity(&modelRefund), nil
}

func (r *refundRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Refund, error) {
	var modelRefunds []*model.Refund
	query := r.db.WithContext(ctx)

	for _, spec := range specs {
		query = spec.Apply(query)
	}

	if err := query.Find(&modelRefunds).Error; err != nil {
		return nil, err
	}

	var refunds []*entity.Refund
	for _, mr := range modelRefunds {
		refunds = append(refunds, r.mapToEntity(mr))
	}

	return refunds, nil
}

// FindAllWithDetails returns refunds with preloaded User and Subscription relations
func (r *refundRepositoryImpl) FindAllWithDetails(ctx context.Context, specs ...specification.Specification) ([]*entity.Refund, error) {
	var modelRefunds []*model.Refund
	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("Subscription")

	for _, spec := range specs {
		query = spec.Apply(query)
	}

	if err := query.Find(&modelRefunds).Error; err != nil {
		return nil, err
	}

	var refunds []*entity.Refund
	for _, mr := range modelRefunds {
		refund := r.mapToEntity(mr)
		// Map User
		refund.User = entity.User{
			Id:       mr.User.Id,
			Email:    mr.User.Email,
			FullName: mr.User.FullName,
		}
		// Map Subscription
		refund.Subscription = entity.UserSubscription{
			Id:                 mr.Subscription.Id,
			PlanId:             mr.Subscription.PlanId,
			CurrentPeriodStart: mr.Subscription.CurrentPeriodStart,
		}
		refunds = append(refunds, refund)
	}

	return refunds, nil
}

func (r *refundRepositoryImpl) Update(ctx context.Context, refund *entity.Refund) error {
	return r.db.WithContext(ctx).Model(&model.Refund{}).
		Where("id = ?", refund.ID).
		Updates(map[string]interface{}{
			"amount":       refund.Amount,
			"reason":       refund.Reason,
			"status":       refund.Status,
			"admin_notes":  refund.AdminNotes,
			"processed_at": refund.ProcessedAt,
		}).Error
}

func (r *refundRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Refund{}, id).Error
}

func (r *refundRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.Refund{}).Error
}

// mapToEntity converts model.Refund to entity.Refund
func (r *refundRepositoryImpl) mapToEntity(mr *model.Refund) *entity.Refund {
	return &entity.Refund{
		ID:             mr.ID,
		SubscriptionID: mr.SubscriptionID,
		UserID:         mr.UserID,
		Amount:         mr.Amount,
		Reason:         mr.Reason,
		Status:         mr.Status,
		AdminNotes:     mr.AdminNotes,
		ProcessedAt:    mr.ProcessedAt,
		CreatedAt:      mr.CreatedAt,
		UpdatedAt:      mr.UpdatedAt,
	}
}
