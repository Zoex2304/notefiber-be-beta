package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"ai-notetaking-be/internal/repository/implementation"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/pkg/database"
	"ai-notetaking-be/pkg/embedding"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DB_CONNECTION_STRING not set")
	}

	// Initialize embedding provider (Ollama - local)
	ollamaBaseURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaBaseURL == "" {
		ollamaBaseURL = "http://localhost:11434"
	}
	ollamaModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if ollamaModel == "" {
		ollamaModel = "nomic-embed-text"
	}
	embeddingProvider := embedding.NewOllamaProvider(ollamaBaseURL, ollamaModel)

	// Connect to DB
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	// Initialize repositories
	noteEmbeddingRepo := implementation.NewNoteEmbeddingRepository(db)
	noteRepo := implementation.NewNoteRepository(db)

	// === CONFIGURATION ===
	// Hardcoded for diagnostic test
	userIdStr := "d8e7d6dc-880a-4977-b9bb-f6e299213355"
	if len(os.Args) > 1 {
		userIdStr = os.Args[1]
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		log.Fatal("Invalid user ID:", err)
	}

	// === THRESHOLDS TO TEST ===
	dbThreshold := 0.0 // Current: No DB-level filtering
	logicThresholds := []float64{0.35, 0.30, 0.25, 0.20, 0.15, 0.10}

	// === TEST QUERIES ===
	queries := []string{
		"How much is my business profit?",
		"profit calculation",
		"revenue costs profit",
		"business financial summary",
		"profit formula",
	}

	ctx := context.Background()

	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("RAG RETRIEVAL DIAGNOSTIC TOOL")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Printf("User ID: %s\n", userId)
	fmt.Printf("DB Threshold: %.2f\n", dbThreshold)
	fmt.Println()

	// First, list all notes for this user
	fmt.Println("--- ALL USER NOTES ---")
	allNotes, err := noteRepo.FindAll(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		log.Printf("Failed to fetch notes: %v", err)
	} else {
		for i, note := range allNotes {
			fmt.Printf("%d. [%s] %s\n", i+1, note.Id.String()[:8], note.Title)
		}
		fmt.Printf("\nTotal: %d notes\n", len(allNotes))
	}
	fmt.Println()

	// Run diagnostics for each query
	for _, query := range queries {
		fmt.Println("-" + strings.Repeat("-", 79))
		fmt.Printf("QUERY: \"%s\"\n", query)
		fmt.Println("-" + strings.Repeat("-", 79))

		// Generate embedding for query
		embeddingRes, err := embeddingProvider.Generate(query, "RETRIEVAL_QUERY")
		if err != nil {
			log.Printf("Embedding failed for query '%s': %v", query, err)
			continue
		}

		// Search with no threshold (get all)
		topK := 10
		scoredResults, err := noteEmbeddingRepo.SearchSimilarWithScore(
			ctx,
			embeddingRes.Embedding.Values,
			topK,
			userId,
			dbThreshold, // No DB filtering
		)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}

		fmt.Printf("\nRaw Results (TopK=%d, DBThreshold=%.2f):\n", topK, dbThreshold)
		fmt.Println()

		// Build title map
		noteIds := make([]uuid.UUID, len(scoredResults))
		for i, r := range scoredResults {
			noteIds[i] = r.Embedding.NoteId
		}
		titleMap := make(map[string]string)
		if len(noteIds) > 0 {
			notes, _ := noteRepo.FindAll(ctx, specification.ByIDs{IDs: noteIds})
			for _, n := range notes {
				titleMap[n.Id.String()] = n.Title
			}
		}

		// Display all results with scores
		fmt.Printf("%-4s %-40s %-12s", "#", "Title", "Similarity")
		for _, thresh := range logicThresholds {
			fmt.Printf(" @%.2f", thresh)
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", 100))

		for i, res := range scoredResults {
			noteId := res.Embedding.NoteId.String()
			title := titleMap[noteId]
			if title == "" {
				title = "Unknown"
			}
			if len(title) > 38 {
				title = title[:35] + "..."
			}

			fmt.Printf("%-4d %-40s %-12.4f", i+1, title, res.Similarity)

			// Show pass/fail for each threshold
			for _, thresh := range logicThresholds {
				if res.Similarity >= thresh {
					fmt.Print("   Y  ")
				} else {
					fmt.Print("   -  ")
				}
			}
			fmt.Println()
		}
		fmt.Println()

		// Summary
		fmt.Println("Summary by Threshold:")
		for _, thresh := range logicThresholds {
			count := 0
			for _, res := range scoredResults {
				if res.Similarity >= thresh {
					count++
				}
			}
			fmt.Printf("  Threshold %.2f: %d notes pass\n", thresh, count)
		}
		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("ANALYSIS COMPLETE")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()
	fmt.Println("Current System Configuration:")
	fmt.Println("  pkg/rag/search/orchestrator.go:")
	fmt.Println("    - DBThreshold:    0.0  (no DB-level filtering)")
	fmt.Println("    - LogicThreshold: 0.35 (post-retrieval filtering)")
	fmt.Println("    - TopK:           5    (max candidates)")
	fmt.Println()
	fmt.Println("Additional Filters:")
	fmt.Println("  pkg/rag/context/grounder.go:")
	fmt.Println("    - evaluateRelevance(): LLM-based semantic filter")
	fmt.Println("    - Can further reduce candidates based on LLM judgment")
	fmt.Println()
	fmt.Println("Recommendations:")
	fmt.Println("  1. If relevant notes score below 0.35, lower LogicThreshold")
	fmt.Println("  2. If TopK=5 excludes relevant notes, increase TopK")
	fmt.Println("  3. Review LLM semantic filter prompts in grounder.go")
}
