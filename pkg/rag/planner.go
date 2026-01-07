package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/store"
)

// PlannerService orchestrates semantic intent analysis
type PlannerService struct {
	llmProvider     llm.LLMProvider
	contextRenderer *ContextRenderer
	promptComposer  *PromptComposer
}

// NewPlannerService constructs a semantically-aware planner
func NewPlannerService(llmProvider llm.LLMProvider) *PlannerService {
	return &PlannerService{
		llmProvider:     llmProvider,
		contextRenderer: NewContextRenderer(),
		promptComposer:  NewPromptComposer(),
	}
}

// AnalyzeIntent interprets user intention through semantic understanding
func (p *PlannerService) AnalyzeIntent(
	ctx context.Context,
	session *store.Session,
	userQuery string,
	history []llm.Message,
) (*dto.AIActionPlan, error) {
	
	// Render rich semantic context for the model
	semanticContext := p.contextRenderer.Render(session, history)
	
	// Compose structured reasoning prompt
	prompt := p.promptComposer.Compose(semanticContext, userQuery)
	
	// Let LLM understand semantics naturally
	response, err := p.llmProvider.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("llm generation failed: %w", err)
	}
	
	// Extract structured action plan
	plan, err := extractActionPlan(response)
	if err != nil {
		return nil, fmt.Errorf("action plan extraction failed: %w", err)
	}
	
	return plan, nil
}

// ContextRenderer transforms system state into rich semantic description
type ContextRenderer struct{}

func NewContextRenderer() *ContextRenderer {
	return &ContextRenderer{}
}

// SemanticContext represents the interpreted semantic state
type SemanticContext struct {
	ConversationNarrative string
	AvailableItems        string
	CurrentFocus          string
	SystemState           string
	DialogueFlow          string
}

// Render creates a rich semantic description of current context
func (r *ContextRenderer) Render(
	session *store.Session,
	history []llm.Message,
) SemanticContext {
	
	return SemanticContext{
		ConversationNarrative: r.narrateConversation(history),
		AvailableItems:        r.describeAvailableItems(session.Candidates),
		CurrentFocus:          r.describeFocus(session.FocusedNote),
		SystemState:           r.interpretState(session.State),
		DialogueFlow:          r.characterizeDialogue(history),
	}
}

// narrateConversation creates a flowing narrative of recent exchanges
func (r *ContextRenderer) narrateConversation(history []llm.Message) string {
	if len(history) == 0 {
		return "This is the beginning of our conversation. No prior context exists."
	}
	
	windowSize := 4
	if len(history) < windowSize {
		windowSize = len(history)
	}
	
	var narrative strings.Builder
	narrative.WriteString("Recent conversation flow:\n")
	
	startIdx := len(history) - windowSize
	for i := startIdx; i < len(history); i++ {
		speaker := r.identifySpeaker(history[i].Role)
		message := r.trimContent(history[i].Content, 200)
		
		narrative.WriteString(fmt.Sprintf("%s said: \"%s\"\n", speaker, message))
	}
	
	return narrative.String()
}

func (r *ContextRenderer) identifySpeaker(role string) string {
	if role == "assistant" || role == "model" {
		return "Assistant"
	}
	return "User"
}

func (r *ContextRenderer) trimContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// describeAvailableItems creates semantic description of candidates
func (r *ContextRenderer) describeAvailableItems(candidates []store.Document) string {
	if len(candidates) == 0 {
		return "No items are currently available in the knowledge base."
	}
	
	var description strings.Builder
	description.WriteString(fmt.Sprintf("There are %d items available:\n", len(candidates)))
	
	for i, doc := range candidates {
		description.WriteString(fmt.Sprintf("  Item %d: \"%s\"\n", i, doc.Title))
	}
	
	return description.String()
}

// describeFocus articulates the current focus state
func (r *ContextRenderer) describeFocus(note *store.Document) string {
	if note == nil {
		return "No specific item is currently in focus. The conversation is open-ended."
	}
	
	return fmt.Sprintf("The conversation is currently focused on: \"%s\"", note.Title)
}

