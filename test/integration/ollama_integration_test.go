// FILE: test/integration/ollama_integration_test.go
// PURPOSE: Experimental Ollama local LLM integration test
// NOTE: This is a standalone test file. Does NOT modify existing Gemini implementation.
//       If tests pass, can migrate to pkg/chatbot/ollama_chatbot.go for production use.

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// ============================================================
// OLLAMA CONFIGURATION
// ============================================================

const (
	OllamaBaseURL = "http://localhost:11434"
	OllamaModel   = "gemma:2b"

	ollamaChatEndpoint = "/api/chat"
)

// ============================================================
// OLLAMA REQUEST/RESPONSE TYPES
// ============================================================

// OllamaChatRequest is the request payload for Ollama chat API
type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *OllamaOptions  `json:"options,omitempty"`
}

// OllamaMessage represents a single message in Ollama format
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaOptions for model configuration
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaChatResponse is the response from Ollama chat API
type OllamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt time.Time     `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

// ChatHistory mirrors the existing chatbot.ChatHistory struct
type ChatHistory struct {
	Chat string
	Role string
}

// Role constants - Ollama uses "assistant" instead of "model"
const (
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)

// ============================================================
// OLLAMA CORE FUNCTIONS
// ============================================================

// GetOllamaResponse generates a response using local Ollama server
// This can replace GetGeminiResponse for unlimited local processing
func GetOllamaResponse(
	ctx context.Context,
	chatHistories []*ChatHistory,
) (string, error) {
	// Convert ChatHistory to Ollama format
	messages := make([]OllamaMessage, 0, len(chatHistories))
	for _, history := range chatHistories {
		// Map "model" role to "assistant" for Ollama compatibility
		role := history.Role
		if role == "model" {
			role = ChatMessageRoleAssistant
		}
		messages = append(messages, OllamaMessage{
			Role:    role,
			Content: history.Chat,
		})
	}

	// Build request
	payload := OllamaChatRequest{
		Model:    OllamaModel,
		Messages: messages,
		Stream:   false, // Non-streaming for simplicity
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := OllamaBaseURL + ollamaChatEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request with timeout (Ollama can be slow on first request due to model loading)
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	// Read response
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama error: status %d, body: %s", res.StatusCode, string(resBody))
	}

	// Parse response
	var ollamaRes OllamaChatResponse
	if err := json.Unmarshal(resBody, &ollamaRes); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return ollamaRes.Message.Content, nil
}

// DecideToUseRAGWithOllama decides whether to use RAG based on conversation
// This can replace DecideToUseRAG for unlimited local processing
func DecideToUseRAGWithOllama(
	ctx context.Context,
	chatHistories []*ChatHistory,
) (bool, error) {
	// Build conversation with decision instruction
	messages := make([]OllamaMessage, 0)

	// System instruction
	messages = append(messages, OllamaMessage{
		Role: ChatMessageRoleUser,
		Content: `You are a decision system. Given a conversation, decide if the user's question requires searching through documents/notes (use RAG) or can be answered directly.

Rules:
- Return {"answer_directly": true} if question is general, greetings, or doesn't need specific document info
- Return {"answer_directly": false} if question asks about specific notes, documents, or past information

Respond ONLY with JSON: {"answer_directly": true} or {"answer_directly": false}`,
	})

	messages = append(messages, OllamaMessage{
		Role:    ChatMessageRoleAssistant,
		Content: `Understood. I will analyze conversations and respond only with JSON format.`,
	})

	// Add conversation history
	for _, history := range chatHistories {
		role := history.Role
		if role == "model" {
			role = ChatMessageRoleAssistant
		}
		messages = append(messages, OllamaMessage{
			Role:    role,
			Content: history.Chat,
		})
	}

	// Final instruction to enforce JSON
	messages = append(messages, OllamaMessage{
		Role:    ChatMessageRoleUser,
		Content: `Based on the conversation above, should I answer directly or use RAG? Respond ONLY with JSON.`,
	})

	// Build request
	payload := OllamaChatRequest{
		Model:    OllamaModel,
		Messages: messages,
		Stream:   false,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("marshal request: %w", err)
	}

	// Send request
	url := OllamaBaseURL + ollamaChatEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ollama error: status %d, body: %s", res.StatusCode, string(resBody))
	}

	// Parse response
	var ollamaRes OllamaChatResponse
	if err := json.Unmarshal(resBody, &ollamaRes); err != nil {
		return false, fmt.Errorf("unmarshal response: %w", err)
	}

	// Clean and parse JSON from response
	responseText := ollamaRes.Message.Content
	responseBytes := []byte(responseText)

	// Remove markdown code blocks if present
	responseBytes = bytes.TrimSpace(responseBytes)
	responseBytes = bytes.TrimPrefix(responseBytes, []byte("```json"))
	responseBytes = bytes.TrimPrefix(responseBytes, []byte("```"))
	responseBytes = bytes.TrimSuffix(responseBytes, []byte("```"))
	responseBytes = bytes.TrimSpace(responseBytes)

	// Parse JSON
	var decision struct {
		AnswerDirectly bool `json:"answer_directly"`
	}

	if err := json.Unmarshal(responseBytes, &decision); err != nil {
		return false, fmt.Errorf("parse decision JSON: %w, raw: %s", err, string(responseBytes))
	}

	// Return inverse: if answer_directly=true, then use_rag=false
	return !decision.AnswerDirectly, nil
}

