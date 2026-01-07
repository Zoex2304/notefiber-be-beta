// FILE: pkg/rag/intent.go
// PURPOSE: Detect user intent to decide if RAG is needed

package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/entity"
)

// Intent represents the detected user intent
type Intent struct {
	Type    string `json:"intent"`   // follow_up, clarification, new_query, confirmation
	NeedRAG bool   `json:"need_rag"` // true if RAG retrieval is needed
	Reason  string `json:"reason"`   // explanation for the decision
}

// DetectIntent analyzes conversation history and new message to determine if RAG is needed
func DetectIntent(ctx context.Context, history []*entity.ChatMessageRaw, newMessage string) (*Intent, error) {
	// If no history (first message), always need RAG
	if len(history) <= 2 { // Only system prompt + greeting
		return &Intent{
			Type:    "new_query",
			NeedRAG: true,
			Reason:  "First user message in session",
		}, nil
	}

	// Build conversation history string for prompt
	historyStr := buildHistoryString(history)

	// Create intent detection prompt
	prompt := fmt.Sprintf(constant.IntentDetectionPrompt, historyStr, newMessage)

	// Call Ollama for intent detection
	intent, err := callOllamaForIntent(ctx, prompt)
	if err != nil {
		// On error, default to RAG (safer)
		return &Intent{
			Type:    "new_query",
			NeedRAG: true,
			Reason:  "Intent detection failed, defaulting to RAG",
		}, nil
	}

	return intent, nil
}

// buildHistoryString creates a readable history for the intent prompt
func buildHistoryString(history []*entity.ChatMessageRaw) string {
	var sb strings.Builder

	for i, msg := range history {
		// Skip system prompt (index 0) and greeting (index 1)
		if i <= 1 {
			continue
		}

		role := "User"
		if msg.Role == "model" || msg.Role == "assistant" {
			role = "Assistant"
		}

		// Truncate long messages for prompt efficiency
		chat := msg.Chat
		if len(chat) > 200 {
			chat = chat[:200] + "..."
		}

		sb.WriteString(fmt.Sprintf("%s: %s\n", role, chat))
	}

	return sb.String()
}

// callOllamaForIntent calls Ollama to detect intent
func callOllamaForIntent(ctx context.Context, prompt string) (*Intent, error) {
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}

	payload := map[string]interface{}{
		"model":    getOllamaModel(),
		"messages": messages,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": 0.1, // Low for consistent classification
			"num_predict": 150,
		},
	}

	payloadJSON, _ := json.Marshal(payload)

	url := getOllamaBaseURL() + "/api/chat"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	// Parse Ollama response
	var ollamaRes struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &ollamaRes); err != nil {
		return nil, err
	}

	return parseIntentResponse(ollamaRes.Message.Content)
}

// parseIntentResponse extracts intent from LLM response
func parseIntentResponse(response string) (*Intent, error) {
	// Clean response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to extract JSON from response (might be wrapped in text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		response = response[jsonStart : jsonEnd+1]
	}

	var intent Intent
	if err := json.Unmarshal([]byte(response), &intent); err == nil {
		return &intent, nil
	}

	// JSON parse failed - use keyword-based fallback detection
	responseLower := strings.ToLower(response)

	// Check for follow-up indicators
	if containsAny(responseLower, []string{"follow_up", "follow up", "followup"}) ||
		containsAny(responseLower, []string{"need_rag\": false", "need_rag\":false", "\"need_rag\": false"}) {
		return &Intent{
			Type:    "follow_up",
			NeedRAG: false,
			Reason:  "keyword detection: follow_up",
		}, nil
	}

	// Check for clarification indicators
	if containsAny(responseLower, []string{"clarification", "clarify"}) {
		return &Intent{
			Type:    "clarification",
			NeedRAG: false,
			Reason:  "keyword detection: clarification",
		}, nil
	}

	// Check for confirmation indicators
	if containsAny(responseLower, []string{"confirmation", "confirm"}) {
		return &Intent{
			Type:    "confirmation",
			NeedRAG: false,
			Reason:  "keyword detection: confirmation",
		}, nil
	}

	// Default to RAG if we can't determine
	return &Intent{
		Type:    "new_query",
		NeedRAG: true,
		Reason:  "parse failed, default to RAG",
	}, nil
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
