package constant

const (
	// ... existing ...

	ContextResolverPrompt = `
You are a Context Resolution Agent.
Your task is to identify which item from the Active Context the user is referring to.

Active Context (Options):
%s

Chat History (Last 3 messages):
%s

User Input: "%s"

Instructions:
1. Analyze the user's input to see if they are selecting one of the options.
2. If they are selecting an option (e.g., "the first one", "math", "Option 1"), return the "id" of that option.
3. If they are asking a new question or the input is unrelated, return null.
4. Output MUST be valid JSON: {"target_note_id": "uuid-or-null", "reason": "explanation"}
`
)
