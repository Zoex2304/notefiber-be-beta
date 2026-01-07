package response

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/store"
)

// AdaptiveMessenger generates context-aware, language-adaptive messages
// This replaces hardcoded Indonesian strings with LLM-generated responses
type AdaptiveMessenger struct {
	llmProvider llm.LLMProvider
	logger      *log.Logger
}

// NewAdaptiveMessenger creates a new adaptive messenger
func NewAdaptiveMessenger(llmProvider llm.LLMProvider, logger *log.Logger) *AdaptiveMessenger {
	return &AdaptiveMessenger{
		llmProvider: llmProvider,
		logger:      logger,
	}
}

// AmbiguityContext contains information about the ambiguous situation
type AmbiguityContext struct {
	Query      string
	Candidates []store.Document
	UserLang   string // Detected from history, empty means auto-detect
}

// GenerateAmbiguityMessage creates a clarification message when multiple notes match
// The message is generated in the user's language and presents clear options
func (m *AdaptiveMessenger) GenerateAmbiguityMessage(
	ctx context.Context,
	ambCtx *AmbiguityContext,
	history []llm.Message,
) string {
	// Build candidate list
	var candidateList strings.Builder
	for i, c := range ambCtx.Candidates {
		candidateList.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}

	prompt := fmt.Sprintf(`<task>
Generate a brief, helpful clarification message for the user.

CONTEXT:
- User searched for: "%s"
- Found %d relevant notes (listed below)
- Need user to pick one OR confirm they want info from all

NOTES FOUND:
%s

REQUIREMENTS:
1. Match the language the user used in their query
2. Be concise (2-3 sentences max before the list)
3. Present the numbered list clearly
4. Offer option to get info from "all" if they want

OUTPUT FORMAT:
[Your message in user's language]
1. [Note title 1]
2. [Note title 2]
...

Which one would you like to focus on, or should I summarize all of them?
</task>

Generate the message now:`,
		ambCtx.Query,
		len(ambCtx.Candidates),
		candidateList.String(),
	)

	// Add history context for language detection
	messages := append(history, llm.Message{Role: "user", Content: prompt})

	response, err := m.llmProvider.Chat(ctx, messages)
	if err != nil {
		m.logger.Printf("[WARN] Adaptive message generation failed: %v", err)
		return m.fallbackAmbiguityMessage(ambCtx.Candidates)
	}

	return response
}

// GenerateNotFoundMessage creates a message when no notes match
func (m *AdaptiveMessenger) GenerateNotFoundMessage(
	ctx context.Context,
	query string,
	history []llm.Message,
) string {
	prompt := fmt.Sprintf(`<task>
Generate a brief, helpful "not found" message.

CONTEXT:
- User searched for: "%s"
- No relevant notes were found

REQUIREMENTS:
1. Match the language the user used in their query
2. Be helpful and suggest what they could try
3. Keep it to 1-2 sentences

Generate the message now:
</task>`, query)

	messages := append(history, llm.Message{Role: "user", Content: prompt})

	response, err := m.llmProvider.Chat(ctx, messages)
	if err != nil {
		m.logger.Printf("[WARN] Not found message generation failed: %v", err)
		return "I couldn't find any notes matching your search. Try different keywords or check if the note exists."
	}

	return response
}

// GenerateClarifyMessage creates a message asking for more details
func (m *AdaptiveMessenger) GenerateClarifyMessage(
	ctx context.Context,
	history []llm.Message,
) string {
	prompt := `<task>
Generate a brief clarification request.

CONTEXT:
- User's query was unclear or too vague
- Need them to provide more details

REQUIREMENTS:
1. Match the language from the conversation
2. Be polite and helpful
3. Keep it to 1 sentence

Generate the message now:
</task>`

	messages := append(history, llm.Message{Role: "user", Content: prompt})

	response, err := m.llmProvider.Chat(ctx, messages)
	if err != nil {
		m.logger.Printf("[WARN] Clarify message generation failed: %v", err)
		return "Could you provide more details about what you're looking for?"
	}

	return response
}

// GenerateInvalidSelectionMessage creates a message for invalid selection
func (m *AdaptiveMessenger) GenerateInvalidSelectionMessage(
	ctx context.Context,
	maxOptions int,
	history []llm.Message,
) string {
	prompt := fmt.Sprintf(`<task>
Generate a brief "invalid selection" message.

CONTEXT:
- User tried to select an option that doesn't exist
- Valid options are 1 to %d

REQUIREMENTS:
1. Match the language from the conversation
2. Be helpful, not scolding
3. Keep it to 1 sentence

Generate the message now:
</task>`, maxOptions)

	messages := append(history, llm.Message{Role: "user", Content: prompt})

	response, err := m.llmProvider.Chat(ctx, messages)
	if err != nil {
		m.logger.Printf("[WARN] Invalid selection message generation failed: %v", err)
		return fmt.Sprintf("That option isn't available. Please choose a number between 1 and %d.", maxOptions)
	}

	return response
}

// GenerateLostContextMessage creates a message when session context is lost
func (m *AdaptiveMessenger) GenerateLostContextMessage(
	ctx context.Context,
	history []llm.Message,
) string {
	prompt := `<task>
Generate a brief "lost context" message asking user to search again.

REQUIREMENTS:
1. Match the language from the conversation
2. Apologize briefly and ask them to search again
3. Keep it to 1-2 sentences

Generate the message now:
</task>`

	messages := append(history, llm.Message{Role: "user", Content: prompt})

	response, err := m.llmProvider.Chat(ctx, messages)
	if err != nil {
		m.logger.Printf("[WARN] Lost context message generation failed: %v", err)
		return "I seem to have lost track of the context. Could you please search again?"
	}

	return response
}

// GenerateErrorMessage creates a generic error message
func (m *AdaptiveMessenger) GenerateErrorMessage(
	ctx context.Context,
	history []llm.Message,
) string {
	// For errors, use a fast fallback - no need to call LLM
	return "Sorry, something went wrong while processing your request. Please try again."
}

// Fallback messages (English) when LLM fails
func (m *AdaptiveMessenger) fallbackAmbiguityMessage(candidates []store.Document) string {
	var builder strings.Builder
	builder.WriteString("I found several relevant notes. Which one would you like to focus on?\n")
	for i, c := range candidates {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}
	builder.WriteString("\nOr say 'all' to get information from all of them.")
	return builder.String()
}
