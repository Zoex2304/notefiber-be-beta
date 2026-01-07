package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
)

// OllamaProvider implements EmbeddingProvider for local Ollama models (e.g., nomic-embed-text)
type OllamaProvider struct {
	BaseURL string
	Model   string
}

func NewOllamaProvider(baseURL string, model string) EmbeddingProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaProvider{
		BaseURL: baseURL,
		Model:   model,
	}
}

// Ollama Embedding Request/Response structures
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"` // Ollama returns float64 usually
}

func (p *OllamaProvider) Generate(text string, taskType string) (*EmbeddingResponse, error) {
	// TaskType is ignored for Nomic/Ollama usually, but kept for interface compatibility

	reqBody := ollamaEmbeddingRequest{
		Model:  p.Model,
		Prompt: text,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/embeddings", p.BaseURL)
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding error: %s", string(bodyBytes))
	}

	var ollamaResp ollamaEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return nil, err
	}

	// Convert float64 to float32 for compatibility with our system
	values := make([]float32, len(ollamaResp.Embedding))
	for i, v := range ollamaResp.Embedding {
		values[i] = float32(v)
	}

	// CRITICAL: Normalize the vector for accurate cosine similarity
	// Cosine distance in pgvector requires normalized vectors (magnitude = 1)
	normalizedValues := normalizeVector(values)

	return &EmbeddingResponse{
		Embedding: EmbeddingResponseEmbedding{
			Values: normalizedValues,
		},
	}, nil
}

// normalizeVector normalizes a vector to unit length (magnitude = 1)
// This is REQUIRED for accurate cosine similarity calculation
func normalizeVector(vec []float32) []float32 {
	var magnitude float64
	for _, v := range vec {
		magnitude += float64(v) * float64(v)
	}
	magnitude = math.Sqrt(magnitude)

	// Avoid division by zero
	if magnitude == 0 {
		return vec
	}

	normalized := make([]float32, len(vec))
	for i, v := range vec {
		normalized[i] = float32(float64(v) / magnitude)
	}
	return normalized
}
