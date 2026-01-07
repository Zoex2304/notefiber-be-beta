package factory

import (
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/llm/huggingface"
	"ai-notetaking-be/pkg/llm/ollama"
	"fmt"
)

func NewLLMProvider(providerType, modelName, baseURL, apiKey string) (llm.LLMProvider, error) {
	switch providerType {
	case "ollama":
		if baseURL == "" {
			baseURL = "http://localhost:11434" // Default
		}
		return ollama.NewOllamaProvider(baseURL, modelName), nil
	case "huggingface":
		if baseURL == "" {
			baseURL = "https://router.huggingface.co/v1" // Default
		}
		return huggingface.NewHuggingFaceProvider(apiKey, baseURL, modelName), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerType)
	}
}
