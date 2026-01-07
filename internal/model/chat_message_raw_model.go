package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatMessageRaw struct {
	Id            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Chat          string         `gorm:"type:text;not null"`
	Role          string         `gorm:"type:varchar(50);not null"`
	ChatSessionId uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (ChatMessageRaw) TableName() string {
	return "chat_messages_raw"
}