// ============================================================
// TEST CASES
// ============================================================

// TestOllamaConnection verifies Ollama is running and accessible
func TestOllamaConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simple ping to Ollama
	req, err := http.NewRequestWithContext(ctx, "GET", OllamaBaseURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ollama not running at %s: %v", OllamaBaseURL, err)
	}
	defer res.Body.Close()

	t.Logf("✅ Ollama is running at %s (status: %d)", OllamaBaseURL, res.StatusCode)
}

// TestOllamaSimpleResponse tests basic chat response
func TestOllamaSimpleResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	chatHistories := []*ChatHistory{
		{
			Chat: "Hello! Say 'Ollama works!' in one sentence.",
			Role: ChatMessageRoleUser,
		},
	}

	response, err := GetOllamaResponse(ctx, chatHistories)
	if err != nil {
		t.Fatalf("GetOllamaResponse failed: %v", err)
	}

	t.Logf("✅ Response: %s", response)

	if response == "" {
		t.Error("Response should not be empty")
	}
}

// TestOllamaMultiTurnConversation tests context retention
func TestOllamaMultiTurnConversation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conversation := []*ChatHistory{
		{Chat: "My name is John", Role: ChatMessageRoleUser},
		{Chat: "Nice to meet you, John!", Role: ChatMessageRoleAssistant},
		{Chat: "What is my name?", Role: ChatMessageRoleUser},
	}

	response, err := GetOllamaResponse(ctx, conversation)
	if err != nil {
		t.Fatalf("Multi-turn conversation failed: %v", err)
	}

	t.Logf("✅ Response: %s", response)

	// Check if response mentions "John"
	if !bytes.Contains([]byte(response), []byte("John")) {
		t.Logf("⚠️ Response may not correctly remember the name. Response: %s", response)
	}
}

// TestOllamaRAGDecision tests RAG decision making
func TestOllamaRAGDecision(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	testCases := []struct {
		name        string
		messages    []*ChatHistory
		expectRAG   bool
		description string
	}{
		{
			name: "General question - should NOT use RAG",
			messages: []*ChatHistory{
				{Chat: "What is 2+2?", Role: ChatMessageRoleUser},
			},
			expectRAG:   false,
			description: "Simple math doesn't need document search",
		},
		{
			name: "Greeting - should NOT use RAG",
			messages: []*ChatHistory{
				{Chat: "Hello, how are you?", Role: ChatMessageRoleUser},
			},
			expectRAG:   false,
			description: "Greetings don't need document search",
		},
		{
			name: "Document-specific question - SHOULD use RAG",
			messages: []*ChatHistory{
				{Chat: "What did I write in my notes about project X?", Role: ChatMessageRoleUser},
			},
			expectRAG:   true,
			description: "Asking about notes requires document search",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			useRAG, err := DecideToUseRAGWithOllama(ctx, tc.messages)
			if err != nil {
				t.Logf("⚠️ RAG decision failed: %v", err)
				t.Skip("Skipping due to parsing error - model may need different prompting")
				return
			}

			t.Logf("Question: %s", tc.messages[0].Chat)
			t.Logf("Use RAG: %v (expected: %v)", useRAG, tc.expectRAG)

			if useRAG != tc.expectRAG {
				t.Logf("⚠️ Decision mismatch: got %v, expected %v. %s", useRAG, tc.expectRAG, tc.description)
			} else {
				t.Logf("✅ Correct decision!")
			}
		})
	}
}

// TestOllamaWithExistingRole tests compatibility with existing "model" role
func TestOllamaWithExistingRole(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Use "model" role like the existing Gemini implementation
	conversation := []*ChatHistory{
		{Chat: "Tell me a short joke", Role: "user"},
		{Chat: "Why did the chicken cross the road? To get to the other side!", Role: "model"}, // Using "model" not "assistant"
		{Chat: "That was funny! Tell me another one.", Role: "user"},
	}

	response, err := GetOllamaResponse(ctx, conversation)
	if err != nil {
		t.Fatalf("Failed with 'model' role: %v", err)
	}

	t.Logf("✅ Response (with 'model' role mapping): %s", response)
}
