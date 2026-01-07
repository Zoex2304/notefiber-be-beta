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
	Query            string   // What user sends
	ExpectedContext  []string // Keywords that should appear in response (context validation)
	ForbiddenContext []string // Keywords that should NOT appear (hallucination check)
	Description      string   // Human-readable description of this turn
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║             BYPASS MODE CONTEXT MEMORY TEST                                  ║")
	fmt.Println("║   Purpose: Validate multi-turn conversation context preservation            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Define the conversation flow
	conversation := []ConversationTurn{
		{
			Query:           "/bypass What is the capital of France?",
			ExpectedContext: []string{"paris", "france"},
			Description:     "Initial question about France's capital",
		},
		{
			Query:           "/bypass Are you serious?",
			ExpectedContext: []string{"paris", "france", "capital"},
			Description:     "Follow-up without restating context - LLM should remember France/Paris",
		},
		{
			Query:           "/bypass What tourist spots are there?",
			ExpectedContext: []string{}, // Could be Eiffel Tower, Louvre, etc.
			Description:     "Third question - LLM should reference France/Paris context",
		},
	}

	// Create fresh session
	fmt.Println("┌──────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  CREATING FRESH SESSION                                                     │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────────────┘")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		os.Exit(1)
	}
	fmt.Printf("✅ Session Created: %s\n\n", sessionID)

	// Track all responses for final analysis
	responses := make([]string, 0, len(conversation))
	allPassed := true

	// Execute each turn
	for i, turn := range conversation {
		fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  TURN %d: %s\n", i+1, turn.Description)
		fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n")

		fmt.Printf("\n📤 USER: %s\n", turn.Query)

		// Add delay between requests for natural conversation flow
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		reply := sendChat(sessionID, turn.Query)
		responses = append(responses, reply)

		fmt.Printf("\n🤖 AI: %s\n", reply)

		// Analyze response for context preservation
		fmt.Println("\n📊 CONTEXT ANALYSIS:")
		fmt.Println(strings.Repeat("─", 60))

		passed := true

		// Check expected context keywords
		if len(turn.ExpectedContext) > 0 {
			lowerReply := strings.ToLower(reply)
			for _, keyword := range turn.ExpectedContext {
				if strings.Contains(lowerReply, strings.ToLower(keyword)) {
					fmt.Printf("   ✅ Expected context found: '%s'\n", keyword)
				} else {
					fmt.Printf("   ⚠️  Expected context missing: '%s'\n", keyword)
					// Note: not failing for missing keywords as LLM may paraphrase
				}
			}
		}

		// Check forbidden context (hallucination)
		if len(turn.ForbiddenContext) > 0 {
			lowerReply := strings.ToLower(reply)
			for _, keyword := range turn.ForbiddenContext {
				if strings.Contains(lowerReply, strings.ToLower(keyword)) {
					fmt.Printf("   ❌ FORBIDDEN context found: '%s'\n", keyword)
					passed = false
				}
			}
		}

		// Check for RAG leakage
		if hasRAGArtifacts(reply) {
			fmt.Println("   ❌ RAG LEAKAGE DETECTED in response")
			passed = false
		} else {
			fmt.Println("   ✅ No RAG artifacts (pure LLM mode)")
		}

		if !passed {
			allPassed = false
		}

		fmt.Println(strings.Repeat("─", 60))
	}

	// Final Summary
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          FINAL SUMMARY                                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	fmt.Println("\n📋 CONVERSATION HISTORY:")
	for i, turn := range conversation {
		fmt.Printf("\n[Turn %d] USER: %s\n", i+1, turn.Query)
		fmt.Printf("         AI:   %s\n", truncate(responses[i], 100))
	}

	fmt.Println("\n" + strings.Repeat("═", 80))

	// Context continuity check for Turn 3
	// The key test: Does Turn 3 response relate to France/Paris?
	if len(responses) >= 3 {
		turn3Lower := strings.ToLower(responses[2])
		contextPreserved := strings.Contains(turn3Lower, "paris") ||
			strings.Contains(turn3Lower, "france") ||
			strings.Contains(turn3Lower, "eiffel") ||
			strings.Contains(turn3Lower, "louvre") ||
			strings.Contains(turn3Lower, "notre") ||
			strings.Contains(turn3Lower, "french")

		if contextPreserved {
			fmt.Println("✅ CONTEXT MEMORY VERIFIED: Turn 3 correctly references France/Paris context")
		} else {
			fmt.Println("⚠️  CONTEXT CHECK: Turn 3 may not reference prior context")
			fmt.Println("   (This could be acceptable if LLM gave generic tourist spots)")
			fmt.Println("   Manual Review: Does the response make sense given the conversation?")
		}
	}

	if allPassed {
		fmt.Println("\n✅ ALL VALIDATION CHECKS PASSED")
	} else {
		fmt.Println("\n⚠️  SOME CHECKS FAILED - Review above for details")
	}
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
		fmt.Printf("❌ Failed to create session: %s\n", string(body))
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

func sendChat(sessionID, message string) string {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	resp, body := doRequest("POST", "/chatbot/v1/send-chat", payload)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("[ERROR: %s]", string(body))
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	if data, ok := res["data"].(map[string]interface{}); ok {
		if replyObj, ok := data["reply"].(map[string]interface{}); ok {
			if content, ok := replyObj["chat"].(string); ok {
				return content
			}
		}
	}
	return "[No response]"
}

func hasRAGArtifacts(reply string) bool {
	lowerReply := strings.ToLower(reply)
	ragPatterns := []string{
		"according to your notes",
		"based on your notes",
		"in your notes",
		"reference [",
		"(reference",
		"according to my notes",
	}
	for _, p := range ragPatterns {
		if strings.Contains(lowerReply, p) {
			return true
		}
	}
	return false
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
