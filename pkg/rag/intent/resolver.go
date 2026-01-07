package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/store"
)

// Intent represents a stable, resolved user intention
type Intent struct {
	Action       string  `json:"action"`       // SEARCH, FOCUS, AGGREGATE, ANSWER, BROWSE, CLARIFY
	Target       int     `json:"target"`       // Index of target item (for FOCUS/SWITCH)
	Query        string  `json:"query"`        // Search query (for SEARCH)
	Scope        string  `json:"scope"`        // ALL, SINGLE, NONE
	Confidence   float32 `json:"confidence"`   // 0.0-1.0
	Reasoning    string  `json:"reasoning"`    // Why this intent was chosen
	Explicitness string  `json:"explicitness"` // HIGH, MEDIUM, LOW - how explicit is the user's instruction
}

// Action constants
const (
	ActionSearch       = "SEARCH"
	ActionFocus        = "FOCUS"
	ActionAggregate    = "AGGREGATE"
	ActionAnswer       = "ANSWER"
	ActionBrowse       = "BROWSE"
	ActionMetaAnalysis = "META_ANALYSIS"
	ActionClarify      = "CLARIFY"
)

// Scope constants
const (
	ScopeAll    = "ALL"
	ScopeSingle = "SINGLE"
	ScopeNone   = "NONE"
)

// Explicitness constants - how clear/actionable is the user's prompt
const (
	ExplicitnessHigh   = "HIGH"   // Clear, actionable command → execute directly
	ExplicitnessMedium = "MEDIUM" // Intent clear but scope ambiguous → may clarify
	ExplicitnessLow    = "LOW"    // Vague or exploratory → browse/clarify
)

// Resolver performs pure LLM-based intent resolution
// This is Phase 1 - NO RAG, just understanding
type Resolver struct {
	llmProvider llm.LLMProvider
	logger      *log.Logger
}

// NewResolver creates a new intent resolver
func NewResolver(llmProvider llm.LLMProvider, logger *log.Logger) *Resolver {
	return &Resolver{
		llmProvider: llmProvider,
		logger:      logger,
	}
}

// Resolve analyzes the user query and produces a stable intent
// This is a pure LLM call - no RAG, no database access
func (r *Resolver) Resolve(
	ctx context.Context,
	query string,
	history []llm.Message,
	session *store.Session,
) (*Intent, error) {

	// Build context-aware prompt
	prompt := r.buildPrompt(query, history, session)

	// Pure LLM call for intent resolution (Temperature 0 for deterministic output)
	response, err := r.llmProvider.Generate(ctx, prompt, llm.WithTemperature(0.0))
	if err != nil {
		r.logger.Printf("[ERROR] Intent resolution failed: %v", err)
		return r.fallbackIntent(query, session), nil
	}

	// Parse structured intent
	intent, err := r.parseIntent(response)
	if err != nil {
		r.logger.Printf("[WARN] Intent parsing failed, using fallback: %v", err)
		return r.fallbackIntent(query, session), nil
	}

	r.logger.Printf("[INTENT] Resolved: %s (Target: %d, Scope: %s, Explicitness: %s, Confidence: %.2f)",
		intent.Action, intent.Target, intent.Scope, intent.Explicitness, intent.Confidence)

	return intent, nil
}

