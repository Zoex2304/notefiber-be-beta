// FILE: internal/repository/implementation/feature_repository_impl.go
// Implementation of FeatureRepository
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

type FeatureRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.FeatureMapper
}

func NewFeatureRepository(db *gorm.DB) contract.FeatureRepository {
	return &FeatureRepositoryImpl{
		db:     db,
		mapper: mapper.NewFeatureMapper(),
	}
}

func (r *FeatureRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *FeatureRepositoryImpl) Create(ctx context.Context, feature *entity.Feature) error {
	m := r.mapper.ToModel(feature)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*feature = *r.mapper.ToEntity(m)
	return nil
}

func (r *FeatureRepositoryImpl) Update(ctx context.Context, feature *entity.Feature) error {
	m := r.mapper.ToModel(feature)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*feature = *r.mapper.ToEntity(m)
	return nil
}

func (r *FeatureRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Feature{}, id).Error
}

func (r *FeatureRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Feature, error) {
	var m model.Feature
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}

func (r *FeatureRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Feature, error) {
	var models []*model.Feature
	query := r.applySpecifications(r.db.WithContext(ctx).Order("sort_order ASC"), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	return r.mapper.ToEntities(models), nil
}

func (r *FeatureRepositoryImpl) FindByKey(ctx context.Context, key string) (*entity.Feature, error) {
	var m model.Feature
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}
