package mapper

import (
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"

	"gorm.io/gorm"
)

type NoteMapper struct{}

func NewNoteMapper() *NoteMapper {
	return &NoteMapper{}
}

func (m *NoteMapper) ToEntity(n *model.Note) *entity.Note {
	if n == nil {
		return nil
	}

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

	return &entity.Note{
		Id:         n.Id,
		Title:      n.Title,
		Content:    n.Content,
		NotebookId: n.NotebookId,
		UserId:     n.UserId,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  updatedAt,
		DeletedAt:  deletedAt,
		IsDeleted:  n.DeletedAt.Valid,
	}
}

func (m *NoteMapper) ToModel(n *entity.Note) *model.Note {
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

	return &model.Note{
		Id:         n.Id,
		Title:      n.Title,
		Content:    n.Content,
		NotebookId: n.NotebookId,
		UserId:     n.UserId,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  updatedAt,
		DeletedAt:  deletedAt,
	}
}

func (m *NoteMapper) ToEntities(notes []*model.Note) []*entity.Note {
	entities := make([]*entity.Note, len(notes))
	for i, n := range notes {
		entities[i] = m.ToEntity(n)
	}
	return entities
}

func (m *NoteMapper) ToModels(notes []*entity.Note) []*model.Note {
	models := make([]*model.Note, len(notes))
	for i, n := range notes {
		models[i] = m.ToModel(n)
	}
	return models
}
