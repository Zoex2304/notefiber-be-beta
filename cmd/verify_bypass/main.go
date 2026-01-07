package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	apiBaseURL = "http://localhost:3000/api"
	// Token for user zikri (same as previous scripts)
	authToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0NTQxMjYsInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.dbazacYBZiUusWzHSehKGTexYRxu8bwTKr-z_l8z5Eo"
)

func main() {
	fmt.Println("🔍 VERIFYING BYPASS FIX (CLEAN SESSION)")
	fmt.Println("--------------------------------------------------")

	// 1. Create Session
	fmt.Println("[STEP 1] Creating Fresh Session...")
	sessionID := createSession()
	fmt.Printf("✅ Session Created: %s\n", sessionID)

	// 2. Send Bypass Chat
	query := "What is the capital of China?"
	fmt.Printf("\n[STEP 2] Sending Bypass Query: '/bypass %s'\n", query)

	reply, citations := sendChat(sessionID, "/bypass "+query)

	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("🤖 AI Reply:\n%s\n", reply)
	fmt.Println("--------------------------------------------------")

	if len(citations) == 0 {
		fmt.Println("✅ RESULT: No citations found. (PURE LLM CONFIRMED)")
	} else {
		fmt.Printf("❌ RESULT: FAILED! %d citations found. (RAG STILL ACTIVE)\n", len(citations))
		for i, c := range citations {
			fmt.Printf("   [%d] %v\n", i, c)
		}
	}
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
		fmt.Printf("Failed to create session: %s\n", string(body))
		os.Exit(1)
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

func sendChat(sessionID, message string) (string, []interface{}) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	resp, body := doRequest("POST", "/chatbot/v1/send-chat", payload)
	if resp.StatusCode != 200 {
		fmt.Printf("Failed to send chat: %s\n", string(body))
		os.Exit(1)
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	reply := ""
	var citations []interface{}

	if data, ok := res["data"].(map[string]interface{}); ok {
		if r, ok := data["reply"].(string); ok {
			reply = r
		} // Changed: Use top-level reply field from data object
		// Correct check based on DTO structure (data -> reply object -> citations?? No, wait.)
		// Chatbot controller: Returns &dto.SendChatResponse{ Sent: ..., Reply: ... }
		// Reply is of type *dto.SendChatResponseChat which has Citations field.

		if replyObj, ok := data["reply"].(map[string]interface{}); ok {
			// If reply is an object (which it is in the updated controller output)
			if content, ok := replyObj["chat"].(string); ok {
				reply = content
			}
			if c, ok := replyObj["citations"].([]interface{}); ok {
				citations = c
			}
		} else {
			// Older format or if controller returns flat reply?
			// Looking at chatbot_service.go:330: Reply: &dto.SendChatResponseChat{ ... Citations: ... }
			// So data["reply"] is an OBJECT.
		}
	}
	return reply, citations
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Network Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}
