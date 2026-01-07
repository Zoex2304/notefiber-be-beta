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
	fmt.Println("║                    NUANCE FEATURE TEST                                       ║")
	fmt.Println("║   Purpose: Test different nuance modes and observe behavioral changes       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Available nuances: engineering, creative, formal, academic, concise
	nuanceTests := []struct {
		Key         string
		Description string
		Query       string
		Indicators  []string // Keywords that should appear with this nuance
	}{
		{
			Key:         "engineering",
			Description: "Technical/engineering analysis style",
			Query:       "What are the best practices for error handling?",
			Indicators:  []string{"edge case", "design", "pattern", "solid", "testing", "performance"},
		},
		{
			Key:         "creative",
			Description: "Creative/brainstorming style",
			Query:       "Give me ideas for a mobile app",
			Indicators:  []string{"idea", "explore", "imagine", "alternative", "creative", "unconventional"},
		},
		{
			Key:         "concise",
			Description: "Brief/direct style",
			Query:       "What is machine learning?",
			Indicators:  []string{}, // Will check for SHORT response
		},
	}

	for _, test := range nuanceTests {
		testNuance(test.Key, test.Description, test.Query, test.Indicators)
	}

	// Test session-scoped nuance
	fmt.Println("\n" + strings.Repeat("═", 80))
	testSessionScopedNuance()
}

func testNuance(key, description, query string, indicators []string) {
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  NUANCE: %s\n", key)
	fmt.Printf("║  Description: %s\n", description)
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session: %s\n", sessionID)

	message := fmt.Sprintf("/nuance:%s %s", key, query)
	fmt.Printf("\n📤 USER: %s\n", message)

	reply, citations := sendChat(sessionID, message)
	fmt.Printf("\n🤖 AI: %s\n", reply)

	// Analysis
	fmt.Println("\n📊 ANALYSIS:")
	fmt.Println(strings.Repeat("─", 60))

	// Check for RAG artifacts (should NOT be present in nuance mode)
	if hasRAGArtifacts(reply) {
		fmt.Println("   ⚠️  RAG artifacts detected (nuance uses bypass pipeline)")
	} else {
		fmt.Println("   ✅ No RAG artifacts (pure LLM mode)")
	}

	if citations > 0 {
		fmt.Printf("   ⚠️  Citations: %d (not expected for nuance mode)\n", citations)
	} else {
		fmt.Println("   ✅ No citations (correct for nuance mode)")
	}

	// Check for nuance-specific indicators
	if len(indicators) > 0 {
		lowerReply := strings.ToLower(reply)
		found := 0
		for _, ind := range indicators {
			if strings.Contains(lowerReply, strings.ToLower(ind)) {
				found++
			}
		}
		if found > 0 {
			fmt.Printf("   ✅ Nuance indicators found: %d/%d keywords\n", found, len(indicators))
		} else {
			fmt.Println("   ℹ️  No specific indicators found (LLM may paraphrase)")
		}
	}

	// For concise mode, check response length
	if key == "concise" {
		wordCount := len(strings.Fields(reply))
		if wordCount < 100 {
			fmt.Printf("   ✅ Concise response: %d words\n", wordCount)
		} else {
			fmt.Printf("   ⚠️  Response may not be concise: %d words\n", wordCount)
		}
	}

	fmt.Println(strings.Repeat("─", 60))
}

func testSessionScopedNuance() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  SESSION-SCOPED NUANCE TEST                                                 ║")
	fmt.Println("║  Purpose: Verify nuance mode persists without prefix on follow-ups         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session: %s\n", sessionID)

	messages := []struct {
		Message   string
		HasPrefix bool
	}{
		{"/nuance:engineering What is dependency injection?", true},
		{"Can you give me an example?", false}, // Should still use engineering nuance
		{"What are the trade-offs?", false},    // Should still use engineering nuance
	}

	for i, msg := range messages {
		prefix := "(no prefix)"
		if msg.HasPrefix {
			prefix = "(with prefix)"
		}
		fmt.Printf("\n[Turn %d] %s: %s\n", i+1, prefix, msg.Message)

		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		reply, _ := sendChat(sessionID, msg.Message)
		fmt.Printf("AI: %s\n", truncate(reply, 150))

		// Check for RAG artifacts
		if hasRAGArtifacts(reply) {
			fmt.Println("   ⚠️  RAG artifact detected - mode may have leaked")
		} else {
			fmt.Println("   ✅ No RAG artifacts")
		}
	}

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("✅ SESSION-SCOPED NUANCE TEST COMPLETE")
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
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
	citations := 0

	if data, ok := res["data"].(map[string]interface{}); ok {
		if replyObj, ok := data["reply"].(map[string]interface{}); ok {
			if content, ok := replyObj["chat"].(string); ok {
				reply = content
			}
			if cits, ok := replyObj["citations"].([]interface{}); ok {
				citations = len(cits)
			}
		}
	}
	return reply, citations
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
