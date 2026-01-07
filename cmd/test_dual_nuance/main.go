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
	fmt.Println("║                 DUAL-MODE NUANCE TEST                                        ║")
	fmt.Println("║   Testing: /bypass/nuance:key (bypass+nuance)                               ║")
	fmt.Println("║            /nuance:key (RAG+nuance)                                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Test 1: Bypass + Nuance (engineering)
	testBypassNuance()

	// Test 2: RAG + Nuance (teacher with English exam notes)
	testRAGNuance()

	// Test 3: Session-scoped behavior
	testSessionScoped()
}

func testBypassNuance() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TEST 1: BYPASS + NUANCE (/bypass/nuance:engineering)                       ║")
	fmt.Println("║  Expected: Pure LLM with engineering style, NO RAG artifacts                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session: %s\n", sessionID)

	query := "/bypass/nuance:engineering What is dependency injection?"
	fmt.Printf("\n📤 USER: %s\n", query)

	reply, citations := sendChat(sessionID, query)
	fmt.Printf("\n🤖 AI: %s\n", reply)

	// Analysis
	fmt.Println("\n📊 ANALYSIS:")
	fmt.Println(strings.Repeat("─", 60))

	if hasRAGArtifacts(reply) {
		fmt.Println("   ❌ RAG artifacts detected (FAIL - should be pure LLM)")
	} else {
		fmt.Println("   ✅ No RAG artifacts (correct for bypass+nuance)")
	}

	if citations > 0 {
		fmt.Printf("   ❌ Citations: %d (FAIL - bypass should have none)\n", citations)
	} else {
		fmt.Println("   ✅ No citations (correct for bypass mode)")
	}

	// Check for engineering indicators
	indicators := []string{"injection", "dependency", "design", "interface", "coupling", "abstraction"}
	lowerReply := strings.ToLower(reply)
	found := 0
	for _, ind := range indicators {
		if strings.Contains(lowerReply, ind) {
			found++
		}
	}
	if found > 0 {
		fmt.Printf("   ✅ Technical terms found: %d/6\n", found)
	}
	fmt.Println(strings.Repeat("─", 60))
}

func testRAGNuance() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TEST 2: RAG + NUANCE (/nuance:teacher)                                     ║")
	fmt.Println("║  Expected: Search notes + respond as teacher, MAY have citations           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session: %s\n", sessionID)

	query := "/nuance:teacher Tell me about the English exam"
	fmt.Printf("\n📤 USER: %s\n", query)

	reply, citations := sendChat(sessionID, query)
	fmt.Printf("\n🤖 AI: %s\n", reply)

	// Analysis
	fmt.Println("\n📊 ANALYSIS:")
	fmt.Println(strings.Repeat("─", 60))

	// RAG+nuance CAN have references since it uses RAG pipeline
	if citations > 0 {
		fmt.Printf("   📎 Citations: %d (expected for RAG mode)\n", citations)
	} else {
		fmt.Println("   ℹ️  No citations (might be answering from context)")
	}

	// Check for teacher-like language
	teacherWords := []string{"student", "exam", "practice", "understand", "learn", "study", "explain", "prepare", "correct", "answer"}
	lowerReply := strings.ToLower(reply)
	found := 0
	for _, w := range teacherWords {
		if strings.Contains(lowerReply, w) {
			found++
		}
	}
	if found >= 2 {
		fmt.Printf("   ✅ Teacher-like language detected: %d/10 keywords\n", found)
	} else {
		fmt.Printf("   ℹ️  Teacher-like language: %d/10 keywords\n", found)
	}
	fmt.Println(strings.Repeat("─", 60))
}

func testSessionScoped() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TEST 3: SESSION-SCOPED NUANCE                                              ║")
	fmt.Println("║  Expected: First message sets mode, follow-ups stay in mode                ║")
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
		{"/bypass/nuance:engineering What is dependency injection?", true},
		{"Can you give me an example?", false},
		{"What are the trade-offs?", false},
	}

	allBypass := true
	for i, msg := range messages {
		prefix := "(no prefix)"
		if msg.HasPrefix {
			prefix = "(with prefix)"
		}
		fmt.Printf("\n[Turn %d] %s: %s\n", i+1, prefix, msg.Message)

		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		reply, citations := sendChat(sessionID, msg.Message)
		fmt.Printf("AI: %s\n", truncate(reply, 120))

		if hasRAGArtifacts(reply) || citations > 0 {
			fmt.Println("   ⚠️  Mode leaked to RAG!")
			allBypass = false
		} else {
			fmt.Println("   ✅ Still in bypass+nuance mode")
		}
	}

	fmt.Println("\n" + strings.Repeat("─", 60))
	if allBypass {
		fmt.Println("✅ SESSION-SCOPED NUANCE VERIFIED: All turns stayed in bypass+nuance mode")
	} else {
		fmt.Println("⚠️  Some turns may have leaked to RAG mode")
	}
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
