//go:build ignore

package main

import (
	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/pkg/embedding"
	"fmt"
	"log"
	"math"
)

// CosineSimilarity calculates similarity between two vectors
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func main() {
	cfg := config.Load()

	// 1. Initialize Providers
	fmt.Println("--- Initializing Providers ---")
	gemini := embedding.NewGeminiProvider(cfg.Keys.GoogleGemini)
	nomic := embedding.NewOllamaProvider(cfg.Ai.OllamaBaseURL, cfg.Ai.OllamaModel)

	// 2. Define Test Cases
	text1 := "The quick brown fox jumps over the lazy dog"      // Original
	text2 := "A fast brown fox leaps over a sleepy canine"      // Semantically similar
	text3 := "Quantum physics explores the nature of particles" // Completely different

	fmt.Println("\n--- Generating Embeddings ---")

	// Helper to generate and print info
	generate := func(name string, p embedding.EmbeddingProvider, t1, t2, t3 string) ([]float32, []float32, []float32) {
		fmt.Printf("\n[%s] Generating...\n", name)

		v1, err := p.Generate(t1, "RETRIEVAL_DOCUMENT")
		if err != nil {
			log.Printf("Error %s (Text 1): %v", name, err)
			return nil, nil, nil
		}
		fmt.Printf("[%s] Text 1 Dimensions: %d\n", name, len(v1.Embedding.Values))

		v2, err := p.Generate(t2, "RETRIEVAL_DOCUMENT")
		if err != nil {
			log.Printf("Error %s (Text 2): %v", name, err)
			return nil, nil, nil
		}

		v3, err := p.Generate(t3, "RETRIEVAL_DOCUMENT")
		if err != nil {
			log.Printf("Error %s (Text 3): %v", name, err)
			return nil, nil, nil
		}

		return v1.Embedding.Values, v2.Embedding.Values, v3.Embedding.Values
	}

	// 3. Run Gemini
	g1, g2, g3 := generate("GEMINI", gemini, text1, text2, text3)

	// 4. Run Nomic
	n1, n2, n3 := generate("NOMIC", nomic, text1, text2, text3)

	// 5. Compare Similarity
	fmt.Println("\n--- Semantic Similarity Comparison ---")
	fmt.Println("(Higher is better, 1.0 = identical)")

	if g1 != nil && g2 != nil && g3 != nil {
		fmt.Printf("\n[GEMINI] (3072 dims)\n")
		fmt.Printf("Similarity (Text 1 vs Text 2 - Similar): %.4f\n", CosineSimilarity(g1, g2))
		fmt.Printf("Similarity (Text 1 vs Text 3 - Different): %.4f\n", CosineSimilarity(g1, g3))
	}

	if n1 != nil && n2 != nil && n3 != nil {
		fmt.Printf("\n[NOMIC] (768 dims)\n")
		fmt.Printf("Similarity (Text 1 vs Text 2 - Similar): %.4f\n", CosineSimilarity(n1, n2))
		fmt.Printf("Similarity (Text 1 vs Text 3 - Different): %.4f\n", CosineSimilarity(n1, n3))
	}

	fmt.Println("\n--- Conclusion ---")
	fmt.Println("Check if Nomic correctly identifies Text 1 & 2 as more similar than Text 1 & 3.")
}
