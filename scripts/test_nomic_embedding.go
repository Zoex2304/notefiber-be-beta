//go:build ignore

package main

import (
	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/pkg/embedding"
	"fmt"
	"log"
)

func main() {
	// 1. Load Config
	cfg := config.Load()
	fmt.Printf("Loaded Config > Embedding Provider: %s\n", cfg.Ai.EmbeddingProvider)
	fmt.Printf("Loaded Config > Ollama URL: %s\n", cfg.Ai.OllamaBaseURL)
	fmt.Printf("Loaded Config > Ollama Model: %s\n", cfg.Ai.OllamaModel)

	// 2. Initialize Ollama Provider explicitly for testing (ignoring main provider for now)
	provider := embedding.NewOllamaProvider(cfg.Ai.OllamaBaseURL, cfg.Ai.OllamaModel)

	// 3. Test Text
	text := "The quick brown fox jumps over the lazy dog."
	fmt.Printf("\nGenerating embedding for: \"%s\"\n", text)

	// 4. Generate
	resp, err := provider.Generate(text, "RETRIEVAL_QUERY")
	if err != nil {
		log.Fatalf("Error generating embedding: %v", err)
	}

	// 5. Inspect Result
	dims := len(resp.Embedding.Values)
	fmt.Printf("Success! Generated Embedding Dimensions: %d\n", dims)

	if dims > 5 {
		fmt.Printf("First 5 values: %v...\n", resp.Embedding.Values[:5])
	}

	// 6. Validation
	// nomic-embed-text should be 768 dimensions
	if dims == 768 {
		fmt.Println("✅ Dimensions match expected Nomic output (768).")
	} else {
		fmt.Printf("⚠️  Dimensions %d do NOT match expected 768 for nomic-embed-text. (Is it a different model?)\n", dims)
	}
}
