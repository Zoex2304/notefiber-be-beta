package specification

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ByChatSessionID struct {
	ChatSessionID uuid.UUID
}

func (s ByChatSessionID) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("chat_session_id = ?", s.ChatSessionID)
}
