package embedding

type EmbeddingRequestContentPart struct {
	Text string `json:"text"`
}

type EmbeddingRequestContent struct {
	Parts []EmbeddingRequestContentPart `json:"parts"`
}

type EmbeddingRequest struct {
	Model    string                  `json:"model"`
	Content  EmbeddingRequestContent `json:"content"`
	TaskType string                  `json:"task_type,omitempty"`
}

type EmbeddingResponseEmbedding struct {
	Values []float32 `json:"values"`
}

type EmbeddingResponse struct {
	Embedding EmbeddingResponseEmbedding `json:"embedding"`
}

func GetGeminiEmbedding(
	apiKey string,
	text string,
	taskType string,
) (*EmbeddingResponse, error) {
	// Wrapper for backward compatibility using the new Provider
	provider := NewGeminiProvider(apiKey)
	return provider.Generate(text, taskType)
}
