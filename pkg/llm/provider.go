package llm

import (
	"context"
)

// Message represents a chat message in a provider-agnostic format
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
}

// Option allows for optional parameters like Temperature, MaxTokens, etc.
type Option func(*Options)

type Options struct {
	Temperature float64
	MaxTokens   int
	Model       string // Override default model
}

func WithTemperature(temp float64) Option {
	return func(o *Options) {
		o.Temperature = temp
	}
}

func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// LLMProvider defines the contract for any LLM backend
type LLMProvider interface {
	// Chat sends a chat history to the model and returns the response
	Chat(ctx context.Context, history []Message, options ...Option) (string, error)

	// Generate sends a single prompt to the model (convenience method)
	Generate(ctx context.Context, prompt string, options ...Option) (string, error)
}
