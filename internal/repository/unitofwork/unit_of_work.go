package unitofwork

import (
	"context"

	"ai-notetaking-be/internal/repository/contract"
)

type UnitOfWork interface {
	Begin(ctx context.Context) error
	Commit() error
	Rollback() error

	UserRepository() contract.UserRepository
	NotebookRepository() contract.NotebookRepository
	NoteRepository() contract.NoteRepository
	NoteEmbeddingRepository() contract.NoteEmbeddingRepository

	ChatSessionRepository() contract.ChatSessionRepository
	ChatMessageRepository() contract.ChatMessageRepository
	ChatMessageRawRepository() contract.ChatMessageRawRepository
	ChatMessageReferenceRepository() contract.ChatMessageReferenceRepository
	ChatCitationRepository() contract.ChatCitationRepository
	SubscriptionRepository() contract.SubscriptionRepository // Restored
	FeatureRepository() contract.FeatureRepository
	BillingRepository() contract.BillingRepository
	RefundRepository() contract.RefundRepository
	CancellationRepository() contract.CancellationRepository
	AiConfigRepository() contract.IAiConfigRepository
}
