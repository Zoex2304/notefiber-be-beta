package prompt

import (
	"strings"

	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/store"
)

// ContextualBuilder builds domain-agnostic prompts
type ContextualBuilder struct {
	session *store.Session
	query   string
	history []llm.Message
}

// NewContextualBuilder creates a new contextual prompt builder
func NewContextualBuilder(session *store.Session, query string, history []llm.Message) *ContextualBuilder {
	return &ContextualBuilder{
		session: session,
		query:   query,
		history: history,
	}
}

// Build creates a semantic, domain-agnostic prompt that trusts LLM intelligence
func (b *ContextualBuilder) Build() string {
	var prompt strings.Builder

	// Inject reference material
	b.writeReferenceMaterial(&prompt)

	// Define task semantically
	b.writeTask(&prompt)

	// Set universal guidelines
	b.writeGuidelines(&prompt)

	// Inject user query
	b.writeUserQuery(&prompt)

	return prompt.String()
}

func (b *ContextualBuilder) writeReferenceMaterial(prompt *strings.Builder) {
	if b.session.FocusedNote == nil {
		return
	}

	prompt.WriteString("<reference_material>\n")
	prompt.WriteString(b.session.FocusedNote.Content)
	prompt.WriteString("\n</reference_material>\n\n")
}

func (b *ContextualBuilder) writeTask(prompt *strings.Builder) {
	prompt.WriteString("<task>\n")
	prompt.WriteString("You are a knowledgeable assistant helping the user understand and extract information from their notes.\n")
	prompt.WriteString("Your goal is to provide exactly what the user needs based on their question and the reference material.\n")
	prompt.WriteString("</task>\n\n")
}

func (b *ContextualBuilder) writeGuidelines(prompt *strings.Builder) {
	prompt.WriteString("<guidelines>\n")
	prompt.WriteString("Understand the user's question semantically:\n")
	prompt.WriteString("- If they need calculations or totals, perform the math and show your work\n")
	prompt.WriteString("- If they need comparisons or analysis, identify patterns and provide insights\n")
	prompt.WriteString("- If they need specific information, extract it comprehensively\n")
	prompt.WriteString("- If they need summaries, synthesize the key points\n")
	prompt.WriteString("- If the material contains structured content (questions, items, entries), address all of them unless the user specifies otherwise\n")
	prompt.WriteString("\n")
	prompt.WriteString("Response principles:\n")
	prompt.WriteString("1. Base your answer strictly on the reference material provided\n")
	prompt.WriteString("2. Adapt your response style to match what the question requires\n")
	prompt.WriteString("3. Be complete - don't skip relevant information from the material\n")
	prompt.WriteString("4. Be clear and well-organized in your presentation\n")
	prompt.WriteString("5. If the material doesn't contain what's being asked, say so honestly\n")
	prompt.WriteString("</guidelines>\n\n")
}

func (b *ContextualBuilder) writeUserQuery(prompt *strings.Builder) {
	prompt.WriteString("<user_question>\n")
	prompt.WriteString(b.query)
	prompt.WriteString("\n</user_question>\n\n")
	prompt.WriteString("Now provide your complete response based on the reference material:")
}
