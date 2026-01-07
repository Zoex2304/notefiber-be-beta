//go:build ignore
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/llm/ollama"
	"ai-notetaking-be/pkg/rag/executor"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// --- MOCK REPOSITORY ---
type MockNoteRepo struct {
	Notes map[string]*entity.Note
}

func (m *MockNoteRepo) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Note, error) {
	// Simple mock: assume spec is ByID
	// We cheat and iterate because we know how specs work or just return based on ID if we could inspect spec
	// But specification.ByID is a struct.
	// Since we can't easily inspect the internal spec without reflection or type assertion,
	// checking the IDs from my Notes map is hard if I don't know which ID is requested.
	// BUT, typically FindOne uses ByID.

	// Let's rely on the fact that ContextGrounder calls FindOne with specification.ByID{ID: nid}
	// We can try to cast the first spec.
	if len(specs) > 0 {
		if s, ok := specs[0].(specification.ByID); ok {
			if note, found := m.Notes[s.ID.String()]; found {
				return note, nil
			}
		}
	}
	return nil, fmt.Errorf("note not found in mock")
}
func (m *MockNoteRepo) Create(ctx context.Context, note *entity.Note) error { return nil }
func (m *MockNoteRepo) Update(ctx context.Context, note *entity.Note) error { return nil }
func (m *MockNoteRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (m *MockNoteRepo) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Note, error) {
	return nil, nil
}
func (m *MockNoteRepo) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	return 0, nil
}

// --- MOCK UOW ---
type MockUOW struct {
	NoteRepo contract.NoteRepository
}

func (m *MockUOW) NoteRepository() contract.NoteRepository { return m.NoteRepo }
func (m *MockUOW) Begin(ctx context.Context) error         { return nil }
func (m *MockUOW) Commit() error                           { return nil }
func (m *MockUOW) Rollback() error                         { return nil }

// Dummy impls for other repos
func (m *MockUOW) UserRepository() contract.UserRepository                     { return nil }
func (m *MockUOW) NotebookRepository() contract.NotebookRepository             { return nil }
func (m *MockUOW) NoteEmbeddingRepository() contract.NoteEmbeddingRepository   { return nil }
func (m *MockUOW) ChatSessionRepository() contract.ChatSessionRepository       { return nil }
func (m *MockUOW) ChatMessageRepository() contract.ChatMessageRepository       { return nil }
func (m *MockUOW) ChatMessageRawRepository() contract.ChatMessageRawRepository { return nil }
func (m *MockUOW) SubscriptionRepository() contract.SubscriptionRepository     { return nil }
func (m *MockUOW) FeatureRepository() contract.FeatureRepository               { return nil }
func (m *MockUOW) BillingRepository() contract.BillingRepository               { return nil }
func (m *MockUOW) RefundRepository() contract.RefundRepository                 { return nil }

func main() {
	fmt.Println("=== FULL RAG PIPELINE SIMULATION ===")

	// 1. Setup Dependencies
	logger := log.New(os.Stdout, "", log.LstdFlags)
	llmProvider := ollama.NewOllamaProvider("http://localhost:11434", "qwen2.5:7b")
	sessionRepo := memory.NewSessionRepository()

	// 2. Setup Mock Data
	note1ID := uuid.New()
	note2ID := uuid.New()
	note3ID := uuid.New()

	// Store Candidates text for logging
	fmt.Printf("Note 1 ID: %s (English Exam)\n", note1ID)
	fmt.Printf("Note 2 ID: %s (Final Exam)\n", note2ID)
	fmt.Printf("Note 3 ID: %s (Final Examination)\n", note3ID)

	mockRepo := &MockNoteRepo{
		Notes: map[string]*entity.Note{
			note1ID.String(): {
				Id:      note1ID,
				Title:   "english exam",
				Content: "Question 1: The report is submitted yesterday. (Answer: is)\nQuestion 2: She runs fast.",
			},
			note2ID.String(): {
				Id:      note2ID,
				Title:   "final exam",
				Content: "Question 1: Ubiquitous means Omnipresent.\nQuestion 2: Ambiguous means unclear.\nQuestion 3: Obscure means hidden.",
			},
			note3ID.String(): {
				Id:      note3ID,
				Title:   "English Final Examination",
				Content: "1. Neither the manager nor employees WERE aware.\n2. Rarely DO we see such dedication.\n3. Had I KNOWN the answer.",
			},
		},
	}
	uow := &MockUOW{NoteRepo: mockRepo}

	// 3. Construct Pipeline Manualy
	// We can't use NewPipelineExecutor because it creates its own internal Resolver/Grounder.
	// We want to inject ours if needed, but actually NewPipelineExecutor creates standard ones.
	// AND NewPipelineExecutor returns *PipelineExecutor struct.
	// AND the struct fields are unexported?
	// Wait, Check PipelineExecutor struct definition...
	// It has unexported fields: intentResolver, grounder, generator.
	// So I MUST use NewPipelineExecutor?
	// If I use NewPipelineExecutor, I can't inject a Grounder that uses a mock SearchOrchestrator easily?
	// But NewGrounder takes searchOrchestrator.
	// So I can create nil searcher.

	// BUT, I can construct the struct directly if I am in main package?
	// The struct fields are lowercase (unexported) in `executor` package.
	// So main package CANNOT set them.
	// I MUST use `executor.NewPipelineExecutor`.

	// `executor.NewPipelineExecutor` takes `(llm, searchOrchestrator, sessionRepo, logger)`.
	// I can pass `nil` for searchOrchestrator.

	exec := executor.NewPipelineExecutor(llmProvider, nil, sessionRepo, logger)

	// 4. Setup Session (Browsing Mode)
	userID := uuid.New()
	sessionID := uuid.New()

	initialSession := &store.Session{
		ID:     sessionID.String(),
		UserID: userID.String(),
		State:  store.StateBrowsing,
		Candidates: []store.Document{
			{ID: note1ID.String(), Title: "english exam"},
			{ID: note2ID.String(), Title: "final exam"},
			{ID: note3ID.String(), Title: "English Final Examination"},
		},
	}
	sessionRepo.Save(initialSession)

	// 5. Run Queries
	queries := []string{
		"Please answer all questions in the first file.",
		"Please answer all questions in the second file.",
		"Please answer all questions in the third file.",
		"How many questions are there in total from the three files?",
		"Summarize the first file.",
	}

	for i, q := range queries {
		fmt.Printf("\n\n-----------------------------\nTEST CASE %d: \"%s\"\n-----------------------------\n", i+1, q)

		// Reset session state to Browsing for fairness?
		// Or let it flow? Real user flow might keep focus.
		// The user challenge implies a sequence.
		// But "How many... from three files" might require moving from Focus -> Aggregate.
		// The Resolver handles this.
		// However, "FocusedNote" persists in session.
		// If I am focused on Note 3, and ask "Summarize first file", Resolver sees "FOCUS 1".
		// It works.
		// So continuous flow is fine.

		start := time.Now()
		result, err := exec.Execute(context.Background(), userID, sessionID, q, []llm.Message{}, uow)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}
		duration := time.Since(start)

		fmt.Printf("Reply: %s\n", result.Reply)
		fmt.Printf("Citations: %v\n", result.Citations)
		fmt.Printf("Duration: %s\n", duration)
	}
}
