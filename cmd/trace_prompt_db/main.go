package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/llm"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

/*
╔══════════════════════════════════════════════════════════════════════════════╗
║             DATABASE-LEVEL PROMPT TRACE SCRIPT                               ║
╠══════════════════════════════════════════════════════════════════════════════╣
║  Purpose: Query ChatMessageRaw table directly to show exactly what prompts   ║
║  are being loaded as conversation history. This traces the data BEFORE       ║
║  it reaches the LLM.                                                         ║
╚══════════════════════════════════════════════════════════════════════════════╝

USAGE:
  go run cmd/trace_prompt_db/main.go <session_id>

  Replace <session_id> with a chat session UUID.
  Session IDs can be obtained from creating a session via API.
*/

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/trace_prompt_db/main.go <session_id>")
		fmt.Println("\nTo get a session ID:")
		fmt.Println("  1. Create session: POST /api/chatbot/v1/create-session")
		fmt.Println("  2. Send a chat: POST /api/chatbot/v1/send-chat")
		fmt.Println("  3. Use the session_id from the response")
		os.Exit(1)
	}

	sessionIDStr := os.Args[1]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		fmt.Printf("❌ Invalid session ID: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║             DATABASE-LEVEL PROMPT TRACE                                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Target Session: %s\n\n", sessionID)

	// Load config & connect to DB
	godotenv.Load()
	db := connectDB()
	uowFactory := unitofwork.NewRepositoryFactory(db)
	ctx := context.Background()

	// Query ChatMessageRaw table
	uow := uowFactory.NewUnitOfWork(ctx)

	rawChats, err := uow.ChatMessageRawRepository().FindAll(ctx,
		specification.ByChatSessionID{ChatSessionID: sessionID},
		specification.OrderBy{Field: "created_at", Desc: false},
	)
	if err != nil {
		fmt.Printf("❌ Failed to query ChatMessageRaw: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📊 Found %d raw messages in session\n\n", len(rawChats))

	// Display each message
	fmt.Println("┌──────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                     RAW CONVERSATION HISTORY                                 │")
	fmt.Println("│     (These are the prompts that get passed to the LLM)                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────────────┘")

	ragPromptDetected := false
	for i, msg := range rawChats {
		fmt.Printf("\n═══════════════════════════════════════════════════════════════════════════════\n")
		fmt.Printf("MESSAGE #%d\n", i+1)
		fmt.Printf("═══════════════════════════════════════════════════════════════════════════════\n")
		fmt.Printf("ID:        %s\n", msg.Id)
		fmt.Printf("Role:      %s\n", strings.ToUpper(msg.Role))
		fmt.Printf("CreatedAt: %s\n", msg.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("───────────────────────────────────────────────────────────────────────────────")
		fmt.Println("CONTENT:")
		fmt.Println(msg.Chat)
		fmt.Println("───────────────────────────────────────────────────────────────────────────────")

		// Check if this is a RAG system prompt
		isRAGPrompt := isRAGSystemPrompt(msg.Chat)
		if isRAGPrompt {
			ragPromptDetected = true
			fmt.Println("⚠️  WARNING: This message contains RAG SYSTEM PROMPT instructions!")
			fmt.Println("    This should NOT appear in bypass mode history.")
		}
	}

	// Simulate what LoadConversationHistory returns
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              SIMULATED LLM MESSAGE ARRAY                                     ║")
	fmt.Println("║   (What BypassPipeline.Execute() receives as 'history' parameter)           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	// Simulate history loading (last 10, reversed order)
	limit := 10
	if len(rawChats) > limit {
		rawChats = rawChats[len(rawChats)-limit:]
	}

	messages := make([]llm.Message, 0, len(rawChats))
	for _, rc := range rawChats {
		role := "user"
		if rc.Role == constant.ChatMessageRoleModel {
			role = "assistant"
		}
		messages = append(messages, llm.Message{
			Role:    role,
			Content: rc.Chat,
		})
	}

	fmt.Printf("\nTotal messages in LLM history: %d\n\n", len(messages))

	for i, m := range messages {
		content := m.Content
		if len(content) > 200 {
			content = content[:200] + "... [TRUNCATED]"
		}
		fmt.Printf("[%d] ROLE: %-9s | CONTENT: %s\n", i, strings.ToUpper(m.Role), truncateOneLine(content))
	}

	// Final diagnosis
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                           DIAGNOSIS                                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	if ragPromptDetected {
		fmt.Println("🔴 RAG PROMPT CONTAMINATION DETECTED")
		fmt.Println("")
		fmt.Println("   The ChatMessageRaw table contains RAG system prompts that will be")
		fmt.Println("   included in the conversation history for ALL modes, including bypass.")
		fmt.Println("")
		fmt.Println("   Root Cause: CreateSession() in chatbot_service.go seeds these prompts")
		fmt.Println("   unconditionally at session creation time.")
		fmt.Println("")
		fmt.Println("   Impact: When bypass mode executes, it calls LoadConversationHistory()")
		fmt.Println("   which loads these RAG prompts, contaminating the LLM context.")
	} else {
		fmt.Println("🟢 No RAG prompts detected in this session's history.")
		fmt.Println("   This could mean:")
		fmt.Println("   - Session was created differently (custom/test)")
		fmt.Println("   - The RAG prompts have been removed")
		fmt.Println("   - Session ID may be incorrect")
	}
}

func connectDB() *gorm.DB {
	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.Database.Connection), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func isRAGSystemPrompt(content string) bool {
	ragIndicators := []string{
		"pattern-based logic",
		"According to [note_title]",
		"CITATION FORMAT",
		"Reference [N]",
		"personal notes",
		"NOTES DATABASE",
		"knowledge assistant",
		"pattern matching internally",
		"cite sources naturally",
	}

	lowerContent := strings.ToLower(content)
	for _, indicator := range ragIndicators {
		if strings.Contains(lowerContent, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

func truncateOneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
