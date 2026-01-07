package model

import (
	"time"

	"github.com/google/uuid"
)

// ChatMessageReference represents a persistent link between a chat message and a note
type ChatMessageReference struct {
	Id            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChatMessageId uuid.UUID `gorm:"type:uuid;index;not null"`
	NoteId        uuid.UUID `gorm:"type:uuid;index;not null"`
	CreatedAt     time.Time
	UpdatedAt     *time.Time

	// Relationships
	ChatMessage ChatMessage `gorm:"foreignKey:ChatMessageId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Note        Note        `gorm:"foreignKey:NoteId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
