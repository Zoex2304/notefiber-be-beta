package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Note struct {
	Id         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title      string         `gorm:"type:varchar(255);not null"`
	Content    string         `gorm:"type:text"`
	NotebookId uuid.UUID      `gorm:"type:uuid;not null;index"`
	UserId     uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (Note) TableName() string {
	return "notes"
}
