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
	fmt.Println("║               LEXICAL PARSER RAG VERIFICATION                               ║")
	fmt.Println("║   Testing: Table interpretation & Sequential Context                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Create Session
	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session Created: %s\n", sessionID)

	// 2. Sequential Queries
	queries := []string{
		"Calculate how many students have not paid.",
		"What is the total amount of money that has not yet been paid?",
		"Compute the overall aggregate amount.",
		"Which student has the highest unpaid amount?",
	}

	for i, q := range queries {
		fmt.Println("\n" + strings.Repeat("─", 80))
		fmt.Printf("QUERY %d: %s\n", i+1, q)
		fmt.Println(strings.Repeat("─", 80))

		start := time.Now()
		reply, citations := sendChat(sessionID, q)
		duration := time.Since(start)

		fmt.Printf("🤖 AI (%v): %s\n", duration, truncate(reply, 500))
		if citations > 0 {
			fmt.Printf("📎 Citations: %d\n", citations)
		} else {
			fmt.Println("⚠️  No citations (Pure LLM?)")
		}

		// Optional: Pause purely for readability of output stream if needed,
		// but system is synchronous so not required for correctness.
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n✅ SEQUENCE COMPLETE")
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
