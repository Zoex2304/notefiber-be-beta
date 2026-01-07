package embedding

// EmbeddingProvider defines the interface for generating text embeddings
type EmbeddingProvider interface {
	Generate(text string, taskType string) (*EmbeddingResponse, error)
}
