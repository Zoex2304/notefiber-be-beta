package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notebook struct {
	Id        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(255);not null"`
	ParentId  *uuid.UUID     `gorm:"type:uuid;index"`
	UserId    uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Notebook) TableName() string {
	return "notebooks"
}
