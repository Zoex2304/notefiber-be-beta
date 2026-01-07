// FILE: pkg/rag/filter.go
// PURPOSE: Multi-layer RAG filtering for improved retrieval precision

package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/pkg/lexical"
)

// ============================================================
// TYPES
// ============================================================

// EnhancedQuery contains the original query with extracted metadata
type EnhancedQuery struct {
	Original string   `json:"original"`
	Keywords []string `json:"keywords"`
	Domain   string   `json:"domain"` // e.g., "colors", "shopping", "food"
}

// ScoredDocument contains a document with its relevance score
type ScoredDocument struct {
	Document *entity.NoteEmbedding
	Score    int    `json:"score"`  // 0-10
	Reason   string `json:"reason"` // Why this score
}

// ============================================================
// LAYER 1: QUERY ENHANCEMENT
// ============================================================

// EnhanceQuery extracts keywords and domain from the user query
// This is a lightweight preprocessing step (no LLM call for speed)
func EnhanceQuery(query string) *EnhancedQuery {
	enhanced := &EnhancedQuery{
		Original: query,
		Keywords: extractKeywords(query),
		Domain:   detectDomain(query),
	}
	return enhanced
}

// extractKeywords extracts important words from the query
func extractKeywords(query string) []string {
	// Remove common stop words (Indonesian + English)
	stopWords := map[string]bool{
		"apa": true, "yang": true, "saya": true, "aku": true, "kamu": true,
		"ini": true, "itu": true, "di": true, "ke": true, "dari": true,
		"untuk": true, "dengan": true, "adalah": true, "ada": true,
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"what": true, "my": true, "your": true, "i": true, "me": true,
		"sudah": true, "udah": true, "ya": true, "dong": true, "nih": true,
		"beli": true, "punya": true, "mau": true,
	}

	words := strings.Fields(strings.ToLower(query))
	keywords := make([]string, 0)

	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,?!;:")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// detectDomain identifies the topic/domain of the query
func detectDomain(query string) string {
	query = strings.ToLower(query)

	// Domain detection rules
	domains := map[string][]string{
		"colors":   {"warna", "color", "hitam", "putih", "merah", "biru", "hijau", "kuning", "pink"},
		"shopping": {"beli", "belanja", "sparepart", "harga", "toko", "beli"},
		"food":     {"makanan", "makan", "masak", "resep", "food", "bakso", "mie", "nasi"},
		"schedule": {"jadwal", "tanggal", "waktu", "jam", "schedule", "meeting"},
		"personal": {"kesukaan", "favorit", "favorite", "suka", "hobby"},
	}

	for domain, keywords := range domains {
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) {
				return domain
			}
		}
	}

	return "general"
}

// ============================================================
// LAYER 2: RELEVANCE SCORING (LLM-based)
// ============================================================

// ScoreDocuments uses LLM to score each document's relevance to the query
func ScoreDocuments(ctx context.Context, query string, docs []*entity.NoteEmbedding) ([]*ScoredDocument, error) {
	scored := make([]*ScoredDocument, 0, len(docs))

	for _, doc := range docs {
		score, reason, err := scoreDocument(ctx, query, doc.Document)
		if err != nil {
			// On error, give benefit of the doubt with medium score
			score = 5
			reason = "scoring failed, using default"
		}

		scored = append(scored, &ScoredDocument{
			Document: doc,
			Score:    score,
			Reason:   reason,
		})
	}

	return scored, nil
}

// scoreDocument scores a single document using Ollama
func scoreDocument(ctx context.Context, query string, document string) (int, string, error) {
	// Parse Lexical JSON before scoring (LLM can't understand raw JSON)
	parsedDoc := parseDocumentForScoring(document)

	// Truncate for faster scoring
	if len(parsedDoc) > 500 {
		parsedDoc = parsedDoc[:500] + "..."
	}

	prompt := fmt.Sprintf(constant.RAGRelevanceScoringPrompt, query, parsedDoc)

	// Build Ollama request
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}

	payload := map[string]interface{}{
		"model":    getOllamaModel(),
		"messages": messages,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": 0.1, // Low temperature for consistent scoring
			"num_predict": 100, // Short response
		},
	}

	payloadJSON, _ := json.Marshal(payload)

	// Send request
	url := getOllamaBaseURL() + "/api/chat"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	// Parse Ollama response
	var ollamaRes struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &ollamaRes); err != nil {
		return 0, "", err
	}

	// Parse score from response
	return parseScoreResponse(ollamaRes.Message.Content)
}

// parseScoreResponse extracts score and reason from LLM response
func parseScoreResponse(response string) (int, string, error) {
	// Clean response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result struct {
		Score  int    `json:"score"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to extract score manually if JSON parsing fails
		return 5, "parse failed", nil
	}

	// Clamp score to 0-10
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 10 {
		result.Score = 10
	}

	return result.Score, result.Reason, nil
}

// ============================================================
// LAYER 3: CONTEXT FILTERING
// ============================================================

// FilterRelevantDocuments filters documents by score threshold
func FilterRelevantDocuments(scored []*ScoredDocument, threshold int) []*ScoredDocument {
	filtered := make([]*ScoredDocument, 0)

	for _, doc := range scored {
		if doc.Score >= threshold {
			filtered = append(filtered, doc)
		}
	}

	return filtered
}

// HasRelevantDocuments checks if there are any documents above threshold
func HasRelevantDocuments(scored []*ScoredDocument, threshold int) bool {
	for _, doc := range scored {
		if doc.Score >= threshold {
			return true
		}
	}
	return false
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func getOllamaBaseURL() string {
	url := os.Getenv("OLLAMA_BASE_URL")
	if url == "" {
		return "http://localhost:11434"
	}
	return url
}

func getOllamaModel() string {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		return "llama3.1:8b"
	}
	return model
}

// parseDocumentForScoring extracts and parses the content from document for scoring
// Document format: "Note Title: X\nNotebook Title: Y\n\n<CONTENT>\n\nCreated At: ..."
func parseDocumentForScoring(document string) string {
	// Split by double newline to find content section
	parts := strings.SplitN(document, "\n\n", 3)
	if len(parts) < 3 {
		// Can't find content section, try parsing whole document
		return lexical.ParseContent(document)
	}

	// parts[0] = "Note Title: X\nNotebook Title: Y"
	// parts[1] = content (might be Lexical JSON)
	// parts[2] = "Created At: ...\nUpdated At: ..."
	header := parts[0]
	content := parts[1]

	// Parse the content if it's Lexical JSON
	parsedContent := lexical.ParseContent(content)

	// Return header + parsed content (skip footer for brevity)
	return fmt.Sprintf("%s\n\n%s", header, parsedContent)
}
