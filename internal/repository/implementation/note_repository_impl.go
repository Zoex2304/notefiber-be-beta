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

type NoteRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.NoteMapper
}

func NewNoteRepository(db *gorm.DB) contract.NoteRepository {
	return &NoteRepositoryImpl{
		db:     db,
		mapper: mapper.NewNoteMapper(),
	}
}

func (r *NoteRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *NoteRepositoryImpl) Create(ctx context.Context, note *entity.Note) error {
	m := r.mapper.ToModel(note)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*note = *r.mapper.ToEntity(m)
	return nil
}

func (r *NoteRepositoryImpl) Update(ctx context.Context, note *entity.Note) error {
	m := r.mapper.ToModel(note)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*note = *r.mapper.ToEntity(m)
	return nil
}

func (r *NoteRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Note{}, id).Error
}

func (r *NoteRepositoryImpl) DeleteAllByUserIdUnscoped(ctx context.Context, userId uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userId).Delete(&model.Note{}).Error
}

func (r *NoteRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Note, error) {
	var m model.Note
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}

func (r *NoteRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Note, error) {
	var models []*model.Note
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	return r.mapper.ToEntities(models), nil
}

func (r *NoteRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.Note{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
