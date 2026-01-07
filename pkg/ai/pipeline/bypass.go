package pipeline

import (
	"context"
	"log"

	"ai-notetaking-be/pkg/llm"
)

// NuanceConfig holds injected nuance configuration
type NuanceConfig struct {
	Key           string
	Name          string
	SystemPrompt  string  // Injected as system message
	ModelOverride *string // Optional: use different model for this nuance
}

// BypassResult contains the result of bypass execution
type BypassResult struct {
	Reply string
}

// BypassPipeline executes pure LLM without RAG
// This preserves conversation history while bypassing note retrieval
type BypassPipeline struct {
	llmProvider llm.LLMProvider
	logger      *log.Logger
}

// NewBypassPipeline creates a new bypass pipeline
func NewBypassPipeline(llmProvider llm.LLMProvider, logger *log.Logger) *BypassPipeline {
	return &BypassPipeline{
		llmProvider: llmProvider,
		logger:      logger,
	}
}

// Execute runs pure LLM with conversation history
// This is used when user explicitly requests /bypass mode
func (p *BypassPipeline) Execute(
	ctx context.Context,
	query string,
	history []llm.Message,
	nuance *NuanceConfig,
) (*BypassResult, error) {

	var messages []llm.Message

	// Inject nuance system prompt if provided
	if nuance != nil && nuance.SystemPrompt != "" {
		p.logger.Printf("[BYPASS] Injecting nuance: %s", nuance.Key)
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: nuance.SystemPrompt,
		})
	}

	// Append conversation history
	messages = append(messages, history...)

	// Append current query
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: query,
	})

	p.logger.Printf("[BYPASS] Executing with %d messages (incl. history)", len(messages))

	// Prepare options
	var opts []llm.Option
	if nuance != nil && nuance.ModelOverride != nil && *nuance.ModelOverride != "" {
		p.logger.Printf("[BYPASS] Overriding model: %s", *nuance.ModelOverride)
		opts = append(opts, llm.WithModel(*nuance.ModelOverride))
	}

	// Call LLM directly
	response, err := p.llmProvider.Chat(ctx, messages, opts...)
	if err != nil {
		p.logger.Printf("[BYPASS] LLM error: %v", err)
		return nil, err
	}

	p.logger.Printf("[BYPASS] Response generated successfully")

	return &BypassResult{
		Reply: response,
	}, nil
}
