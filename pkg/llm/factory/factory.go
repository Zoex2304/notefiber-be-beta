package factory

import (
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/llm/ollama"
	"fmt"
)

func NewLLMProvider(providerType, modelName, baseURL string) (llm.LLMProvider, error) {
	switch providerType {
	case "ollama":
		if baseURL == "" {
			baseURL = "http://localhost:11434" // Default
		}
		return ollama.NewOllamaProvider(baseURL, modelName), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerType)
	}
}
