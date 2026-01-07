//go:build ignore
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/repository/implementation"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/pkg/database"
	"ai-notetaking-be/pkg/embedding"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	// 2. Connect DB
	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DB_CONNECTION_STRING not set")
	}
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// 3. Setup Repos
	noteRepo := implementation.NewNoteRepository(db)
	embProvider := embedding.NewOllamaProvider("", "")

	// 4. Get Note ID
	noteIDStr := "5336a88a-35bd-4a19-b7e0-363828359b79"
	if len(os.Args) > 1 {
		noteIDStr = os.Args[1]
	}
	noteID := uuid.MustParse(noteIDStr)

	ctx := context.Background()

	// 5. Fetch Original Note
	fmt.Println("--- FETCHING ORIGINAL NOTE ---")
	note, err := noteRepo.FindOne(ctx, specification.ByID{ID: noteID})
	if err != nil {
		log.Fatalf("Failed to fetch note: %v", err)
	}
	if note == nil {
		log.Fatalf("Note not found: %s", noteID)
	}
	fmt.Printf("Note Title: %s\n", note.Title)
	fmt.Printf("Total Length: %d chars\n", len(note.Content))
	fmt.Println("--------------------------------")

	// 6. Fetch Chunks
	fmt.Println("--- FETCHING EMBEDDING CHUNKS ---")

	var embeddings []model.NoteEmbedding
	if err := db.Where("note_id = ?", noteID).Order("chunk_index asc").Find(&embeddings).Error; err != nil {
		log.Fatalf("Failed to fetch embeddings: %v", err)
	}

	fmt.Printf("Found %d chunks for this note.\n", len(embeddings))

	for i, emb := range embeddings {
		fmt.Printf("[Chunk %d] Index: %d, Length: %d chars\n", i, emb.ChunkIndex, len(emb.Document))
		fmt.Printf("Preview: %s...\n", mbSubstr(emb.Document, 50))
	}
	fmt.Println("--------------------------------")

	// 7. Verify Coverage
	fmt.Println("--- VERIFYING COVERAGE ---")
	startSample := mbSubstr(note.Content, 100)
	midIndex := len(note.Content) / 2
	midSample := mbSubstr(note.Content[midIndex:], 100)
	endIndex := len(note.Content) - 100
	if endIndex < 0 {
		endIndex = 0
	}
	endSample := mbSubstr(note.Content[endIndex:], 100)

	// START CHECK
	found := false
	for _, c := range embeddings {
		if strings.Contains(c.Document, startSample) {
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("[START] NOT FOUND! Sample: %s\n", startSample)
	} else {
		fmt.Println("[START] Found")
	}

	// MID CHECK
	found = false
	for _, c := range embeddings {
		if strings.Contains(c.Document, midSample) {
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("[MIDDLE] NOT FOUND! Sample: %s\n", midSample)
	} else {
		fmt.Println("[MIDDLE] Found")
	}

	// END CHECK
	found = false
	for _, c := range embeddings {
		if strings.Contains(c.Document, endSample) {
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("[END] NOT FOUND! Sample: %s\n", endSample)
	} else {
		fmt.Println("[END] Found")
	}
	fmt.Println("--------------------------------")

	// 8. Test Search
	query := "Identify the error neither the manager nor the employees"
	fmt.Printf("--- TEST QUERY: \"%s\" ---\n", query)

	embRes, err := embProvider.Generate(query, "RETRIEVAL_QUERY")
	if err != nil {
		log.Fatalf("Embedding gen failed: %v", err)
	}

	// Manual Cosine Sim
	bestScore := float32(-1.0)
	bestChunkIdx := -1

	for i, emb := range embeddings {
		vec := emb.EmbeddingValue.Slice()
		score := cosineSimilarity(embRes.Embedding.Values, vec)
		fmt.Printf("Chunk %d Score: %.4f\n", i, score)
		if score > bestScore {
			bestScore = score
			bestChunkIdx = i
		}
	}

	if bestChunkIdx != -1 {
		fmt.Printf(">> Best Match: Chunk %d (Score: %.4f)\n", bestChunkIdx, bestScore)
		fmt.Printf(">> Content Preview: %s\n", mbSubstr(embeddings[bestChunkIdx].Document, 200))
	}
}

func mbSubstr(s string, l int) string {
	r := []rune(s)
	if len(r) > l {
		return string(r[:l])
	}
	return s
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, magA, magB float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (sqrt(magA) * sqrt(magB))
}

func sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}
