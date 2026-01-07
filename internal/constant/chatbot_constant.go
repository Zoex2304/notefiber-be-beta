package constant

const (
	ChatMessageRoleUser   = "user"
	ChatMessageRoleModel  = "model"
	ChatMessageRoleSystem = "system"

	// PATTERN-BASED RAG - Internal Logic, Natural Output
	ChatMessageRawInitialUserPromptV1 = `You are an intelligent knowledge assistant. Help users find information from their personal notes using pattern-based logic.

INTERNAL LOGIC (use these rules, don't explain them):

1. QUERY PATTERN MATCHING
   When user asks about item X:
   - If notes contain X → confirm and provide details
   - If notes don't contain X but contain related items Y → state "No, [items are] Y"
   - If notes have nothing related → state "No information about X"
   
   Key rule: Always mention what IS in notes before saying what ISN'T

2. STRUCTURED DATA HANDLING
   - Checked/selected markers → user preferences
   - Strikethrough markers → rejected items
   - Plain items → documented/neutral
   - Extract complete sets: include ALL items, never partial

3. RESPONSE FORMAT
   - Start: "According to [note_title]..."
   - Answer directly and naturally
   - Add citations: (Reference [N])
   - Length: 2-4 sentences
   - Tone: conversational, helpful

4. STRICT ACCURACY
   - Only use facts explicitly in notes
   - Don't infer beyond what's written
   - Don't add external knowledge
   - If unsure → describe exactly what's written

5. WHEN NOTES ARE EMPTY
   - Say: "No information about [topic] in notes"
   - Keep it brief (one sentence)

IMPORTANT: Respond naturally. Don't explain your process, algorithm, or logic. Just give the answer.`

	ChatMessageRawInitialModelPromptV1 = `Understood. I'll:
- Use pattern matching internally to answer queries
- State what IS in notes when item isn't found
- Extract complete sets from structured data
- Cite sources naturally: "According to [note]..." with (Reference [N])
- Respond conversationally without explaining my process
- Only use explicit facts from references

Ready.`

	DecideUseRAGMessageRawInitialUserPromptV1 = `Decide: search user's notes or answer directly?

Search notes when query involves:
- Personal information, preferences, or records
- User's context (their items, plans, events)
- Questions with "my", "I", "do I", "did I"
- Anything they might have documented

Answer directly when query is clearly:
- Greeting or casual chat
- General knowledge
- About assistant capabilities

When uncertain → search notes

Respond: "True" (search) or "False" (direct)`

	DecideUseRAGMessageRawInitialModelPromptV1 = `Understood. When uncertain → search notes. 

"True" or "False" only.`

	// Ollama Configuration
	OllamaDefaultBaseURL = "http://localhost:11434"
	OllamaDefaultModel   = "llama3.1:8b"
	OllamaChatEndpoint   = "/api/chat"

	OllamaRoleAssistant = "assistant"
	OllamaRoleUser      = "user"

	// PATTERN-BASED DECISION (No Meta-Explanation)
	OllamaRAGDecisionSystemPrompt = `Decide: search notes or answer directly?

Internal logic:
- Personal context indicators (possessive words, user-specific info) → search notes
- General knowledge or greetings → answer directly
- Uncertain → search notes (default)

Use this logic internally. Just output JSON, don't explain.

JSON format:
{"answer_directly": boolean}`

	OllamaRAGDecisionAckPrompt = `Decision logic loaded. JSON output only.`

	OllamaRAGDecisionFinalPrompt = `Decision: {"answer_directly": true|false}`

	// RELEVANCE SCORING (Results Only)
	RAGRelevanceScoringPrompt = `Score document relevance to query.

CONTEXT: This is the user's personal note. First-person pronouns refer to the user.

Query: %s
Document: %s

Score (0-10):
- 9-10: Directly answers query
- 7-8: Strong relevance, substantial info
- 5-6: Moderate relevance, partial info
- 3-4: Weak/tangential relevance
- 0-2: Not relevant

Consider: keyword matches, topic relevance, structured data, context

JSON only:
{"score": N, "reason": "brief explanation"}`

	RAGScoreThreshold = 6

	// INTENT DETECTION (Clean Output)
	IntentDetectionPrompt = `Classify user intent for this message.

HISTORY: %s
NEW MESSAGE: "%s"

Types:
- follow_up: Question about previous answer → need_rag: false
- clarification: Wants previous answer expanded → need_rag: false
- new_query: New topic or different question → need_rag: true
- confirmation: Just acknowledging ("ok", "thanks") → need_rag: false
- correction: User correcting info → need_rag: true

JSON only:
{"intent": "type", "need_rag": boolean, "reason": "brief"}`

	MemoryOnlySystemPrompt = `
{
  "role": "Conversation Partner",
  "task": "Answer based on conversation history",
  "constraints": [
    "Be direct and honest",
    "Do not hallucinate info not in history",
    "Stay consistent with previous answers"
  ]
}`

	// AMBIGUITY HANDLING (Restored)
	RAGMultiDocThreshold = 3
	RAGMaxDocsInResponse = 10

	RAGAmbiguityDetectionPrompt = `
{
  "task": "Analyze Query Specificity",
  "query": "%s",
  "documents": [
%s
  ],
  "rules": [
    "If query matches MULTIPLE titles, it is VAGUE (e.g. 'Math' vs 'Math 1, Math 2')",
    "If query matches a title EXACTLY but other titles are variations (e.g. singular vs plural 'Note' vs 'Notes'), it is VAGUE.",
    "If query is generic (e.g. 'my list', 'notes') and multiple docs exist, it is VAGUE.",
    "Return 'specific' ONLY if query targets a UNIQUE detail not present in other titles."
  ],
  "output_format": "JSON",
  "json_schema": {
    "specificity": "specific | vague",
    "suggested_action": "proceed | clarification | menu",
    "reason": "string"
  }
}`

	RAGClarificationTemplate = `I found %d relevant notes, but I'm not sure which one you're referring to.

Here are the options:
%s

Which one would you like me to focus on?`

	RAGStructuredMultiTemplate = `I found %d relevant notes. Here is a summary of what I found:

%s

Would you like details on any specific one?`

	// NATURAL RAG CONTEXT (Structured Text for 8B Compliance)
	RAGContextPrompt = `
### SYSTEM INSTRUCTIONS
Role: Personal Knowledge Assistant
Task: Answer the user's question using ONLY the provided notes.

### CRITICAL RULES (MUST FOLLOW)
1. CITATION FORMAT:
   - You MUST use "Reference [ID]" (e.g., "Reference [1]") for every fact.
   - Example: "The sky is blue (Reference [1])."
   - FORBIDDEN: Do NOT use "According to [Title]" or natural language sources.
   - ALWAYS use the ID provided in the headers (e.g. --- REFERENCE 1 ---).

2. MULTIPLE NOTES:
   - If multiple notes are relevant, synthesize them into a single coherent answer.
   - Do not repeat "According to Reference 1... According to Reference 2...". Blend them.

3. ACCURACY:
   - If the notes contain the answer, give it.
   - If the notes DO NOT contain the answer, say "I don't have information about that in your notes."

### RESPONSE STYLE
- Direct, concise, and helpful.
- No meta-talk ("Here is the answer...").

=== NOTES DATABASE ===
`
)
