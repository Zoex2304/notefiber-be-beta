package mapper

import (
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"

	"gorm.io/gorm"
)

type NotebookMapper struct{}

func NewNotebookMapper() *NotebookMapper {
	return &NotebookMapper{}
}

func (m *NotebookMapper) ToEntity(n *model.Notebook) *entity.Notebook {
	if n == nil {
		return nil
	}
	// Note: gorm.DeletedAt is struct { Time time.Time; Valid bool }
	var deletedAt *time.Time
	if n.DeletedAt.Valid {
		t := n.DeletedAt.Time
		deletedAt = &t
	}

	var updatedAt *time.Time
	if !n.UpdatedAt.IsZero() {
		t := n.UpdatedAt
		updatedAt = &t
	}

	return &entity.Notebook{
		Id:        n.Id,
		Name:      n.Name,
		ParentId:  n.ParentId,
		UserId:    n.UserId,
		CreatedAt: n.CreatedAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
		IsDeleted: n.DeletedAt.Valid,
	}
}

func (m *NotebookMapper) ToModel(n *entity.Notebook) *model.Notebook {
	if n == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if n.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *n.DeletedAt, Valid: true}
	} else if n.IsDeleted {
		deletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}

	var updatedAt time.Time
	if n.UpdatedAt != nil {
		updatedAt = *n.UpdatedAt
	}

	return &model.Notebook{
		Id:        n.Id,
		Name:      n.Name,
		ParentId:  n.ParentId,
		UserId:    n.UserId,
		CreatedAt: n.CreatedAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}
}

func (m *NotebookMapper) ToEntities(notebooks []*model.Notebook) []*entity.Notebook {
	entities := make([]*entity.Notebook, len(notebooks))
	for i, n := range notebooks {
		entities[i] = m.ToEntity(n)
	}
	return entities
}

func (m *NotebookMapper) ToModels(notebooks []*entity.Notebook) []*model.Notebook {
	models := make([]*model.Notebook, len(notebooks))
	for i, n := range notebooks {
		models[i] = m.ToModel(n)
	}
	return models
}