// interpretState provides semantic meaning to system state
func (r *ContextRenderer) interpretState(state string) string {
	stateInterpretations := map[string]string{
		"initial": "The system is in its initial state. " +
			"The user has not yet explored any specific items. " +
			"This is a discovery phase where the user may want to search or explore.",
		
		"has_candidates": "Multiple items have been retrieved and are available for discussion. " +
			"The user can choose to focus on one, ask about multiple, or search for something different.",
		
		"focused": "A specific item has been selected and is the center of discussion. " +
			"The user may ask detailed questions about it, switch to another item, or explore further.",
		
		"answered": "Information has been provided to the user. " +
			"The user may continue exploring the current topic, switch focus, or search for new information.",
	}
	
	if interpretation, exists := stateInterpretations[state]; exists {
		return interpretation
	}
	
	return fmt.Sprintf("Current system state: %s", state)
}

// characterizeDialogue analyzes the nature of the conversation
func (r *ContextRenderer) characterizeDialogue(history []llm.Message) string {
	if len(history) == 0 {
		return "The dialogue has just begun."
	}
	
	if len(history) <= 2 {
		return "This is an early-stage conversation with limited context."
	}
	
	// Analyze depth and continuity
	userTurns := 0
	for i := len(history) - 1; i >= 0 && i >= len(history)-6; i-- {
		if history[i].Role == "user" {
			userTurns++
		}
	}
	
	if userTurns >= 3 {
		return "The user is engaged in an extended exploration. " +
			"They have asked multiple questions, suggesting they are building understanding or comparing information."
	}
	
	return "The conversation is developing. The user is exploring the knowledge base."
}

// PromptComposer structures the reasoning framework for semantic interpretation
type PromptComposer struct{}

func NewPromptComposer() *PromptComposer {
	return &PromptComposer{}
}

// Compose creates a structured XML prompt that guides semantic reasoning
func (c *PromptComposer) Compose(context SemanticContext, userQuery string) string {
	var prompt strings.Builder
	
	c.writeSystemRole(&prompt)
	c.writeActionDefinitions(&prompt)
	c.writeCriticalExamples(&prompt)
	c.writeSemanticContext(&prompt, context)
	c.writeUserInput(&prompt, userQuery)
	c.writeReasoningFramework(&prompt)
	c.writeOutputStructure(&prompt)
	
	return prompt.String()
}

func (c *PromptComposer) writeSystemRole(prompt *strings.Builder) {
	prompt.WriteString("<system_role>\n")
	prompt.WriteString("You are a semantic intent analyzer for an intelligent knowledge base assistant.\n")
	prompt.WriteString("Your purpose is to understand what the user truly wants, not just match keywords.\n")
	prompt.WriteString("You interpret meaning from context, conversation flow, and natural language understanding.\n")
	prompt.WriteString("</system_role>\n\n")
}

