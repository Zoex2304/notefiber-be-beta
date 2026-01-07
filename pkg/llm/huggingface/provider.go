package huggingface

import (
	"ai-notetaking-be/pkg/llm"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HuggingFaceProvider struct {
	apiKey  string
	baseURL string
	model   string
}

// Request Payload Structure (OpenAI Compatible)
type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []llm.Message `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewHuggingFaceProvider(apiKey, baseURL, model string) *HuggingFaceProvider {
	if baseURL == "" {
		baseURL = "https://router.huggingface.co/v1" // Default Router URL
	}
	return &HuggingFaceProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

func (p *HuggingFaceProvider) Chat(ctx context.Context, history []llm.Message, options ...llm.Option) (string, error) {
	opts := &llm.Options{
		Model:     p.model,
		MaxTokens: 500, // Default sane limit
	}
	for _, o := range options {
		o(opts)
	}

	reqBody := chatRequest{
		Model:     opts.Model,
		Messages:  history,
		MaxTokens: opts.MaxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("huggingface api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("huggingface api returned error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices from huggingface api")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (p *HuggingFaceProvider) Generate(ctx context.Context, prompt string, options ...llm.Option) (string, error) {
	// Wrap single prompt into a user message
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}
	return p.Chat(ctx, messages, options...)
}
