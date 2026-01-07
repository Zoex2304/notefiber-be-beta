package response

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ai-notetaking-be/pkg/llm"
	ragcontext "ai-notetaking-be/pkg/rag/context"
	"ai-notetaking-be/pkg/rag/intent"
	"ai-notetaking-be/pkg/store"
)

// Generator creates contextual responses based on grounded context
type Generator struct {
	llmProvider llm.LLMProvider
	logger      *log.Logger
}

// NewGenerator creates a new response generator
func NewGenerator(llmProvider llm.LLMProvider, logger *log.Logger) *Generator {
	return &Generator{
		llmProvider: llmProvider,
		logger:      logger,
	}
}

// GenerateFromGroundedContext creates an answer using ONLY the grounded context
// This is Phase 3 - Answer generation from grounded data
func (g *Generator) GenerateFromGroundedContext(
	ctx context.Context,
	query string,
	groundedContext *ragcontext.GroundedContext,
	history []llm.Message,
) string {

	// Handle META_ANALYSIS (Scope None)
	if groundedContext != nil && groundedContext.Scope == intent.ScopeNone {
		// Just answer from history
		g.logger.Printf("[GENERATION] Meta-Analysis requested (Scope: NONE)")
		promptText := fmt.Sprintf("<task>\nAnswer the user's question regarding the CONVERSATION HISTORY above.\nDo NOT look for new information.\n</task>\n\nQuestion: %s", query)
		fullHistory := append(history, llm.Message{Role: "user", Content: promptText})

		response, err := g.llmProvider.Chat(ctx, fullHistory)
		if err != nil {
			g.logger.Printf("[ERROR] LLM generation failed: %v", err)
			return "Sorry, an error occurred while generating the answer."
		}
		return response
	}

	if groundedContext == nil || len(groundedContext.Notes) == 0 {
		g.logger.Printf("[ERROR] Cannot generate: no grounded context")
		return "Sorry, no context is available to answer your question."
	}

	// Build grounded prompt
	promptText := g.buildGroundedPrompt(query, groundedContext)

	// Create message history with grounded context
	fullHistory := append(history, llm.Message{Role: "user", Content: promptText})

	// Generate response
	response, err := g.llmProvider.Chat(ctx, fullHistory)
	if err != nil {
		g.logger.Printf("[ERROR] LLM generation failed: %v", err)
		return "Maaf, terjadi kesalahan saat menyusun jawaban."
	}

	g.logger.Printf("[GENERATION] Answer generated from %d notes (Scope: %s)",
		len(groundedContext.Notes), groundedContext.Scope)

	return response
}