func (c *PromptComposer) writeActionDefinitions(prompt *strings.Builder) {
	prompt.WriteString("<action_definitions>\n")
	prompt.WriteString("You must choose ONE action that best represents the user's SEMANTIC INTENT.\n")
	prompt.WriteString("Consider not just what the user says, but what they are trying to ACHIEVE.\n\n")
	
	prompt.WriteString("<action name=\"SEARCH\">\n")
	prompt.WriteString("  Intent Type: DISCOVERY - User needs new information not currently available\n")
	prompt.WriteString("  When to use:\n")
	prompt.WriteString("    - User introduces completely new topic/subject\n")
	prompt.WriteString("    - User asks about something not covered in available items\n")
	prompt.WriteString("    - User wants to explore different domain/area\n")
	prompt.WriteString("  Requires: search_query\n")
	prompt.WriteString("  Example: \"Find notes about machine learning\" (when no ML notes available)\n")
	prompt.WriteString("</action>\n\n")
	
	prompt.WriteString("<action name=\"SELECT\">\n")
	prompt.WriteString("  Intent Type: FOCUSING - User wants to dive into one specific item\n")
	prompt.WriteString("  When to use:\n")
	prompt.WriteString("    - User references specific item by position, title, or description\n")
	prompt.WriteString("    - User wants detailed discussion about ONE particular item\n")
	prompt.WriteString("    - Question scope is clearly about single item, not multiple\n")
	prompt.WriteString("  Requires: target_index\n")
	prompt.WriteString("  Example: \"Tell me about the second note\" or \"What does the Python tutorial say?\"\n")
	prompt.WriteString("</action>\n\n")
	
	prompt.WriteString("<action name=\"SWITCH\">\n")
	prompt.WriteString("  Intent Type: REFOCUSING - User wants to change focus to different item\n")
	prompt.WriteString("  When to use:\n")
	prompt.WriteString("    - An item is already focused, user wants to look at another one\n")
	prompt.WriteString("    - User signals transition/comparison (\"instead\", \"what about the other one\")\n")
	prompt.WriteString("  Requires: target_index\n")
	prompt.WriteString("  Example: \"Actually, show me the first note instead\"\n")
	prompt.WriteString("</action>\n\n")
	
	prompt.WriteString("<action name=\"ANSWER_CURRENT\">\n")
	prompt.WriteString("  Intent Type: ELABORATION - User wants more details about focused item\n")
	prompt.WriteString("  When to use:\n")
	prompt.WriteString("    - Item is already focused\n")
	prompt.WriteString("    - User asks follow-up questions about that specific item\n")
	prompt.WriteString("    - Question clearly refers to the item in focus\n")
	prompt.WriteString("  Requires: nothing\n")
	prompt.WriteString("  Example: \"Can you explain that part more?\" or \"What does it say about exceptions?\"\n")
	prompt.WriteString("</action>\n\n")
	
	prompt.WriteString("<action name=\"ANSWER_ALL\">\n")
	prompt.WriteString("  Intent Type: SYNTHESIS/AGGREGATION - User needs information derived from ALL items\n")
	prompt.WriteString("  When to use:\n")
	prompt.WriteString("    - User asks for totals, summaries, patterns across multiple items\n")
	prompt.WriteString("    - Question requires combining/calculating data from all available notes\n")
	prompt.WriteString("    - User wants comprehensive overview, not details of one note\n")
	prompt.WriteString("    - User seeks derived insights (totals, averages, trends, comparisons)\n")
	prompt.WriteString("  Requires: nothing\n")
	prompt.WriteString("  Critical Examples:\n")
	prompt.WriteString("    - \"What is my total money?\" → needs to sum across all financial notes\n")
	prompt.WriteString("    - \"How many tasks do I have overall?\" → count across all task notes\n")
	prompt.WriteString("    - \"What are the common themes?\" → analyze patterns across all notes\n")
	prompt.WriteString("    - \"Give me a summary of everything\" → synthesize all notes\n")
	prompt.WriteString("  DO NOT ask \"which note?\" when intent clearly requires ALL notes for answer\n")
	prompt.WriteString("</action>\n\n")
	
	prompt.WriteString("<action name=\"CLARIFY\">\n")
	prompt.WriteString("  Intent Type: AMBIGUOUS - Cannot determine intent with confidence\n")
	prompt.WriteString("  When to use:\n")
	prompt.WriteString("    - User's question genuinely ambiguous (not just lacking keywords)\n")
	prompt.WriteString("    - Multiple interpretations equally valid\n")
	prompt.WriteString("    - Insufficient context to make reasonable inference\n")
	prompt.WriteString("  Requires: nothing\n")
	prompt.WriteString("  Note: Prefer making reasonable inference over asking for clarification\n")
	prompt.WriteString("</action>\n")
	
	prompt.WriteString("</action_definitions>\n\n")
}

func (c *PromptComposer) writeSemanticContext(prompt *strings.Builder, ctx SemanticContext) {
	prompt.WriteString("<semantic_context>\n")
	
	prompt.WriteString("<conversation_narrative>\n")
	prompt.WriteString(ctx.ConversationNarrative)
	prompt.WriteString("</conversation_narrative>\n\n")
	
	prompt.WriteString("<available_items>\n")
	prompt.WriteString(ctx.AvailableItems)
	prompt.WriteString("</available_items>\n\n")
	
	prompt.WriteString("<current_focus>\n")
	prompt.WriteString(ctx.CurrentFocus)
	prompt.WriteString("</current_focus>\n\n")
	
	prompt.WriteString("<system_state>\n")
	prompt.WriteString(ctx.SystemState)
	prompt.WriteString("</system_state>\n\n")
	
	prompt.WriteString("<dialogue_flow>\n")
	prompt.WriteString(ctx.DialogueFlow)
	prompt.WriteString("</dialogue_flow>\n")
	
	prompt.WriteString("</semantic_context>\n\n")
}

