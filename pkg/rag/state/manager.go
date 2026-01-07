package state

import (
	"log"

	"ai-notetaking-be/pkg/store"
)

// Manager handles session state transitions
type Manager struct {
	logger *log.Logger
}

// NewManager creates a new state manager
func NewManager(logger *log.Logger) *Manager {
	return &Manager{logger: logger}
}

// TransitionToFocused sets session focus to a single document
func (m *Manager) TransitionToFocused(session *store.Session, document store.Document) {
	session.FocusedNote = &document
	session.State = store.StateFocused
	m.logger.Printf("[STATE] Transitioned to FOCUSED: %s", document.Title)
}

// TransitionToBrowsing sets session to browsing mode with multiple candidates
func (m *Manager) TransitionToBrowsing(session *store.Session, candidates []store.Document) {
	session.Candidates = candidates
	session.FocusedNote = nil
	session.State = store.StateBrowsing
	m.logger.Printf("[STATE] Transitioned to BROWSING: %d candidates", len(candidates))
}

// TransitionToAggregated combines multiple notes into a single focused document
func (m *Manager) TransitionToAggregated(session *store.Session, candidates []store.Document, aggregatedContent string) {
	session.FocusedNote = &store.Document{
		ID:      "aggregated",
		Title:   "All Notes",
		Content: aggregatedContent,
	}
	session.State = store.StateFocused
	m.logger.Printf("[STATE] Transitioned to AGGREGATED: %d notes combined", len(candidates))
}
