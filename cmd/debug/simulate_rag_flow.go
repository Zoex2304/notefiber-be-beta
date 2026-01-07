//go:build ignore
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"ai-notetaking-be/internal/dto"
	mockmemory "ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/internal/service"
	"ai-notetaking-be/pkg/database"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/llm/ollama"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// Mock Factory
type MockUOWFactory struct {
	db *gorm.DB
}

func (m *MockUOWFactory) NewUnitOfWork(ctx context.Context) unitofwork.UnitOfWork {
	return unitofwork.NewUnitOfWork(m.db)
}

func main() {
	godotenv.Load()
	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DB_CONNECTION_STRING required")
	}

	// 1. Setup DB
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal(err)
	}
	uowFactory := &MockUOWFactory{db: db}

	// 2. Setup Components
	// EXPLICIT URLS TO FIX "unsupported protocol scheme"
	embProvider := embedding.NewOllamaProvider("http://localhost:11434", "nomic-embed-text")
	llmProvider := ollama.NewOllamaProvider("http://localhost:11434", "qwen2.5:7b")

	sessionRepo := mockmemory.NewSessionRepository()

	// 3. Setup Service
	chatbotSvc := service.NewChatbotService(uowFactory, embProvider, llmProvider, sessionRepo)

	// 4. Run Flow
	ctx := context.Background()
	noteID := uuid.MustParse("5336a88a-35bd-4a19-b7e0-363828359b79")

	var noteOwnerIDStr string
	err = db.Table("notes").Select("user_id").Where("id = ?", noteID).Scan(&noteOwnerIDStr).Error
	if err != nil {
		log.Fatalf("Failed to find owner of note: %v", err)
	}
	noteOwnerID := uuid.MustParse(noteOwnerIDStr)
	fmt.Printf("Simulating User: %s\n", noteOwnerID)

	sessResp, err := chatbotSvc.CreateSession(ctx, noteOwnerID)
	if err != nil {
		log.Fatal(err)
	}
	sessionID := sessResp.Id
	fmt.Printf("Session Created: %s\n", sessionID)

	// 4b. Turn 1: "answer my english exam"
	fmt.Println("\n--- Turn 1: \"answer my english exam\" ---")
	resp1, err := chatbotSvc.SendChat(ctx, noteOwnerID, &dto.SendChatRequest{
		ChatSessionId: sessionID,
		Chat:          "answer my english exam",
	})
	if err != nil {
		log.Printf("Chat 1 failed: %v", err)
	}
	if resp1 != nil && resp1.Reply != nil {
		fmt.Printf("AI: %s\n", resp1.Reply.Chat)
		printCitations(resp1.Reply.Citations)
	}

	// 4c. Turn 2: "the third one"
	fmt.Println("\n--- Turn 2: \"i want you to answer all of that\" ---")
	time.Sleep(1 * time.Second)

	resp2, err := chatbotSvc.SendChat(ctx, noteOwnerID, &dto.SendChatRequest{
		ChatSessionId: sessionID,
		Chat:          "the third one",
	})
	if err != nil {
		log.Printf("Chat 2 failed: %v", err)
	}
	if resp2 != nil && resp2.Reply != nil {
		fmt.Printf("AI: %s\n", resp2.Reply.Chat)
		printCitations(resp2.Reply.Citations)
	}
}

func printCitations(cits []dto.CitationDTO) {
	if len(cits) == 0 {
		return
	}
	fmt.Println("Citations:")
	for _, c := range cits {
		fmt.Printf("- %s (%s)\n", c.Title, c.NoteId)
	}
}
