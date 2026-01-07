package session

import (
	"context"
	"fmt"
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// Manager handles session operations
type Manager struct {
	sessionRepo *memory.SessionRepository
}

// NewManager creates a new session manager
func NewManager(sessionRepo *memory.SessionRepository) *Manager {
	return &Manager{sessionRepo: sessionRepo}
}

// LoadOrCreate retrieves or creates an in-memory session
func (m *Manager) LoadOrCreate(userId uuid.UUID, sessionId uuid.UUID) *store.Session {
	sessionID := sessionId.String()
	session, found := m.sessionRepo.Get(sessionID)
	if !found {
		session = &store.Session{
			ID:     sessionID,
			UserID: userId.String(),
			State:  store.StateBrowsing,
		}
	}
	return session
}

// Save persists session state
func (m *Manager) Save(session *store.Session) {
	m.sessionRepo.Save(session)
}

// VerifyChatSession validates session ownership
func (m *Manager) VerifyChatSession(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID, sessionId uuid.UUID) (*entity.ChatSession, error) {
	session, err := uow.ChatSessionRepository().FindOne(ctx,
		specification.ByID{ID: sessionId},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found or access denied")
	}
	return session, nil
}

// UpdateTitle updates session title
func (m *Manager) UpdateTitle(ctx context.Context, uow unitofwork.UnitOfWork, session *entity.ChatSession, title string, now time.Time) error {
	session.Title = title
	session.UpdatedAt = &now
	return uow.ChatSessionRepository().Update(ctx, session)
}
