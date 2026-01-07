package mapper

import (
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"

	"gorm.io/gorm"
)

type ChatMapper struct{}

func NewChatMapper() *ChatMapper {
	return &ChatMapper{}
}

// Session Mappers

func (m *ChatMapper) ChatSessionToEntity(s *model.ChatSession) *entity.ChatSession {
	if s == nil {
		return nil
	}

	var deletedAt *time.Time
	if s.DeletedAt.Valid {
		t := s.DeletedAt.Time
		deletedAt = &t
	}

	var updatedAt *time.Time
	if !s.UpdatedAt.IsZero() {
		t := s.UpdatedAt
		updatedAt = &t
	}

	return &entity.ChatSession{
		Id:        s.Id,
		UserId:    s.UserId,
		Title:     s.Title,
		CreatedAt: s.CreatedAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
		IsDeleted: s.DeletedAt.Valid,
	}
}

func (m *ChatMapper) ChatSessionToModel(s *entity.ChatSession) *model.ChatSession {
	if s == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if s.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *s.DeletedAt, Valid: true}
	} else if s.IsDeleted {
		deletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}

	var updatedAt time.Time
	if s.UpdatedAt != nil {
		updatedAt = *s.UpdatedAt
	}

	return &model.ChatSession{
		Id:        s.Id,
		UserId:    s.UserId,
		Title:     s.Title,
		CreatedAt: s.CreatedAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}
}

// Message Mappers

func (m *ChatMapper) ChatMessageToEntity(msg *model.ChatMessage) *entity.ChatMessage {
	if msg == nil {
		return nil
	}

	var deletedAt *time.Time
	if msg.DeletedAt.Valid {
		t := msg.DeletedAt.Time
		deletedAt = &t
	}

	var updatedAt *time.Time
	if !msg.UpdatedAt.IsZero() {
		t := msg.UpdatedAt
		updatedAt = &t
	}

	return &entity.ChatMessage{
		Id:            msg.Id,
		Chat:          msg.Chat,
		Role:          msg.Role,
		ChatSessionId: msg.ChatSessionId,
		CreatedAt:     msg.CreatedAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
		IsDeleted:     msg.DeletedAt.Valid,
	}
}

func (m *ChatMapper) ChatMessageToModel(msg *entity.ChatMessage) *model.ChatMessage {
	if msg == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if msg.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *msg.DeletedAt, Valid: true}
	} else if msg.IsDeleted {
		deletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}

	var updatedAt time.Time
	if msg.UpdatedAt != nil {
		updatedAt = *msg.UpdatedAt
	}

	return &model.ChatMessage{
		Id:            msg.Id,
		Chat:          msg.Chat,
		Role:          msg.Role,
		ChatSessionId: msg.ChatSessionId,
		CreatedAt:     msg.CreatedAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
	}
}

// Raw Message Mappers

func (m *ChatMapper) ChatMessageRawToEntity(msg *model.ChatMessageRaw) *entity.ChatMessageRaw {
	if msg == nil {
		return nil
	}

	var deletedAt *time.Time
	if msg.DeletedAt.Valid {
		t := msg.DeletedAt.Time
		deletedAt = &t
	}

	var updatedAt *time.Time
	if !msg.UpdatedAt.IsZero() {
		t := msg.UpdatedAt
		updatedAt = &t
	}

	return &entity.ChatMessageRaw{
		Id:            msg.Id,
		Chat:          msg.Chat,
		Role:          msg.Role,
		ChatSessionId: msg.ChatSessionId,
		CreatedAt:     msg.CreatedAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
		IsDeleted:     msg.DeletedAt.Valid,
	}
}

func (m *ChatMapper) ChatMessageRawToModel(msg *entity.ChatMessageRaw) *model.ChatMessageRaw {
	if msg == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if msg.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *msg.DeletedAt, Valid: true}
	} else if msg.IsDeleted {
		deletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}

	var updatedAt time.Time
	if msg.UpdatedAt != nil {
		updatedAt = *msg.UpdatedAt
	}

	return &model.ChatMessageRaw{
		Id:            msg.Id,
		Chat:          msg.Chat,
		Role:          msg.Role,
		ChatSessionId: msg.ChatSessionId,
		CreatedAt:     msg.CreatedAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
	}
}

// Batch methods (Optional, add if needed)