func (r *Resolver) buildPrompt(query string, history []llm.Message, session *store.Session) string {
	var prompt strings.Builder

	prompt.WriteString("<system>\n")
	prompt.WriteString("You are an intent analyzer. Your ONLY job is to understand what the user wants to DO.\n")
	prompt.WriteString("You do NOT answer questions. You only classify intent.\n")
	prompt.WriteString("</system>\n\n")

	// Session state context
	prompt.WriteString("<session_state>\n")
	if session.FocusedNote != nil && session.FocusedNote.ID != "aggregated" {
		prompt.WriteString(fmt.Sprintf("FOCUSED_NOTE: \"%s\"\n", session.FocusedNote.Title))
		prompt.WriteString("User is currently viewing a specific note.\n")
	} else if len(session.Candidates) > 0 {
		prompt.WriteString("BROWSING_MODE: User was shown a list of options:\n")
		for i, c := range session.Candidates {
			prompt.WriteString(fmt.Sprintf("  %d. \"%s\"\n", i+1, c.Title))
		}
		prompt.WriteString("NO note is currently focused. User must SELECT one first.\n")
	} else {
		prompt.WriteString("INITIAL_STATE: No notes loaded yet.\n")
	}
	prompt.WriteString("</session_state>\n\n")

	// User query
	prompt.WriteString("<user_query>\n")
	prompt.WriteString(query)
	prompt.WriteString("\n</user_query>\n\n")

	// Intent definitions
	prompt.WriteString("<intent_definitions>\n")
	prompt.WriteString("Choose ONE intent that best matches what the user wants:\n\n")

	prompt.WriteString("SEARCH: User wants to find notes on a NEW topic or START a new search\n")
	prompt.WriteString("  - Use when: User introduces new subject (e.g. 'answer my english exam', 'search for biology')\n")
	prompt.WriteString("  - Use when: 'INITIAL_STATE' is active (No notes loaded yet)\n")
	prompt.WriteString("  - Requires: query (what to search for)\n\n")

	prompt.WriteString("FOCUS: User wants to select ONE specific item from the list\n")
	prompt.WriteString("  - Use when: User targets a SINGLE file (e.g. 'first one', 'file 2', 'English Exam')\n")
	prompt.WriteString("  - Use when: User targets CONTENT within a SINGLE file (e.g. 'read all questions in the third file', 'summarize file 1')\n")
	prompt.WriteString("  - Rule: If the target is Singular ('third file'), intent MUST be FOCUS.\n")
	prompt.WriteString("  - Requires: target (1-indexed)\n\n")

	prompt.WriteString("AGGREGATE: User wants information derived from MULTIPLE notes or ALL available data\n")
	prompt.WriteString("  - Use when: User asks for 'Profit', 'Total', 'Summary', 'Count', or 'Compare'\n")
	prompt.WriteString("  - Use when: The answer requires combining numbers/data from different files (e.g. 'calculate business profit')\n")
	prompt.WriteString("  - Use when: User asks about the collection as a whole (e.g. 'what are these files about?')\n")
	prompt.WriteString("  - Note: Using AGGREGATE will load ALL candidates for the answer\n\n")

	prompt.WriteString("ANSWER: User asks follow-up on the CURRENTLY focused note or PREVIOUS answer\n")
	prompt.WriteString("  - Use when: A note IS ALREADY focused (see <session_state>) and user asks follow-up.\n")
	prompt.WriteString("  - Use when: User asks for clarification like 'are you sure?', 'why is that?', 'explain more' (Assumes context is the previous answer)\n")
	prompt.WriteString("  - INVALID if 'INITIAL_STATE' (No context yet). Use SEARCH.\n")
	prompt.WriteString("  - INVALID if user explicitly targets a DIFFERENT file (Use FOCUS)\n\n")

	prompt.WriteString("BROWSE: User wants to see the list of options again\n")
	prompt.WriteString("  - Use when: 'show options', 'what are my choices', 'list them'\n\n")

	prompt.WriteString("META_ANALYSIS: User asks about the conversation history itself\n")
	prompt.WriteString("  - Use when: 'what did I just ask?', 'summarize our chat', 'report on previous answers'\n")
	prompt.WriteString("  - Scope: NONE (Does not require Note content)\n\n")

	prompt.WriteString("CLARIFY: Cannot determine intent with confidence\n")
	prompt.WriteString("  - Use only if query is gibberish or completely unrelated to notes/chat.\n")
	prompt.WriteString("</intent_definitions>\n\n")

	// Explicitness assessment
	prompt.WriteString("<explicitness_assessment>\n")
	prompt.WriteString("Assess how EXPLICIT the user's instruction is:\n\n")
	prompt.WriteString("HIGH: User gives a clear, actionable command that can be executed immediately\n")
	prompt.WriteString("  - 'Answer all the questions in this exam'\n")
	prompt.WriteString("  - 'Calculate my total profit'\n")
	prompt.WriteString("  - 'How much is my business profit for this month?'\n")
	prompt.WriteString("  - 'Summarize this document'\n\n")
	prompt.WriteString("MEDIUM: User's intent is clear but scope or target is ambiguous\n")
	prompt.WriteString("  - 'Tell me about the exam' (which exam?)\n")
	prompt.WriteString("  - 'What's in my notes?' (all notes?)\n\n")
	prompt.WriteString("LOW: User's request is vague or exploratory\n")
	prompt.WriteString("  - 'Something about costs'\n")
	prompt.WriteString("  - 'Help me with this'\n")
	prompt.WriteString("  - 'What do you have?'\n\n")
	prompt.WriteString("Rule: If Explicitness is HIGH, the system should execute directly without asking.\n")
	prompt.WriteString("      If Explicitness is LOW, the system may browse or ask for clarification.\n")
	prompt.WriteString("</explicitness_assessment>\n\n")

	// Output format
	prompt.WriteString("<output_format>\n")
	prompt.WriteString("Respond with ONLY valid JSON:\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"action\": \"SEARCH|FOCUS|AGGREGATE|ANSWER|BROWSE|CLARIFY\",\n")
	prompt.WriteString("  \"target\": 1,\n")
	prompt.WriteString("  \"query\": \"search terms if SEARCH, otherwise empty\",\n")
	prompt.WriteString("  \"scope\": \"ALL|SINGLE|NONE\",\n")
	prompt.WriteString("  \"explicitness\": \"HIGH|MEDIUM|LOW\",\n")
	prompt.WriteString("  \"confidence\": 0.95,\n")
	prompt.WriteString("  \"reasoning\": \"Brief explanation\"\n")
	prompt.WriteString("}\n")
	prompt.WriteString("</output_format>")

	return prompt.String()
}

