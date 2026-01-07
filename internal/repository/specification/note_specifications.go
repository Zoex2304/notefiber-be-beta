package specification

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ByNotebookID struct {
	NotebookID uuid.UUID
}

func (s ByNotebookID) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("notebook_id = ?", s.NotebookID)
}

type ByNotebookIDs struct {
	NotebookIDs []uuid.UUID
}

func (s ByNotebookIDs) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("notebook_id IN ?", s.NotebookIDs)
}

type NoteOwnedByUser struct {
	UserID uuid.UUID
}

func (s NoteOwnedByUser) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("notes.user_id = ?", s.UserID)
}

type ByTitle struct {
	Title string
}

func (s ByTitle) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("title = ?", s.Title)
}
