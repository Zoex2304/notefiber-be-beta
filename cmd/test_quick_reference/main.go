package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	apiBaseURL = "http://localhost:3000/api"
	authToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc1MzM0NzEsInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.d2FCLmPtYypI9s5cqKDmhpVWq_eahtOv8miUhoNbBM4"
)

// Response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SemanticSearchResult struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	NotebookID     string   `json:"notebook_id"`
	SearchType     string   `json:"search_type"`
	RelevanceScore *float64 `json:"relevance_score"`
}

type ChatResponse struct {
	ChatSessionID      string                `json:"chat_session_id"`
	Title              string                `json:"title"`
	Mode               string                `json:"mode"`
	ResolvedReferences []ResolvedRefResponse `json:"resolved_references"`
	Reply              *ChatMessage          `json:"reply"`
}

type ResolvedRefResponse struct {
	NoteID   string `json:"note_id"`
	Title    string `json:"title"`
	Resolved bool   `json:"resolved"`
}

type ChatMessage struct {
	Chat string `json:"chat"`
}

func main() {
	fmt.Println("=== Quick Reference System v1.5.0 Test Script ===\n")

	// Test 1: Semantic Search
	fmt.Println("📋 TEST 1: Semantic Search for 'exam'")
	testSemanticSearch("exam")

	fmt.Println("\n" + strings.Repeat("─", 50) + "\n")

	// Test 2: Semantic Search with different query
	fmt.Println("📋 TEST 2: Semantic Search for 'notes'")
	testSemanticSearch("notes")

	fmt.Println("\n" + strings.Repeat("─", 50) + "\n")

	// Test 3: Reference Parser Syntax (just display info)
	fmt.Println("📋 TEST 3: Reference Parser Syntax Examples")
	fmt.Println("   @notes:uuid-here          → UUID lookup")
	fmt.Println("   @notes:\"Meeting Notes\"    → Title match")
	fmt.Println("   [[Recipe Book]]           → Wiki-link syntax")
	fmt.Println("   Max 5 references per prompt")

	fmt.Println("\n=== Tests Complete ===")
}

func testSemanticSearch(query string) {
	encodedQuery := url.QueryEscape(query)
	reqURL := fmt.Sprintf("%s/note/v1/semantic-search?q=%s", apiBaseURL, encodedQuery)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		fmt.Printf("   ❌ Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   ❌ Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("   URL: %s\n", reqURL)
	fmt.Printf("   Status: %d\n", resp.StatusCode)

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Printf("   ❌ Error parsing response: %v\n", err)
		fmt.Printf("   Raw: %s\n", string(body))
		return
	}

	fmt.Printf("   Success: %v\n", apiResp.Success)
	fmt.Printf("   Message: %s\n", apiResp.Message)

	// Check if data exists and parse it
	if apiResp.Data == nil {
		fmt.Printf("   ⚠️  Data: null (no results)\n")
		return
	}

	// Parse data as array of results
	dataBytes, _ := json.Marshal(apiResp.Data)
	var results []SemanticSearchResult
	if err := json.Unmarshal(dataBytes, &results); err != nil {
		fmt.Printf("   ⚠️  Could not parse data as array: %v\n", err)
		fmt.Printf("   Raw data: %s\n", string(dataBytes))
		return
	}

	fmt.Printf("   📊 Results: %d notes found\n", len(results))

	if len(results) == 0 {
		fmt.Println("   ⚠️  No results returned - check threshold or embeddings")
		return
	}

	// Display results
	for i, r := range results {
		scoreStr := "N/A"
		if r.RelevanceScore != nil {
			scoreStr = fmt.Sprintf("%.4f", *r.RelevanceScore)
		}
		titlePreview := r.Title
		if len(titlePreview) > 30 {
			titlePreview = titlePreview[:30] + "..."
		}
		fmt.Printf("   %d. [%s] %s (Score: %s, Type: %s)\n",
			i+1, r.ID[:8], titlePreview, scoreStr, r.SearchType)
	}

	fmt.Println("   ✅ Semantic search working!")
}