func (g *Generator) buildGroundedPrompt(query string, groundedContext *ragcontext.GroundedContext) string {
	var prompt strings.Builder

	// 1. Inject Context Menu (Semantic Awareness)
	// This tells the LLM "Where we are" and "What the user sees"
	if len(groundedContext.Candidates) > 0 {
		prompt.WriteString("<context_menu>\n")
		prompt.WriteString("The user is viewing the following list of documents:\n")
		for i, c := range groundedContext.Candidates {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
		}
		prompt.WriteString("</context_menu>\n\n")

		// EXPLICIT HINT: If we focused on a specific item index, tell the LLM
		if groundedContext.FocusIndex > 0 {
			prompt.WriteString(fmt.Sprintf("SYSTEM CONFIRMATION: The user selected Item #%d. The content below belongs to Item #%d.\n\n", groundedContext.FocusIndex, groundedContext.FocusIndex))
		}
	}

	// 2. Inject grounded reference material (The actual data)
	prompt.WriteString("<grounded_reference_material>\n")
	prompt.WriteString("CRITICAL: This is the ONLY data source. Do NOT use outside knowledge.\n")
	prompt.WriteString("Structure: Each note is separated by headers. Treat them as distinct sources.\n\n")

	for _, note := range groundedContext.Notes {
		g.logger.Printf("[GENERATION] Grounding Note: '%s' (Length: %d characters)", note.Title, len(note.Content))
		prompt.WriteString(fmt.Sprintf("\n--- CONTENT OF: %s ---\n", note.Title))
		prompt.WriteString(note.Content)
		prompt.WriteString(fmt.Sprintf("\n--- END OF: %s ---\n", note.Title))
	}
	prompt.WriteString("</grounded_reference_material>\n\n")

	// 3. Task description with Semantic Logic
	prompt.WriteString("<task_instructions>\n")
	prompt.WriteString("You are a diligent assistant answering based on the provided content.\n\n")

	prompt.WriteString("EXECUTION RULES (MUST FOLLOW):\n")
	prompt.WriteString("1. ANSWER DIRECTLY if sufficient data exists. Never ask 'Do you want me to...'.\n")
	prompt.WriteString("2. Extract ALL relevant values from ALL provided notes.\n")
	prompt.WriteString("3. Show your work step-by-step for any calculations.\n")
	prompt.WriteString("4. Always provide a FINAL numeric answer (e.g., 'Profit = $14,000').\n\n")

	prompt.WriteString("RESPONSE STYLE:\n")
	prompt.WriteString("1. Match your tone and format to the user's question style.\n")
	prompt.WriteString("2. For direct note references (user explicitly mentions a note), use 'According to [Title]...'.\n")
	prompt.WriteString("3. For exploratory questions (e.g., 'I think I have notes about...'), be conversational and confirmatory.\n")
	prompt.WriteString("4. DO NOT use [N1], [N2], or similar markers. Citations are handled separately by the system.\n\n")

	prompt.WriteString("GROUNDING RULES:\n")
	prompt.WriteString("1. Answer ONLY using the text in <grounded_reference_material>.\n")
	prompt.WriteString("2. If the user asks for 'all' or 'every', be EXHAUSTIVE.\n")
	prompt.WriteString("3. For counting, list items explicitly and count what is visible.\n\n")

	prompt.WriteString("FORMATTING INSTRUCTIONS:\n")
	prompt.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	prompt.WriteString("## 📋 Adaptive Output Formatting\n\n")

	prompt.WriteString("### Core Principles\n")
	prompt.WriteString("1. **Clarity First:** Structure your response for maximum readability\n")
	prompt.WriteString("2. **Context-Aware:** Adapt formatting to the content type (questions, notes, code, analysis, etc.)\n")
	prompt.WriteString("3. **Visual Hierarchy:** Use markdown elements to create clear information layers\n")
	prompt.WriteString("4. **Consistency:** Maintain uniform styling throughout the response\n\n")

	prompt.WriteString("### Markdown Formatting Standards\n\n")

	prompt.WriteString("**Headers:**\n")
	prompt.WriteString("• Use `##` for main sections/topics\n")
	prompt.WriteString("• Use `###` for subsections\n")
	prompt.WriteString("• Use `####` for detailed breakdowns\n\n")

	prompt.WriteString("**Emphasis:**\n")
	prompt.WriteString("• **Bold** for answers, key terms, important conclusions, and critical information\n")
	prompt.WriteString("• *Italic* for explanations, notes, context, or supporting details\n")
	prompt.WriteString("• `Code blocks` for technical content, formulas, or literal values\n\n")

	prompt.WriteString("**Lists:**\n")
	prompt.WriteString("• Use bullet points (•) for unordered information\n")
	prompt.WriteString("• Use numbers (1. 2. 3.) for sequential steps or ranked items\n")
	prompt.WriteString("• Use checkboxes (- [ ]) for actionable items or checklists\n\n")

	prompt.WriteString("**Sections:**\n")
	prompt.WriteString("• Use `---` as dividers between major topics\n")
	prompt.WriteString("• Add blank lines between sections for breathing room\n")
	prompt.WriteString("• Group related information together\n\n")

	prompt.WriteString("### Content-Specific Formatting\n\n")

	prompt.WriteString("**For Questions/Answers:**\n")
	prompt.WriteString("```\n")
	prompt.WriteString("## [Question]\n")
	prompt.WriteString("**Answer:** [Your answer]\n")
	prompt.WriteString("*Context:* [Supporting information from source]\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("**For Explanations/Concepts:**\n")
	prompt.WriteString("```\n")
	prompt.WriteString("## [Concept Name]\n")
	prompt.WriteString("[Clear explanation]\n\n")
	prompt.WriteString("**Key Points:**\n")
	prompt.WriteString("• Point 1\n")
	prompt.WriteString("• Point 2\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("**For Comparisons:**\n")
	prompt.WriteString("```\n")
	prompt.WriteString("| Aspect | Option A | Option B |\n")
	prompt.WriteString("|--------|----------|----------|\n")
	prompt.WriteString("| Detail | Info     | Info     |\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("**For Step-by-Step Information:**\n")
	prompt.WriteString("```\n")
	prompt.WriteString("1. **Step One:** Description\n")
	prompt.WriteString("2. **Step Two:** Description\n")
	prompt.WriteString("3. **Step Three:** Description\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("**For Code or Technical Content:**\n")
	prompt.WriteString("````\n")
	prompt.WriteString("```language\n")
	prompt.WriteString("[code here]\n")
	prompt.WriteString("```\n")
	prompt.WriteString("**Explanation:** [What this code does]\n")
	prompt.WriteString("````\n\n")

	prompt.WriteString("### Visual Enhancement\n\n")
	prompt.WriteString("Use emojis strategically for visual markers (optional):\n")
	prompt.WriteString("• 📝 Notes/Explanations\n")
	prompt.WriteString("• 💡 Key insights\n")
	prompt.WriteString("• ⚠️ Important warnings\n")
	prompt.WriteString("• ✓ Correct/Confirmed\n")
	prompt.WriteString("• ✗ Incorrect/Avoid\n")
	prompt.WriteString("• 📊 Data/Statistics\n")
	prompt.WriteString("• 🔍 Details/Analysis\n\n")

	prompt.WriteString("### Spacing Rules\n\n")
	prompt.WriteString("• **Between major sections:** 2 blank lines\n")
	prompt.WriteString("• **Between subsections:** 1 blank line\n")
	prompt.WriteString("• **Between list items:** No blank lines (unless complex items)\n")
	prompt.WriteString("• **After headers:** 1 blank line before content\n\n")

	prompt.WriteString("### Response Quality Guidelines\n\n")
	prompt.WriteString("✓ Lead with the most relevant information\n")
	prompt.WriteString("✓ Use progressive disclosure (summary → details)\n")
	prompt.WriteString("✓ Highlight actionable items or key takeaways\n")
	prompt.WriteString("✓ Reference source material when providing facts\n")
	prompt.WriteString("✓ Keep paragraphs concise (3-5 lines max)\n")
	prompt.WriteString("✓ Use tables for structured comparisons\n")
	prompt.WriteString("✓ Break complex information into digestible chunks\n\n")

	prompt.WriteString("### Adaptive Behavior\n\n")
	prompt.WriteString("**Detect content type and adjust:**\n")
	prompt.WriteString("• Academic content → Structured, formal, citation-heavy\n")
	prompt.WriteString("• Technical docs → Code blocks, examples, step-by-step\n")
	prompt.WriteString("• Meeting notes → Action items, decisions, key points\n")
	prompt.WriteString("• Research → Findings, methodology, conclusions\n")
	prompt.WriteString("• General notes → Flexible, focus on clarity\n\n")

	prompt.WriteString("**Always prioritize:**\n")
	prompt.WriteString("1. Accuracy over verbosity\n")
	prompt.WriteString("2. Clarity over complexity\n")
	prompt.WriteString("3. Relevance over completeness\n")
	prompt.WriteString("4. User intent over literal interpretation\n\n")

	prompt.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	prompt.WriteString("</task_instructions>\n\n")

	// User query
	prompt.WriteString("<user_question>\n")
	prompt.WriteString(query)
	prompt.WriteString("\n</user_question>\n\n")

	prompt.WriteString("Answer:")

	return prompt.String()
}

// GetCitations returns citations based on grounded context
func (g *Generator) GetCitations(groundedContext *ragcontext.GroundedContext) []string {
	if groundedContext == nil {
		return []string{}
	}
	return groundedContext.IDs
}

// ===============================================
// BACKWARD COMPATIBILITY METHODS (for old action.go)
// These will be deprecated once migration is complete
// ===============================================

// GenerateNotFoundMessage returns not found message
func (g *Generator) GenerateNotFoundMessage() string {
	return "I couldn't find any notes matching your search."
}

// GenerateBrowsingMessage returns browsing menu
func (g *Generator) GenerateBrowsingMessage(candidates []store.Document) string {
	var builder strings.Builder
	builder.WriteString("I found several relevant notes. Which one would you like to focus on?\n")

	for i, c := range candidates {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}

	return builder.String()
}

// GenerateInvalidSelectionMessage returns invalid selection error
func (g *Generator) GenerateInvalidSelectionMessage() string {
	return "Invalid selection. Please choose a number from the available options."
}

// GenerateAnswer generates answer from session (backward compat)
func (g *Generator) GenerateAnswer(
	ctx context.Context,
	session *store.Session,
	query string,
	history []llm.Message,
) string {
	if session.FocusedNote == nil {
		g.logger.Printf("[ERROR] Cannot generate answer: no focused context")
		return "I seem to have lost the context. Could you please search again?"
	}

	// Convert to grounded context
	groundedContext := &ragcontext.GroundedContext{
		Notes: []ragcontext.NoteContent{{
			ID:      session.FocusedNote.ID,
			Title:   session.FocusedNote.Title,
			Content: session.FocusedNote.Content,
		}},
		Scope:     intent.ScopeSingle,
		FocusedID: session.FocusedNote.ID,
		IDs:       []string{session.FocusedNote.ID},
	}

	return g.GenerateFromGroundedContext(ctx, query, groundedContext, history)
}
