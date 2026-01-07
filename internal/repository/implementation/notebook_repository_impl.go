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

type NotebookRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.NotebookMapper
}

func NewNotebookRepository(db *gorm.DB) contract.NotebookRepository {
	return &NotebookRepositoryImpl{
		db:     db,
		mapper: mapper.NewNotebookMapper(),
	}
}

func (r *NotebookRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *NotebookRepositoryImpl) Create(ctx context.Context, notebook *entity.Notebook) error {
	m := r.mapper.ToModel(notebook)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*notebook = *r.mapper.ToEntity(m)
	return nil
}

func (r *NotebookRepositoryImpl) Update(ctx context.Context, notebook *entity.Notebook) error {
	m := r.mapper.ToModel(notebook)
	// Use Select("*") or explicit fields if you want to update zero values too,
	// but generally GORM Updates ignores zero values. Save() updates all fields including zero values if primary key exists.
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*notebook = *r.mapper.ToEntity(m)
	return nil
}

func (r *NotebookRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Notebook{}, id).Error
}

func (r *NotebookRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.Notebook{}).Error
}

func (r *NotebookRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Notebook, error) {
	var m model.Notebook
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}

func (r *NotebookRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Notebook, error) {
	var models []*model.Notebook
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	return r.mapper.ToEntities(models), nil
}

func (r *NotebookRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.Notebook{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
