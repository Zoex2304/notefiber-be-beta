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

type BillingRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.BillingMapper
}

func NewBillingRepository(db *gorm.DB) contract.BillingRepository {
	return &BillingRepositoryImpl{
		db:     db,
		mapper: mapper.NewBillingMapper(),
	}
}

func (r *BillingRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *BillingRepositoryImpl) Create(ctx context.Context, billing *entity.BillingAddress) error {
	m := r.mapper.ToModel(billing)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*billing = *r.mapper.ToEntity(m)
	return nil
}

func (r *BillingRepositoryImpl) Update(ctx context.Context, billing *entity.BillingAddress) error {
	m := r.mapper.ToModel(billing)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*billing = *r.mapper.ToEntity(m)
	return nil
}

func (r *BillingRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.BillingAddress{}, id).Error
}

func (r *BillingRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.BillingAddress{}).Error
}

func (r *BillingRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.BillingAddress, error) {
	var m model.BillingAddress
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}

func (r *BillingRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.BillingAddress, error) {
	var models []*model.BillingAddress
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	entities := make([]*entity.BillingAddress, len(models))
	for i, m := range models {
		entities[i] = r.mapper.ToEntity(m)
	}
	return entities, nil
}

func (r *BillingRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.BillingAddress{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
