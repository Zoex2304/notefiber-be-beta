package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatSession struct {
	Id        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId    uuid.UUID      `gorm:"type:uuid;not null;index"` // User ownership for data isolation
	Title     string         `gorm:"type:text;not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (ChatSession) TableName() string {
	return "chat_sessions"
}
