package entity

import (
	"time"

	"github.com/google/uuid"
)

type ChatMessage struct {
	Id            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Chat          string
	Role          string
	ChatSessionId uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt     time.Time
	UpdatedAt     *time.Time
	DeletedAt     *time.Time
	IsDeleted     bool
}
