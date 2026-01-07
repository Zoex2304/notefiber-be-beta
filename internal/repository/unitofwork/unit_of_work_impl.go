package unitofwork

import (
	"context"
	"fmt"

	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/implementation"

	"gorm.io/gorm"
)

type UnitOfWorkImpl struct {
	db *gorm.DB
	tx *gorm.DB // The active transaction (or just db if no tx) - actually we should keep track if we are in tx
}

func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	return &UnitOfWorkImpl{
		db: db,
	}
}

func (u *UnitOfWorkImpl) getDB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}

func (u *UnitOfWorkImpl) Begin(ctx context.Context) error {
	if u.tx != nil {
		return fmt.Errorf("transaction already started")
	}
	u.tx = u.db.WithContext(ctx).Begin()
	return u.tx.Error
}

func (u *UnitOfWorkImpl) Commit() error {
	if u.tx == nil {
		return fmt.Errorf("no transaction to commit")
	}
	err := u.tx.Commit().Error
	u.tx = nil
	return err
}

func (u *UnitOfWorkImpl) Rollback() error {
	if u.tx == nil {
		return fmt.Errorf("no transaction to rollback")
	}
	err := u.tx.Rollback().Error
	u.tx = nil
	return err
}

// Repository Accessors

func (u *UnitOfWorkImpl) UserRepository() contract.UserRepository {
	return implementation.NewUserRepository(u.getDB())
}

func (u *UnitOfWorkImpl) NotebookRepository() contract.NotebookRepository {
	return implementation.NewNotebookRepository(u.getDB())
}

func (u *UnitOfWorkImpl) NoteRepository() contract.NoteRepository {
	return implementation.NewNoteRepository(u.getDB())
}

func (u *UnitOfWorkImpl) NoteEmbeddingRepository() contract.NoteEmbeddingRepository {
	return implementation.NewNoteEmbeddingRepository(u.getDB())
}

func (u *UnitOfWorkImpl) ChatSessionRepository() contract.ChatSessionRepository {
	return implementation.NewChatSessionRepository(u.getDB())
}

func (u *UnitOfWorkImpl) ChatMessageRepository() contract.ChatMessageRepository {
	return implementation.NewChatMessageRepository(u.getDB())
}

func (u *UnitOfWorkImpl) ChatMessageRawRepository() contract.ChatMessageRawRepository {
	return implementation.NewChatMessageRawRepository(u.getDB())
}

func (u *UnitOfWorkImpl) ChatMessageReferenceRepository() contract.ChatMessageReferenceRepository {
	return implementation.NewChatMessageReferenceRepository(u.getDB())
}

func (u *UnitOfWorkImpl) ChatCitationRepository() contract.ChatCitationRepository {
	return implementation.NewChatCitationRepository(u.getDB())
}

func (u *UnitOfWorkImpl) SubscriptionRepository() contract.SubscriptionRepository {
	return implementation.NewSubscriptionRepository(u.getDB())
}

func (u *UnitOfWorkImpl) FeatureRepository() contract.FeatureRepository {
	return implementation.NewFeatureRepository(u.getDB())
}

func (u *UnitOfWorkImpl) BillingRepository() contract.BillingRepository {
	return implementation.NewBillingRepository(u.getDB())
}

func (u *UnitOfWorkImpl) RefundRepository() contract.RefundRepository {
	return implementation.NewRefundRepository(u.getDB())
}

func (u *UnitOfWorkImpl) CancellationRepository() contract.CancellationRepository {
	return implementation.NewCancellationRepository(u.getDB())
}

func (u *UnitOfWorkImpl) AiConfigRepository() contract.IAiConfigRepository {
	return implementation.NewAiConfigRepository(u.getDB())
}
