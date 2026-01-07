package main

import (
	"context"
	"log"
	"os"

	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/database"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/llm"
	ragcontext "ai-notetaking-be/pkg/rag/context"
	"ai-notetaking-be/pkg/rag/intent"
	"ai-notetaking-be/pkg/rag/search"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// MockLLMProvider stubs the LLM interface
type MockLLMProvider struct{}

func (m *MockLLMProvider) Generate(ctx context.Context, prompt string, options ...llm.Option) (string, error) {
	return "mock response", nil
}

func (m *MockLLMProvider) Chat(ctx context.Context, history []llm.Message, options ...llm.Option) (string, error) {
	return "mock chat", nil
}

// MockEmbeddingProvider for Grounder constructor
type MockEmbeddingProvider struct{}

func (m *MockEmbeddingProvider) Generate(text string, taskType string) (*embedding.EmbeddingResponse, error) {
	return &embedding.EmbeddingResponse{
		Embedding: embedding.EmbeddingResponseEmbedding{
			Values: []float32{0.1, 0.2},
		},
	}, nil
}

func main() {
	// 1. Setup
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found")
	}
	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DB_CONNECTION_STRING not set")
	}
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal(err)
	}

	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	uow := unitofwork.NewUnitOfWork(db)
	llmProvider := &MockLLMProvider{}
	embedProvider := &MockEmbeddingProvider{}

	// Create Grounder
	orchestrator := search.NewOrchestrator(embedProvider, logger)
	grounder := ragcontext.NewGrounder(orchestrator, embedProvider, llmProvider, logger)

	// 2. Prepare Session with "English Final Examination" as Candidate
	// Note ID: 5336a88a-35bd-4a19-b7e0-363828359b79
	noteID := "5336a88a-35bd-4a19-b7e0-363828359b79"

	session := &store.Session{
		ID:    uuid.New().String(),
		State: store.StateBrowsing,
		Candidates: []store.Document{
			{ID: "dummy-id", Title: "Dummy Note", Content: "Dummy Content"},
			{ID: noteID, Title: "English Final Examination", Content: "PARTIAL CHUNK CONTENT"},
		},
	}

	// 3. Execute FOCUS on Index 1 (Note 2)
	intentObj := &intent.Intent{
		Action: intent.ActionFocus,
		Target: 1, // Focus on 2nd item
	}

	logger.Println("🚀 Executing groundFocus on Target 1...")
	result, err := grounder.Ground(context.Background(), intentObj, session, uow, uuid.MustParse("a2b94f4c-b674-433b-90be-65a91a37e7a3"), nil)

	if err != nil {
		log.Fatal("Groounding Failed:", err)
	}

	// 4. Verification
	focused := result.Session.FocusedNote
	logger.Printf("Focused Title: '%s'", focused.Title)
	logger.Printf("Focused Content Length: %d bytes", len(focused.Content))

	if len(focused.Content) > 100 {
		logger.Printf("Start: %.50s...", focused.Content)
	}

	if focused.Content == "PARTIAL CHUNK CONTENT" {
		logger.Fatal("❌ FAILURE: Content was NOT reloaded from DB (Still partial)")
	} else if len(focused.Content) > 5000 {
		logger.Println("✅ SUCCESS: Content reloaded (Full Note detected)")
	} else {
		logger.Println("⚠️  WARNING: Content changed but size is suspicious")
	}
}
