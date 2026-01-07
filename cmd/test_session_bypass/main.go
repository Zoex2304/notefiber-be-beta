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

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         SESSION-SCOPED BYPASS TEST                                           ║")
	fmt.Println("║   Purpose: Validate that /bypass only needs to be used ONCE per session     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Conversation flow:
	// Message 1: /bypass What is the capital of France? -> Sets session to bypass mode
	// Message 2: Are you serious? (NO PREFIX) -> Should still be bypass
	// Message 3: What tourist spots are there? (NO PREFIX) -> Should still be bypass
	conversation := []struct {
		Message     string
		Description string
		HasPrefix   bool
	}{
		{"/bypass What is the capital of France?", "First message WITH /bypass prefix", true},
		{"Are you serious?", "Follow-up WITHOUT prefix - should still be bypass", false},
		{"What tourist spots are there?", "Third message WITHOUT prefix - should still be bypass", false},
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

	responses := make([]string, 0)
	allPassed := true

	for i, turn := range conversation {
		fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  MESSAGE %d: %s\n", i+1, turn.Description)
		fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n")

		prefixIndicator := "WITHOUT /bypass"
		if turn.HasPrefix {
			prefixIndicator = "WITH /bypass"
		}
		fmt.Printf("\n📤 USER [%s]: %s\n", prefixIndicator, turn.Message)

		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		reply, citations := sendChat(sessionID, turn.Message)
		responses = append(responses, reply)

		fmt.Printf("\n🤖 AI: %s\n", reply)

		// Analysis
		fmt.Println("\n📊 ANALYSIS:")
		fmt.Println(strings.Repeat("─", 60))

		// Check for RAG artifacts (should NOT be present in bypass mode)
		if hasRAGArtifacts(reply) {
			fmt.Println("   ❌ RAG ARTIFACTS DETECTED - Mode leaked to RAG!")
			allPassed = false
		} else {
			fmt.Println("   ✅ No RAG artifacts (pure LLM mode)")
		}

		// Check for citations (should be 0 in bypass mode)
		if citations > 0 {
			fmt.Printf("   ⚠️  Citations found: %d (unexpected for bypass)\n", citations)
		} else {
			fmt.Println("   ✅ No citations (correct for bypass mode)")
		}

		fmt.Println(strings.Repeat("─", 60))
	}

	// Final Summary
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          FINAL SUMMARY                                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	fmt.Println("\n📋 CONVERSATION FLOW:")
	for i, turn := range conversation {
		prefix := "  (no prefix)"
		if turn.HasPrefix {
			prefix = "  (/bypass)"
		}
		fmt.Printf("\n[%d]%s USER: %s\n", i+1, prefix, turn.Message)
		fmt.Printf("         AI:   %s\n", truncate(responses[i], 80))
	}

	// Context check
	fmt.Println("\n" + strings.Repeat("═", 80))

	if len(responses) >= 3 {
		turn3Lower := strings.ToLower(responses[2])
		contextPreserved := strings.Contains(turn3Lower, "paris") ||
			strings.Contains(turn3Lower, "france") ||
			strings.Contains(turn3Lower, "eiffel") ||
			strings.Contains(turn3Lower, "louvre")

		if contextPreserved {
			fmt.Println("✅ CONTEXT PRESERVED: Turn 3 references France/Paris")
		} else {
			fmt.Println("⚠️  CONTEXT CHECK: Turn 3 may not reference prior context")
		}
	}

	if allPassed {
		fmt.Println("\n✅ SESSION-SCOPED BYPASS VERIFIED: All messages stayed in bypass mode!")
		fmt.Println("   User only needed /bypass on the FIRST message.")
	} else {
		fmt.Println("\n❌ TEST FAILED: Some messages leaked to RAG mode")
	}
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
