package message

import (
	"context"
	"time"

	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

// Factory handles chat message creation and persistence
type Factory struct{}

// NewFactory creates a new message factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateUserMessage creates a chat message from user input
func (f *Factory) CreateUserMessage(request *dto.SendChatRequest, now time.Time) entity.ChatMessage {
	return entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          request.Chat,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now,
	}
}

// CreateModelMessage creates a chat message from model response
func (f *Factory) CreateModelMessage(sessionId uuid.UUID, content string, now time.Time) entity.ChatMessage {
	return entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          content,
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: sessionId,
		CreatedAt:     now.Add(1 * time.Second),
	}
}

// SaveUserMessage persists user message to both repositories
func (f *Factory) SaveUserMessage(ctx context.Context, uow unitofwork.UnitOfWork, message entity.ChatMessage) error {
	if err := uow.ChatMessageRepository().Create(ctx, &message); err != nil {
		return err
	}

	raw := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          message.Chat,
		Role:          message.Role,
		ChatSessionId: message.ChatSessionId,
		CreatedAt:     message.CreatedAt,
	}
	return uow.ChatMessageRawRepository().Create(ctx, &raw)
}

// SaveModelMessage persists model message to both repositories and saves citations
func (f *Factory) SaveModelMessage(ctx context.Context, uow unitofwork.UnitOfWork, message entity.ChatMessage, citations []dto.CitationDTO) error {
	if err := uow.ChatMessageRepository().Create(ctx, &message); err != nil {
		return err
	}

	raw := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          message.Chat,
		Role:          message.Role,
		ChatSessionId: message.ChatSessionId,
		CreatedAt:     message.CreatedAt,
	}
	if err := uow.ChatMessageRawRepository().Create(ctx, &raw); err != nil {
		return err
	}

	// Save citations if any
	if len(citations) > 0 {
		var chatCitations []*entity.ChatCitation
		for _, c := range citations {
			chatCitations = append(chatCitations, &entity.ChatCitation{
				Id:            uuid.New(),
				ChatMessageId: message.Id,
				NoteId:        c.NoteId,
			})
		}
		if err := uow.ChatMessageRepository().CreateCitations(ctx, chatCitations); err != nil {
			return err
		}
	}

	return nil
}