func (c *PromptComposer) writeUserInput(prompt *strings.Builder, query string) {
	prompt.WriteString("<user_input>\n")
	prompt.WriteString(query)
	prompt.WriteString("\n</user_input>\n\n")
}

func (c *PromptComposer) writeReasoningFramework(prompt *strings.Builder) {
	prompt.WriteString("<reasoning_framework>\n")
	prompt.WriteString("Follow this semantic reasoning process:\n\n")
	
	prompt.WriteString("1. IDENTIFY QUERY NATURE\n")
	prompt.WriteString("   What type of answer does the user need?\n")
	prompt.WriteString("   - INFORMATIONAL: Reading/understanding content of notes\n")
	prompt.WriteString("   - COMPUTATIONAL: Calculating totals, counts, averages, aggregations\n")
	prompt.WriteString("   - ANALYTICAL: Finding patterns, themes, comparisons across notes\n")
	prompt.WriteString("   - NAVIGATIONAL: Moving between notes, exploring structure\n\n")
	
	prompt.WriteString("2. DETERMINE SCOPE REQUIREMENT\n")
	prompt.WriteString("   Can this question be answered by:\n")
	prompt.WriteString("   - ONE specific note? → Consider SELECT or ANSWER_CURRENT\n")
	prompt.WriteString("   - ALL available notes? → Likely ANSWER_ALL\n")
	prompt.WriteString("   - Information not yet available? → SEARCH\n")
	prompt.WriteString("   \n")
	prompt.WriteString("   Key insight: If question semantically requires data from multiple notes\n")
	prompt.WriteString("   (e.g., \"total\", \"overall\", \"combined\", \"summary\"), DO NOT ask \"which note?\"\n")
	prompt.WriteString("   The user has already told you they need ALL notes.\n\n")
	
	prompt.WriteString("3. INTERPRET USER REFERENCES\n")
	prompt.WriteString("   If user mentions items:\n")
	prompt.WriteString("   - Specific item by position/title? → SELECT with target_index\n")
	prompt.WriteString("   - Plural/collective reference? → ANSWER_ALL\n")
	prompt.WriteString("   - Deictic reference (\"this\", \"that\")? → Check current focus\n\n")
	
	prompt.WriteString("4. CONSIDER CONVERSATION CONTINUITY\n")
	prompt.WriteString("   - Is this continuing previous topic? → ANSWER_CURRENT if item focused\n")
	prompt.WriteString("   - Is this a new direction? → May need SEARCH or SELECT\n")
	prompt.WriteString("   - Is this switching topics? → May need SWITCH\n\n")
	
	prompt.WriteString("5. VALIDATE AGAINST CONTEXT\n")
	prompt.WriteString("   - Are required notes available? If not → SEARCH\n")
	prompt.WriteString("   - Does current focus match user intent? If not → SELECT or SWITCH\n")
	prompt.WriteString("   - Is intent genuinely ambiguous? Only then → CLARIFY\n\n")
	
	prompt.WriteString("6. SELECT ACTION WITH CONFIDENCE\n")
	prompt.WriteString("   Choose the action that:\n")
	prompt.WriteString("   - Matches the query nature (computational → ANSWER_ALL, not SELECT)\n")
	prompt.WriteString("   - Aligns with scope requirement (all notes → ANSWER_ALL)\n")
	prompt.WriteString("   - Respects conversation flow\n")
	prompt.WriteString("   - Minimizes unnecessary user friction\n")
	
	prompt.WriteString("</reasoning_framework>\n\n")
}

func (c *PromptComposer) writeOutputStructure(prompt *strings.Builder) {
	prompt.WriteString("<output_format>\n")
	prompt.WriteString("Respond with ONLY valid JSON in this exact structure:\n\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"action\": \"ACTION_NAME\",\n")
	prompt.WriteString("  \"search_query\": \"query text (only if action is SEARCH, otherwise null)\",\n")
	prompt.WriteString("  \"target_index\": 0 (only if action is SELECT or SWITCH, otherwise null),\n")
	prompt.WriteString("  \"reasoning\": \"Clear explanation of why this action matches the user's semantic intent\"\n")
	prompt.WriteString("}\n\n")
	prompt.WriteString("IMPORTANT: Output ONLY the JSON. No preamble, no explanation outside the JSON.\n")
	prompt.WriteString("</output_format>\n")
}

