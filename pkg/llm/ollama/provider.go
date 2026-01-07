package ollama

import (
	"ai-notetaking-be/pkg/llm"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	BaseURL   string
	ModelName string
	Client    *http.Client
}

// Ensure OllamaProvider implements LLMProvider
var _ llm.LLMProvider = &OllamaProvider{}

func NewOllamaProvider(baseURL, modelName string) *OllamaProvider {
	return &OllamaProvider{
		BaseURL:   baseURL,
		ModelName: modelName,
		Client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// --- Request/Response structs (Internal to this package) ---

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaChatResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

// --- Interface Implementation ---

func (o *OllamaProvider) Chat(ctx context.Context, history []llm.Message, opts ...llm.Option) (string, error) {
	// 1. Process Options
	options := &llm.Options{
		Temperature: 0.7, // Default
	}
	for _, opt := range opts {
		opt(options)
	}

	// 2. Map generic messages to Ollama messages
	ollamaMessages := make([]ollamaMessage, len(history))
	for i, msg := range history {
		role := msg.Role
		// Map standard roles if necessary, though "user", "assistant", "system" are standard
		if role == "model" {
			role = "assistant"
		}
		ollamaMessages[i] = ollamaMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	// 3. Prepare Payload
	model := o.ModelName
	if options.Model != "" {
		model = options.Model
	}

	reqPayload := ollamaChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   false,
		Options: &ollamaOptions{
			Temperature: options.Temperature,
		},
	}

	if options.MaxTokens > 0 {
		reqPayload.Options.NumPredict = options.MaxTokens
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// 4. Send Request
	url := o.BaseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// 5. Parse Response
	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}

func (o *OllamaProvider) Generate(ctx context.Context, prompt string, opts ...llm.Option) (string, error) {
	// Reuse Chat for simplicity as most new LLMs are chat-optimized
	return o.Chat(ctx, []llm.Message{{Role: "user", Content: prompt}}, opts...)
}
