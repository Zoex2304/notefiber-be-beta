// FILE: pkg/chatbot/ollama_chatbot.go
// PURPOSE: Ollama local LLM integration for unlimited AI processing
// NOTE: This file complements gemini_chatbot.go - does NOT replace it.
//       Embeddings still use Gemini. Chat responses use Ollama.

package chatbot

import (
	"ai-notetaking-be/internal/constant"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ============================================================
// OLLAMA CONFIGURATION
// ============================================================

// GetOllamaBaseURL returns the Ollama server URL from env or default
func GetOllamaBaseURL() string {
	url := os.Getenv("OLLAMA_BASE_URL")
	if url == "" {
		return constant.OllamaDefaultBaseURL
	}
	return url
}

// GetOllamaModel returns the model from env or default
func GetOllamaModel() string {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		return constant.OllamaDefaultModel
	}
	return model
}

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

// ============================================================
// OLLAMA CORE FUNCTIONS
// ============================================================

// GetOllamaResponse generates a response using local Ollama server
// This replaces GetGeminiResponse for unlimited local processing
// No API key required!
func GetOllamaResponse(
	ctx context.Context,
	chatHistories []*ChatHistory,
) (string, error) {
	// Convert ChatHistory to Ollama format
	messages := make([]OllamaMessage, 0, len(chatHistories))
	for _, history := range chatHistories {
		// Map "model" role to "assistant" for Ollama compatibility
		role := history.Role
		if role == constant.ChatMessageRoleModel {
			role = constant.OllamaRoleAssistant
		}
		messages = append(messages, OllamaMessage{
			Role:    role,
			Content: history.Chat,
		})
	}

	// Build request
	payload := OllamaChatRequest{
		Model:    GetOllamaModel(),
		Messages: messages,
		Stream:   false, // Non-streaming for simplicity
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := GetOllamaBaseURL() + constant.OllamaChatEndpoint
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
		return "", fmt.Errorf("ollama request failed: %w", err)
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
// This replaces DecideToUseRAG for unlimited local processing
func DecideToUseRAGWithOllama(
	ctx context.Context,
	chatHistories []*ChatHistory,
) (bool, error) {
	// Build conversation with decision instruction
	messages := make([]OllamaMessage, 0)

	// System instruction
	messages = append(messages, OllamaMessage{
		Role:    constant.OllamaRoleUser,
		Content: constant.OllamaRAGDecisionSystemPrompt,
	})

	messages = append(messages, OllamaMessage{
		Role:    constant.OllamaRoleAssistant,
		Content: constant.OllamaRAGDecisionAckPrompt,
	})

	// Add conversation history
	for _, history := range chatHistories {
		role := history.Role
		if role == constant.ChatMessageRoleModel {
			role = constant.OllamaRoleAssistant
		}
		messages = append(messages, OllamaMessage{
			Role:    role,
			Content: history.Chat,
		})
	}

	// Final instruction to enforce JSON
	messages = append(messages, OllamaMessage{
		Role:    constant.OllamaRoleUser,
		Content: constant.OllamaRAGDecisionFinalPrompt,
	})

	// Build request
	payload := OllamaChatRequest{
		Model:    GetOllamaModel(),
		Messages: messages,
		Stream:   false,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("marshal request: %w", err)
	}

	// Send request
	url := GetOllamaBaseURL() + constant.OllamaChatEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("ollama request failed: %w", err)
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