func (c *PromptComposer) writeCriticalExamples(prompt *strings.Builder) {
	prompt.WriteString("<critical_examples>\n")
	prompt.WriteString("These examples demonstrate the difference between selection and aggregation intent:\n\n")
	
	prompt.WriteString("Example 1: COMPUTATIONAL INTENT → ANSWER_ALL\n")
	prompt.WriteString("Context: 5 notes about finances (mix of debits and credits)\n")
	prompt.WriteString("User: \"What is my total amount of money right now?\"\n")
	prompt.WriteString("Correct Action: ANSWER_ALL\n")
	prompt.WriteString("Reasoning: User needs aggregated calculation across ALL financial notes.\n")
	prompt.WriteString("The word 'total' indicates computational intent requiring all data points.\n")
	prompt.WriteString("WRONG: Do NOT ask \"which note?\". The user has already indicated they need all notes.\n\n")
	
	prompt.WriteString("Example 2: SELECTION INTENT → SELECT\n")
	prompt.WriteString("Context: 5 notes about finances\n")
	prompt.WriteString("User: \"Tell me about the expense note from yesterday\"\n")
	prompt.WriteString("Correct Action: SELECT (target specific note)\n")
	prompt.WriteString("Reasoning: User wants to read ONE specific note, not aggregate data.\n\n")
	
	prompt.WriteString("Example 3: ANALYTICAL AGGREGATION → ANSWER_ALL\n")
	prompt.WriteString("Context: 10 notes from English class (grammar, vocabulary, essays)\n")
	prompt.WriteString("User: \"What are the main topics covered in my English notes?\"\n")
	prompt.WriteString("Correct Action: ANSWER_ALL\n")
	prompt.WriteString("Reasoning: Requires analyzing patterns across ALL notes, not reading one note.\n\n")
	
	prompt.WriteString("Example 4: FOCUSED QUESTION → ANSWER_CURRENT\n")
	prompt.WriteString("Context: Note titled \"Budget 2024\" is currently focused\n")
	prompt.WriteString("User: \"How much did I spend on groceries?\"\n")
	prompt.WriteString("Correct Action: ANSWER_CURRENT\n")
	prompt.WriteString("Reasoning: Question refers to the focused note's content.\n\n")
	
	prompt.WriteString("Example 5: COUNTING AGGREGATION → ANSWER_ALL\n")
	prompt.WriteString("Context: Multiple task notes\n")
	prompt.WriteString("User: \"How many tasks do I have overall?\"\n")
	prompt.WriteString("Correct Action: ANSWER_ALL\n")
	prompt.WriteString("Reasoning: 'Overall' and counting requires checking ALL notes.\n")
	prompt.WriteString("WRONG: Do NOT ask \"which note?\". The semantics demand aggregation.\n\n")
	
	prompt.WriteString("Example 6: NEW TOPIC → SEARCH\n")
	prompt.WriteString("Context: Only finance notes available\n")
	prompt.WriteString("User: \"Show me my workout routine\"\n")
	prompt.WriteString("Correct Action: SEARCH (query: \"workout routine\")\n")
	prompt.WriteString("Reasoning: Topic not covered in available notes, need to search.\n\n")
	
	prompt.WriteString("Key Principle: When user's question semantically requires data from multiple/all notes\n")
	prompt.WriteString("(totals, counts, patterns, summaries), choose ANSWER_ALL immediately.\n")
	prompt.WriteString("Do NOT default to asking \"which note?\" when the intent clearly spans multiple notes.\n")
	prompt.WriteString("</critical_examples>\n\n")
}

// extractActionPlan parses structured response into action plan
func extractActionPlan(response string) (*dto.AIActionPlan, error) {
	jsonContent := extractJSON(response)
	
	var plan dto.AIActionPlan
	if err := json.Unmarshal([]byte(jsonContent), &plan); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}
	
	return &plan, nil
}

// extractJSON isolates JSON content from response
func extractJSON(response string) string {
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return response
	}
	
	return response[startIdx : endIdx+1]
}