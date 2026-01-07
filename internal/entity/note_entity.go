package entity

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	Id         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title      string
	Content    string
	NotebookId uuid.UUID `gorm:"type:uuid;index"`
	UserId     uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt  time.Time
	UpdatedAt  *time.Time
	DeletedAt  *time.Time
	IsDeleted  bool
}
