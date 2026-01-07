package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Configuration struct {
	APIBaseURL string
	AuthToken  string
	Timeout    time.Duration
}

type ChatClient struct {
	config Configuration
	client *http.Client
}

func NewChatClient(config Configuration) *ChatClient {
	return &ChatClient{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

func (c *ChatClient) CreateSession() (string, error) {
	resp, body, err := c.doRequest("POST", "/chatbot/v1/create-session", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result["data"].(map[string]interface{})["id"].(string), nil
}

func (c *ChatClient) SendMessage(sessionID, message string) (*ChatResponse, error) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}
	resp, body, err := c.doRequest("POST", "/chatbot/v1/send-chat", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %s", string(body))
	}
	return c.parseChatResponse(body)
}

func (c *ChatClient) doRequest(method, endpoint string, body interface{}) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, c.config.APIBaseURL+endpoint, bodyReader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.AuthToken)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	return resp, respBody, err
}

func (c *ChatClient) parseChatResponse(body []byte) (*ChatResponse, error) {
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	data := result["data"].(map[string]interface{})
	replyObj := data["reply"].(map[string]interface{})

	resp := &ChatResponse{}
	if content, ok := replyObj["chat"].(string); ok {
		resp.Content = content
	}
	if citations, ok := replyObj["citations"].([]interface{}); ok {
		resp.CitationCount = len(citations)
	}
	return resp, nil
}

type ChatResponse struct {
	Content       string
	CitationCount int
	Duration      time.Duration
}

func main() {
	config := Configuration{
		APIBaseURL: "http://localhost:3000/api",
		AuthToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0OTAxNTksInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.jaUJYwutyRYvuv_G6zYnbjWuoDdaHcQb8VgYEhVRDpQ",
		Timeout:    180 * time.Second,
	}
	client := NewChatClient(config)

	// 1. Create Session
	sessionID, err := client.CreateSession()
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ Session Created: %s\n\n", sessionID)

	// 2. English Exam
	printStep("1. Focusing on English Exam")
	res1, _ := client.SendMessage(sessionID, "answer my english exam")
	printResponse(res1)

	printStep("2. Selecting Exam")
	res2, _ := client.SendMessage(sessionID, "Please provide a complete answer for English Final Examination")
	printResponse(res2)

	// 3. Switch Topic (Test Search Intent)
	printStep("3. Switching Topic to Class Fund")
	res3, _ := client.SendMessage(sessionID, "Calculate how many students have not paid")
	printResponse(res3)

	// 4. Persistence Check (Implicit)
	// If the server was restarted here, the next message would fail without persistence.
	// But we can't restart server from here.
	// We assume if Step 3 worked, and we continue, it shows flow.

	printStep("4. Back to English Exam (Zigzag)")
	res4, _ := client.SendMessage(sessionID, "What was the answer to question 5 again?")
	printResponse(res4)
}

func printStep(s string) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println(s)
	fmt.Println(strings.Repeat("=", 50))
}

func printResponse(response *ChatResponse) {
	if response == nil {
		return
	}
	fmt.Printf("🤖 AI: %s\n", response.Content)
	fmt.Printf("📎 Citations: %d\n", response.CitationCount)
}