func (r *Resolver) parseIntent(response string) (*Intent, error) {
	// Extract JSON from response
	jsonContent := extractJSON(response)
	if jsonContent == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var intent Intent
	if err := json.Unmarshal([]byte(jsonContent), &intent); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Validate and normalize
	intent.Action = strings.ToUpper(intent.Action)
	if intent.Scope == "" {
		intent.Scope = ScopeNone
	}

	// Fallback for Explicitness if not provided
	if intent.Explicitness == "" {
		intent.Explicitness = ExplicitnessMedium
	} else {
		intent.Explicitness = strings.ToUpper(intent.Explicitness)
	}

	// Adjust 1-based target to 0-based for internal use
	if intent.Target > 0 {
		intent.Target = intent.Target - 1
	}

	return &intent, nil
}

func (r *Resolver) fallbackIntent(query string, session *store.Session) *Intent {
	// Smart fallback based on session state
	if len(session.Candidates) == 0 && session.FocusedNote == nil {
		// Initial state - must search
		return &Intent{
			Action:     ActionSearch,
			Query:      query,
			Scope:      ScopeNone,
			Confidence: 0.5,
			Reasoning:  "Fallback: No context available, defaulting to search",
		}
	}

	if session.FocusedNote != nil && session.FocusedNote.ID != "aggregated" {
		// Has focus - answer about it
		return &Intent{
			Action:     ActionAnswer,
			Scope:      ScopeSingle,
			Confidence: 0.5,
			Reasoning:  "Fallback: Note is focused, assuming follow-up question",
		}
	}

	// Browsing mode - show options
	return &Intent{
		Action:     ActionBrowse,
		Scope:      ScopeNone,
		Confidence: 0.5,
		Reasoning:  "Fallback: In browsing mode, showing options",
	}
}

func extractJSON(response string) string {
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return ""
	}

	return response[startIdx : endIdx+1]
}
