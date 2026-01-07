package jina

import (
	"ai-notetaking-be/pkg/embedding"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type JinaProvider struct {
	apiKey  string
	baseURL string
	model   string
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewJinaProvider(apiKey string) *JinaProvider {
	return &JinaProvider{
		apiKey:  apiKey,
		baseURL: "https://api.jina.ai/v1/embeddings",
		model:   "jina-embeddings-v2-base-en",
	}
}

func (p *JinaProvider) Generate(text string, taskType string) (*embedding.EmbeddingResponse, error) {
	// Jina docs recommend array of inputs. We wrap single text.
	reqBody := embeddingRequest{
		Model: p.model,
		Input: []string{text},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jina api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var jinaResp embeddingResponse
	if err := json.Unmarshal(bodyBytes, &jinaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if jinaResp.Error != nil {
		return nil, fmt.Errorf("jina api returned error: %s", jinaResp.Error.Message)
	}

	if len(jinaResp.Data) == 0 {
		return nil, fmt.Errorf("empty embeddings from jina api")
	}

	// Jina returns 768 dimensions for v2-base-en
	// We map it to the application's EmbeddingResponse format
	// Note: pkg/embedding packages might differ slightly in structure,
	// assuming gemini_embedding.go defines the standard.

	return &embedding.EmbeddingResponse{
		Embedding: embedding.EmbeddingResponseEmbedding{
			Values: jinaResp.Data[0].Embedding,
		},
	}, nil
}
