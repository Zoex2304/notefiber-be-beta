package entity

import (
	"time"

	"github.com/google/uuid"
)

type ChatCitation struct {
	Id            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChatMessageId uuid.UUID `gorm:"type:uuid;not null;index"`
	NoteId        uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`

	// Relationships
	ChatMessage *ChatMessage `gorm:"foreignKey:ChatMessageId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Note        *Note        `gorm:"foreignKey:NoteId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
