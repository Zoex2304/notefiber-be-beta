//go:build ignore
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-notetaking-be/pkg/llm/ollama"
	"ai-notetaking-be/pkg/store"
)

func main() {
	fmt.Println("=== TEST: INITIAL STATE INTENT ===")

	llmProvider := ollama.NewOllamaProvider("http://localhost:11434", "qwen2.5:7b")

	// EMPTY Session (Initial State)
	session := &store.Session{
		Candidates:  []store.Document{},
		FocusedNote: nil,
	}

	query := "answer my english exam"

	fmt.Printf("\n--- TEST: \"%s\" --- (Expect SEARCH)\n", query)

	prompt := buildPromptSearchTest(query, session)

	// Print prompt for debug
	// fmt.Println(prompt)

	fmt.Println("Thinking...")
	start := time.Now()
	response, err := llmProvider.Generate(context.Background(), prompt)
	if err != nil {
		log.Fatalf("LLM Error: %v", err)
	}
	duration := time.Since(start)

	fmt.Printf("[Time: %s]\n%s\n", duration, response)
}

// Manually synced from pkg/rag/intent/resolver.go
func buildPromptSearchTest(query string, session *store.Session) string {
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

	prompt.WriteString("AGGREGATE: User wants information derived from MULTIPLE notes\n")
	prompt.WriteString("  - Use when: User targets a GROUP (e.g. 'the three files', 'all of them', 'both exams')\n")
	prompt.WriteString("  - Use when: 'total questions from all files', 'compare the two'\n")
	prompt.WriteString("  - Rule: If the target is Plural ('three files'), intent MUST be AGGREGATE.\n")
	prompt.WriteString("  - INVALID if user targets a SINGLE specific note (Use FOCUS)\n\n")

	prompt.WriteString("ANSWER: User asks follow-up on the CURRENTLY focused note\n")
	prompt.WriteString("  - Use when: A note IS ALREADY focused (see <session_state>) and user asks follow-up.\n")
	prompt.WriteString("  - INVALID if 'INITIAL_STATE' or 'BROWSING_MODE' (No note focused). Use SEARCH or FOCUS instead.\n")
	prompt.WriteString("  - INVALID if user explicitly targets a DIFFERENT file (Use FOCUS)\n\n")

	prompt.WriteString("BROWSE: User wants to see the list of options again\n")
	prompt.WriteString("  - Use when: 'show options', 'what are my choices', 'list them'\n\n")

	prompt.WriteString("META_ANALYSIS: User asks about the conversation history itself\n")
	prompt.WriteString("  - Use when: 'what did I just ask?', 'summarize our chat', 'report on previous answers'\n")
	prompt.WriteString("  - Scope: NONE (Does not require Note content)\n\n")

	prompt.WriteString("CLARIFY: Cannot determine intent with confidence\n")
	prompt.WriteString("  - Use only as last resort\n")
	prompt.WriteString("</intent_definitions>\n\n")

	// Output format
	prompt.WriteString("<output_format>\n")
	prompt.WriteString("Respond with ONLY valid JSON:\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"action\": \"SEARCH|FOCUS|AGGREGATE|ANSWER|BROWSE|CLARIFY\",\n")
	prompt.WriteString("  \"target\": 1,\n")
	prompt.WriteString("  \"query\": \"search terms if SEARCH, otherwise empty\",\n")
	prompt.WriteString("  \"scope\": \"ALL|SINGLE|NONE\",\n")
	prompt.WriteString("  \"confidence\": 0.95,\n")
	prompt.WriteString("  \"reasoning\": \"Brief explanation\"\n")
	prompt.WriteString("}\n")
	prompt.WriteString("</output_format>")

	return prompt.String()
}
