package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	baseURL     = "http://localhost:3000/api/chatbot/v1"
	accessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjcxNDE2NDgsInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.7jtmgE319K5yQMrw4ABS10GB7Ltc4XDp2LRMZjUjq2k"
)

// Simplified DTOs for the script
type CreateSessionResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type SendChatRequest struct {
	ChatSessionID string `json:"chat_session_id"`
	Chat          string `json:"chat"`
}

type SendChatResponse struct {
	Data struct {
		Reply struct {
			Chat string `json:"chat"`
		} `json:"reply"`
	} `json:"data"`
}

func main() {
	fmt.Println("=== Reactive RAG Simulation Client ===")
	fmt.Println("Connecting as User: a2b94f4c-b674-433b-90be-65a91a37e7a3")

	sessionID, err := createSession()
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	fmt.Printf("Session Created: %s\n", sessionID)

	testCases := []string{
		"find english exam",
	}

	for _, text := range testCases {
		fmt.Printf("\nUSER: %s\n", text)

		start := time.Now()
		reply, err := sendChat(sessionID, text)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("AI (%v): %s\n", elapsed, reply)
		}

		// Small delay to allow async logs to flush on server side (optional)
		time.Sleep(1 * time.Second)
	}
}

func createSession() (string, error) {
	req, _ := http.NewRequest("POST", baseURL+"/create-session", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API Error %d: %s", resp.StatusCode, string(body))
	}

	var res CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Data.ID, nil
}

func sendChat(sessionID, text string) (string, error) {
	payload := SendChatRequest{
		ChatSessionID: sessionID,
		Chat:          text,
	}
	jsonBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/send-chat", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API Error %d: %s", resp.StatusCode, string(body))
	}

	var res SendChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Data.Reply.Chat, nil
}
