package main

import (
	"bytes"
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

// DTOs
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type SemanticSearchResult struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type CreateSessionResponse struct {
	ID string `json:"id"`
}

type NoteReferenceDTO struct {
	NoteId     string `json:"note_id"`
	SourceType string `json:"source_type"`
}

type SendChatRequest struct {
	ChatSessionId string             `json:"chat_session_id"`
	Chat          string             `json:"chat"`
	References    []NoteReferenceDTO `json:"references"`
}

type SendChatResponse struct {
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
	Chat      string        `json:"chat"`
	Citations []CitationDTO `json:"citations"`
}

type CitationDTO struct {
	NoteId string `json:"note_id"`
	Title  string `json:"title"`
}

func main() {
	fmt.Println("=== Test: Export Flow (Search -> Chat) ===\n")

	// 1. Semantic Search
	fmt.Println("1. Searching for 'exam'...")
	results := searchNotes("exam")

	if len(results) == 0 {
		fmt.Println("❌ No results found. Cannot proceed.")
		return
	}

	// 2. Select specific notes requested by user
	// Target partial IDs: 5336a88a and 87f519d7
	var selectedNotes []SemanticSearchResult
	targets := []string{"5336a88a", "87f519d7"}

	for _, r := range results {
		for _, t := range targets {
			if strings.HasPrefix(r.ID, t) {
				selectedNotes = append(selectedNotes, r)
				fmt.Printf("   ✅ Found target note: [%s] %s\n", r.ID[:8], r.Title)
			}
		}
	}

	if len(selectedNotes) != 2 {
		fmt.Printf("⚠️  Warning: Found %d target notes, expected 2. Using what we found.\n", len(selectedNotes))
		if len(selectedNotes) == 0 {
			return
		}
	}

	// 3. Create Chat Session
	fmt.Println("\n2. Creating new chat session...")
	sessionId := createSession()
	fmt.Printf("   Unknown Session ID: %s\n", sessionId)

	// 4. Send Chat with References
	fmt.Println("\n3. Sending chat with exported references...")

	refs := make([]NoteReferenceDTO, len(selectedNotes))
	for i, n := range selectedNotes {
		refs[i] = NoteReferenceDTO{
			NoteId:     n.ID,
			SourceType: "export",
		}
	}

	req := SendChatRequest{
		ChatSessionId: sessionId,
		Chat:          "Please provide a complete answer for these notes.",
		References:    refs,
	}

	chatResp := sendChat(req)

	// 5. Verify Results
	fmt.Println("\n4. Verifying Response...")
	fmt.Printf("   Mode: %s\n", chatResp.Mode)

	if chatResp.Mode == "explicit_rag" {
		fmt.Println("   ✅ Mode is 'explicit_rag' (Correct)")
	} else {
		fmt.Printf("   ❌ Mode matches expectation? NO (Expected explicit_rag)\n")
	}

	fmt.Printf("   Resolved Refs: %d\n", len(chatResp.ResolvedReferences))
	for _, rr := range chatResp.ResolvedReferences {
		status := "❌"
		if rr.Resolved {
			status = "✅"
		}
		fmt.Printf("   %s [%s] %s\n", status, rr.NoteID[:8], rr.Title)
	}

	if chatResp.Reply == nil {
		fmt.Println("❌ Received nil Reply from chatbot. Check previous errors.")
		return
	}

	fmt.Println("\n   🤖 AI Reply:")
	fmt.Printf("   \"%s\"\n", chatResp.Reply.Chat)

	fmt.Println("\n   Citations:")
	for _, c := range chatResp.Reply.Citations {
		fmt.Printf("   - %s\n", c.Title)
	}

	fmt.Println("\n=== Test Complete ===")
}

// Helper functions

func searchNotes(query string) []SemanticSearchResult {
	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("%s/note/v1/semantic-search?q=%s", apiBaseURL, encodedQuery)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var apiResp APIResponse
	json.Unmarshal(body, &apiResp)

	dataBytes, _ := json.Marshal(apiResp.Data)
	var results []SemanticSearchResult
	json.Unmarshal(dataBytes, &results)

	return results
}

func createSession() string {
	url := fmt.Sprintf("%s/chatbot/v1/create-session", apiBaseURL)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// fmt.Printf("DEBUG: CreateSession Raw: %s\n", string(body))

	var apiResp struct {
		Data CreateSessionResponse `json:"data"`
	}
	json.Unmarshal(body, &apiResp)

	if apiResp.Data.ID == "" {
		fmt.Printf("❌ Failed to get Session ID. Raw: %s\n", string(body))
	}

	return apiResp.Data.ID
}

func sendChat(payload SendChatRequest) SendChatResponse {
	url := fmt.Sprintf("%s/chatbot/v1/send-chat", apiBaseURL)
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	// No timeout - wait indefinitely
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("❌ Chat request failed: %v\n", err)
		return SendChatResponse{}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var apiResp struct {
		Data SendChatResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Printf("❌ JSON Parse Error: %v\nRaw: %s\n", err, string(body))
		return SendChatResponse{}
	}

	return apiResp.Data
}
