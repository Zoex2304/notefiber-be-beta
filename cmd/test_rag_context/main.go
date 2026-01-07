package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiBaseURL = "http://localhost:3000/api"
	authToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0OTAxNTksInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.jaUJYwutyRYvuv_G6zYnbjWuoDdaHcQb8VgYEhVRDpQ"
)

// ConversationTurn represents a single turn in the conversation
type ConversationTurn struct {
	Query           string
	ExpectedContext []string // Keywords that should appear (context validation)
	Description     string
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                RAG MODE CONTEXT MEMORY TEST                                  ║")
	fmt.Println("║   Purpose: Validate multi-turn context in note-based conversations          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Define RAG conversation flow (using English exam notes)
	conversation := []ConversationTurn{
		{
			Query:           "What do my notes say about the English exam?",
			ExpectedContext: []string{"english", "exam"},
			Description:     "Initial RAG query about English exam notes",
		},
		{
			Query:           "Tell me more about that",
			ExpectedContext: []string{}, // Should reference the English exam context
			Description:     "Follow-up without explicit context - LLM should remember English exam",
		},
		{
			Query:           "What topics are covered?",
			ExpectedContext: []string{}, // Should reference English exam topics
			Description:     "Third question - should stay in English exam context",
		},
	}

	// Create fresh session
	fmt.Println("┌──────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  CREATING FRESH SESSION (RAG MODE)                                          │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────────────┘")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		os.Exit(1)
	}
	fmt.Printf("✅ Session Created: %s\n\n", sessionID)

	// Track responses
	responses := make([]string, 0, len(conversation))
	citationCounts := make([]int, 0, len(conversation))

	// Execute each turn
	for i, turn := range conversation {
		fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  TURN %d: %s\n", i+1, turn.Description)
		fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n")

		fmt.Printf("\n📤 USER: %s\n", turn.Query)

		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		reply, citations := sendChat(sessionID, turn.Query)
		responses = append(responses, reply)
		citationCounts = append(citationCounts, citations)

		fmt.Printf("\n🤖 AI: %s\n", reply)

		// Analysis
		fmt.Println("\n📊 CONTEXT ANALYSIS:")
		fmt.Println(strings.Repeat("─", 60))

		// Check expected keywords
		if len(turn.ExpectedContext) > 0 {
			lowerReply := strings.ToLower(reply)
			for _, kw := range turn.ExpectedContext {
				if strings.Contains(lowerReply, strings.ToLower(kw)) {
					fmt.Printf("   ✅ Expected context found: '%s'\n", kw)
				} else {
					fmt.Printf("   ⚠️  Expected context missing: '%s'\n", kw)
				}
			}
		}

		// Show citation count
		if citations > 0 {
			fmt.Printf("   📎 Citations: %d (RAG retrieved notes)\n", citations)
		} else {
			fmt.Println("   ℹ️  No citations (may be answering from context)")
		}

		fmt.Println(strings.Repeat("─", 60))
	}

	// Final Summary
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          FINAL SUMMARY                                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	fmt.Println("\n📋 CONVERSATION FLOW:")
	for i, turn := range conversation {
		fmt.Printf("\n[Turn %d] USER: %s\n", i+1, turn.Query)
		fmt.Printf("         AI:   %s\n", truncate(responses[i], 100))
		fmt.Printf("         Citations: %d\n", citationCounts[i])
	}

	fmt.Println("\n" + strings.Repeat("═", 80))

	// Context continuity check
	// Turn 2 and 3 should relate to English exam without explicit mention
	turn2Lower := strings.ToLower(responses[1])
	turn3Lower := strings.ToLower(responses[2])

	contextPreserved := strings.Contains(turn2Lower, "english") ||
		strings.Contains(turn2Lower, "exam") ||
		strings.Contains(turn3Lower, "english") ||
		strings.Contains(turn3Lower, "exam") ||
		strings.Contains(turn3Lower, "topic") ||
		strings.Contains(turn3Lower, "grammar") ||
		strings.Contains(turn3Lower, "vocabulary")

	if contextPreserved {
		fmt.Println("✅ CONTEXT MEMORY VERIFIED: Follow-up turns reference English exam context")
	} else {
		fmt.Println("⚠️  CONTEXT CHECK: Follow-ups may not reference prior context")
		fmt.Println("   Manual review recommended to verify conversational continuity")
	}

	fmt.Println("\n✅ RAG MODE CONTEXT TEST COMPLETE")
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
		fmt.Printf("❌ Failed: %s\n", string(body))
		return ""
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	if data, ok := res["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			return id
		}
	}
	return ""
}

func sendChat(sessionID, message string) (string, int) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	resp, body := doRequest("POST", "/chatbot/v1/send-chat", payload)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("[ERROR: %s]", string(body)), 0
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	reply := ""
	citationCount := 0

	if data, ok := res["data"].(map[string]interface{}); ok {
		if replyObj, ok := data["reply"].(map[string]interface{}); ok {
			if content, ok := replyObj["chat"].(string); ok {
				reply = content
			}
			if citations, ok := replyObj["citations"].([]interface{}); ok {
				citationCount = len(citations)
			}
		}
	}
	return reply, citationCount
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func doRequest(method, url string, body interface{}) (*http.Response, []byte) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, _ := http.NewRequest(method, apiBaseURL+url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Network Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}
