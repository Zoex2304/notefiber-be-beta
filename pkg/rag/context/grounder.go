package context

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/lexical"
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/rag/intent"
	"ai-notetaking-be/pkg/rag/search"
	"ai-notetaking-be/pkg/store"

	"github.com/google/uuid"
)

// NoteContent represents full note content for grounding
type NoteContent struct {
	ID      string
	Title   string
	Content string
}

// GroundedContext represents the context that will be used for answer generation
// This is the SINGLE SOURCE OF TRUTH for the answer - not conversation history
type GroundedContext struct {
	Notes      []NoteContent    // Full content of relevant notes
	Candidates []store.Document // Metadata of ALL candidates (Title/ID only) to provide context
	Scope      string           // SINGLE, MULTIPLE, AGGREGATED
	FocusedID  string           // ID of focused note (if SINGLE)
	FocusIndex int              // 1-based index of the focused item (if applicable)
	IDs        []string         // IDs of all notes in context
}

// Grounder loads the correct context based on resolved intent
// This is Phase 2 - Selective RAG
type Grounder struct {
	searchOrchestrator *search.Orchestrator
	embeddingProvider  embedding.EmbeddingProvider
	llmProvider        llm.LLMProvider
	logger             *log.Logger
}

// NewGrounder creates a new context grounder
func NewGrounder(
	searchOrchestrator *search.Orchestrator,
	embeddingProvider embedding.EmbeddingProvider,
	llmProvider llm.LLMProvider,
	logger *log.Logger,
) *Grounder {
	return &Grounder{
		searchOrchestrator: searchOrchestrator,
		embeddingProvider:  embeddingProvider,
		llmProvider:        llmProvider,
		logger:             logger,
	}
}

// GroundingResult contains the grounded context and updated session state
type GroundingResult struct {
	Context       *GroundedContext
	Session       *store.Session
	ShouldAnswer  bool   // Whether to proceed to answer generation
	BrowseMessage string // Message to show if in browse mode
}

// Ground loads the appropriate context based on the resolved intent
func (g *Grounder) Ground(
	ctx context.Context,
	resolvedIntent *intent.Intent,
	session *store.Session,
	uow unitofwork.UnitOfWork,
	userId uuid.UUID,
	history []llm.Message, // Added for adaptive messaging
) (*GroundingResult, error) {

	switch resolvedIntent.Action {
	case intent.ActionSearch:
		return g.groundSearch(ctx, resolvedIntent, session, uow, userId, history)

	case intent.ActionFocus:
		return g.groundFocus(ctx, resolvedIntent, session, uow, history)

	case intent.ActionAggregate:
		return g.groundAggregate(ctx, resolvedIntent, session, uow)

	case intent.ActionAnswer:
		return g.groundAnswer(ctx, session, uow)

	case intent.ActionBrowse:
		return g.groundBrowse(ctx, session, history)

	case intent.ActionMetaAnalysis:
		return g.groundMetaAnalysis(session)

	default:
		return g.groundClarify(ctx, session, history)
	}
}

// groundSearch executes vector search and returns candidates
func (g *Grounder) groundSearch(
	ctx context.Context,
	resolvedIntent *intent.Intent,
	session *store.Session,
	uow unitofwork.UnitOfWork,
	userId uuid.UUID,
	history []llm.Message,
) (*GroundingResult, error) {

	query := resolvedIntent.Query
	if query == "" {
		return nil, fmt.Errorf("search intent requires query")
	}

	// Execute vector search
	config := search.DefaultConfig()
	candidates, err := g.searchOrchestrator.Execute(ctx, uow, userId, query, config)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// No results - generate adaptive message
	if len(candidates) == 0 {
		notFoundMsg := g.generateNotFoundMessage(ctx, query, history)
		return &GroundingResult{
			Context:       nil,
			Session:       session,
			ShouldAnswer:  false,
			BrowseMessage: notFoundMsg,
		}, nil
	}

	// Single result - auto-focus and prepare for answer
	if len(candidates) == 1 {
		doc := candidates[0]

		// Load full content
		note, err := g.loadNoteContent(ctx, uow, doc.ID)
		if err != nil {
			g.logger.Printf("[WARN] Failed to load note content: %v", err)
		} else {
			doc.Content = note.Content
		}

		// Update session
		session.FocusedNote = &doc
		session.Candidates = candidates
		session.State = store.StateFocused

		return &GroundingResult{
			Context: &GroundedContext{
				Notes:      []NoteContent{{ID: doc.ID, Title: doc.Title, Content: doc.Content}},
				Candidates: candidates,
				Scope:      intent.ScopeSingle,
				FocusedID:  doc.ID,
				FocusIndex: 1, // First and only result
				IDs:        []string{doc.ID},
			},
			Session:      session,
			ShouldAnswer: true,
		}, nil
	}

	// [Legacy] Heuristic match disabled in favor of LLM Semantic Filter (Conductor Pattern)
	/*
		// Multiple results - check if query is specific enough to auto-select
		if specificIdx := g.detectSpecificMatch(query, candidates); specificIdx >= 0 {
			// Specific match found - auto-focus
			g.logger.Printf("[GROUNDING] Specific match found at index %d, auto-focusing", specificIdx)
			doc := candidates[specificIdx]

			note, err := g.loadNoteContent(ctx, uow, doc.ID)
			if err != nil {
				g.logger.Printf("[WARN] Failed to load note content: %v", err)
			} else {
				doc.Content = note.Content
			}

			session.FocusedNote = &doc
			session.Candidates = candidates
			session.State = store.StateFocused

			return &GroundingResult{
				Context: &GroundedContext{
					Notes:      []NoteContent{{ID: doc.ID, Title: doc.Title, Content: doc.Content}},
					Candidates: candidates,
					Scope:      intent.ScopeSingle,
					FocusedID:  doc.ID,
					FocusIndex: specificIdx + 1,
					IDs:        []string{doc.ID},
				},
				Session:      session,
				ShouldAnswer: true,
			}, nil
		}
	*/

	// [Semantic Filter] Filter candidates by relevance using LLM
	relevantIndices, err := g.evaluateRelevance(ctx, query, candidates)
	if err == nil && len(relevantIndices) != len(candidates) {
		g.logger.Printf("[GROUNDING] Semantic filter reduced candidates from %d to %d", len(candidates), len(relevantIndices))

		if len(relevantIndices) == 0 {
			// All filtered out
			notFoundMsg := g.generateNotFoundMessage(ctx, query, history)
			return &GroundingResult{
				Context:       nil,
				Session:       session,
				ShouldAnswer:  false,
				BrowseMessage: notFoundMsg,
			}, nil
		}

		// Update candidates to only relevant ones
		var filtered []store.Document
		for _, idx := range relevantIndices {
			if idx >= 0 && idx < len(candidates) {
				filtered = append(filtered, candidates[idx])
			}
		}
		candidates = filtered

		// If reduced to single candidate -> Auto Focus
		if len(candidates) == 1 {
			doc := candidates[0]
			g.logger.Printf("[GROUNDING] Semantic filter identified single relevant note: %s", doc.Title)

			note, err := g.loadNoteContent(ctx, uow, doc.ID)
			if err != nil {
				g.logger.Printf("[WARN] Failed to load note content: %v", err)
			} else {
				doc.Content = note.Content
			}

			session.FocusedNote = &doc
			session.Candidates = candidates
			session.State = store.StateFocused

			return &GroundingResult{
				Context: &GroundedContext{
					Notes:      []NoteContent{{ID: doc.ID, Title: doc.Title, Content: doc.Content}},
					Candidates: candidates,
					Scope:      intent.ScopeSingle,
					FocusedID:  doc.ID,
					FocusIndex: 1,
					IDs:        []string{doc.ID},
				},
				Session:      session,
				ShouldAnswer: true,
			}, nil
		}
	}

	// === EXPLICITNESS-AWARE GROUNDING ===
	// If user's prompt is explicit (HIGH), don't browse - aggregate and answer directly
	if resolvedIntent.Explicitness == intent.ExplicitnessHigh {
		g.logger.Printf("[GROUNDING] Explicit prompt (HIGH) → Auto-aggregating %d notes", len(candidates))

		// Load full content for all candidates
		var notes []NoteContent
		var ids []string
		for _, cand := range candidates {
			note, err := g.loadNoteContent(ctx, uow, cand.ID)
			if err != nil {
				g.logger.Printf("[WARN] Failed to load note %s: %v", cand.ID, err)
				continue
			}
			notes = append(notes, NoteContent{
				ID:      cand.ID,
				Title:   note.Title,
				Content: note.Content,
			})
			ids = append(ids, cand.ID)
		}

		session.Candidates = candidates
		session.FocusedNote = &store.Document{ID: "aggregated", Title: "All Notes"}
		session.State = store.StateFocused

		return &GroundingResult{
			Context: &GroundedContext{
				Notes:      notes,
				Candidates: candidates,
				Scope:      intent.ScopeAll,
				FocusedID:  "aggregated",
				IDs:        ids,
			},
			Session:      session,
			ShouldAnswer: true,
		}, nil
	}

	// Multiple results with ambiguity (MEDIUM/LOW explicitness) - generate adaptive clarification message
	session.Candidates = candidates
	session.FocusedNote = nil
	session.State = store.StateBrowsing

	ambiguityMsg := g.generateAmbiguityMessage(ctx, query, candidates, history)

	g.logger.Printf("[GROUNDING] Search found %d candidates, Explicitness=%s, entering BROWSE mode",
		len(candidates), resolvedIntent.Explicitness)

	return &GroundingResult{
		Context:       nil,
		Session:       session,
		ShouldAnswer:  false,
		BrowseMessage: ambiguityMsg,
	}, nil
}

// groundFocus loads a specific note based on target index
func (g *Grounder) groundFocus(
	ctx context.Context,
	resolvedIntent *intent.Intent,
	session *store.Session,
	uow unitofwork.UnitOfWork,
	history []llm.Message,
) (*GroundingResult, error) {

	target := resolvedIntent.Target

	// Validate target
	if target < 0 || target >= len(session.Candidates) {
		invalidMsg := g.generateInvalidSelectionMessage(ctx, len(session.Candidates), history)
		return &GroundingResult{
			Context:       nil,
			Session:       session,
			ShouldAnswer:  false,
			BrowseMessage: invalidMsg,
		}, nil
	}

	doc := session.Candidates[target]

	g.logger.Printf("[GROUNDING] Attempting to focus on note index %d (ID: %s)", target, doc.ID)

	// Load full content
	note, err := g.loadNoteContent(ctx, uow, doc.ID)
	if err != nil {
		g.logger.Printf("[ERROR] Failed to load full note content for ID %s: %v", doc.ID, err)

		return &GroundingResult{
			Context:       nil,
			Session:       session,
			ShouldAnswer:  false,
			BrowseMessage: fmt.Sprintf("I found the note '%s' but encountered an error loading its full content. Please try searching for it specifically.", doc.Title),
		}, nil
	}

	// Success - Update Doc
	doc.Content = note.Content
	doc.Title = note.Title

	// Update session - FOCUS on this specific note
	session.FocusedNote = &doc
	session.State = store.StateFocused

	g.logger.Printf("[GROUNDING] Focused on note %d: %s", target, doc.Title)

	return &GroundingResult{
		Context: &GroundedContext{
			Notes:     []NoteContent{{ID: doc.ID, Title: doc.Title, Content: doc.Content}},
			Scope:     intent.ScopeSingle,
			FocusedID: doc.ID,
			IDs:       []string{doc.ID},
		},
		Session:      session,
		ShouldAnswer: true,
	}, nil
}

// groundAggregate loads ALL candidates for aggregation
func (g *Grounder) groundAggregate(
	ctx context.Context,
	resolvedIntent *intent.Intent,
	session *store.Session,
	uow unitofwork.UnitOfWork,
) (*GroundingResult, error) {

	candidates := session.Candidates

	// If no candidates exist (new session or first query), we need to search first
	if len(candidates) == 0 {
		g.logger.Printf("[GROUNDING] AGGREGATE intent but no candidates - performing search first")

		// Use the intent query to search, or fall back to a generic financial query
		searchQuery := resolvedIntent.Query
		if searchQuery == "" {
			searchQuery = resolvedIntent.Reasoning // Use reasoning as context
		}

		// We need userId to search, try to get from session
		userId, err := uuid.Parse(session.UserID)
		if err != nil {
			return &GroundingResult{
				Context:       nil,
				Session:       session,
				ShouldAnswer:  false,
				BrowseMessage: "Unable to identify user for search.",
			}, nil
		}

		// Execute vector search
		config := search.DefaultConfig()
		searchResults, err := g.searchOrchestrator.Execute(ctx, uow, userId, searchQuery, config)
		if err != nil || len(searchResults) == 0 {
			return &GroundingResult{
				Context:       nil,
				Session:       session,
				ShouldAnswer:  false,
				BrowseMessage: "No relevant notes found for aggregation.",
			}, nil
		}

		// Apply semantic filter to exclude irrelevant notes
		relevantIndices, filterErr := g.evaluateRelevance(ctx, searchQuery, searchResults)
		if filterErr == nil && len(relevantIndices) != len(searchResults) {
			// Handle case where filter says none are relevant
			if len(relevantIndices) == 0 {
				g.logger.Printf("[GROUNDING] Semantic filter rejected all %d candidates", len(searchResults))
				return &GroundingResult{
					Context:       nil,
					Session:       session,
					ShouldAnswer:  false,
					BrowseMessage: "No relevant notes found for aggregation.",
				}, nil
			}

			g.logger.Printf("[GROUNDING] Semantic filter reduced candidates from %d to %d for aggregation",
				len(searchResults), len(relevantIndices))
			var filtered []store.Document
			for _, idx := range relevantIndices {
				if idx >= 0 && idx < len(searchResults) {
					filtered = append(filtered, searchResults[idx])
				}
			}
			searchResults = filtered
		}

		candidates = searchResults
		session.Candidates = candidates
		g.logger.Printf("[GROUNDING] Final: %d candidates for aggregation", len(candidates))
	}

	// Load full content for ALL candidates
	var notes []NoteContent
	var ids []string
	var aggregatedContent strings.Builder

	for i, cand := range candidates {
		note, err := g.loadNoteContent(ctx, uow, cand.ID)
		if err != nil {
			g.logger.Printf("[WARN] Failed to load note %s: %v", cand.ID, err)
			continue
		}

		notes = append(notes, NoteContent{
			ID:      cand.ID,
			Title:   note.Title,
			Content: note.Content,
		})
		ids = append(ids, cand.ID)

		aggregatedContent.WriteString(fmt.Sprintf("--- Note %d: %s ---\n", i+1, note.Title))
		aggregatedContent.WriteString(note.Content)
		aggregatedContent.WriteString("\n\n")
	}

	// Update session with aggregated content
	session.FocusedNote = &store.Document{
		ID:      "aggregated",
		Title:   "All Notes",
		Content: aggregatedContent.String(),
	}
	session.State = store.StateFocused

	g.logger.Printf("[GROUNDING] Aggregated %d notes", len(notes))

	return &GroundingResult{
		Context: &GroundedContext{
			Notes:      notes,
			Candidates: candidates,
			Scope:      intent.ScopeAll,
			FocusedID:  "aggregated",
			IDs:        ids,
		},
		Session:      session,
		ShouldAnswer: true,
	}, nil
}

// groundAnswer uses existing focused context
func (g *Grounder) groundAnswer(
	ctx context.Context,
	session *store.Session,
	uow unitofwork.UnitOfWork,
) (*GroundingResult, error) {

	// Must have focused note
	if session.FocusedNote == nil {
		// Fallback to browse mode if we have candidates
		if len(session.Candidates) > 0 {
			return g.groundBrowse(ctx, session, nil) // No history available here
		}
		return &GroundingResult{
			Context:       nil,
			Session:       session,
			ShouldAnswer:  false,
			BrowseMessage: "I seem to have lost the context. Could you please search again?",
		}, nil
	}

	// If aggregated, return all candidate IDs
	if session.FocusedNote.ID == "aggregated" {
		var ids []string
		var notes []NoteContent
		for _, c := range session.Candidates {
			ids = append(ids, c.ID)
			notes = append(notes, NoteContent{ID: c.ID, Title: c.Title, Content: c.Content})
		}

		return &GroundingResult{
			Context: &GroundedContext{
				Notes:      notes,
				Candidates: session.Candidates,
				Scope:      intent.ScopeAll,
				FocusedID:  "aggregated",
				IDs:        ids,
			},
			Session:      session,
			ShouldAnswer: true,
		}, nil
	}

	// Single focused note - ensure we have full content
	doc := session.FocusedNote
	if doc.Content == "" {
		note, err := g.loadNoteContent(ctx, uow, doc.ID)
		if err == nil {
			doc.Content = note.Content
			doc.Title = note.Title
		}
	}

	// Find index in candidates for semantic continuity
	index := 0
	for i, c := range session.Candidates {
		if c.ID == doc.ID {
			index = i + 1
			break
		}
	}

	return &GroundingResult{
		Context: &GroundedContext{
			Notes:      []NoteContent{{ID: doc.ID, Title: doc.Title, Content: doc.Content}},
			Candidates: session.Candidates,
			Scope:      intent.ScopeSingle,
			FocusedID:  doc.ID,
			FocusIndex: index,
			IDs:        []string{doc.ID},
		},
		Session:      session,
		ShouldAnswer: true,
	}, nil
}

// groundBrowse returns the candidate list without answering
func (g *Grounder) groundBrowse(ctx context.Context, session *store.Session, history []llm.Message) (*GroundingResult, error) {
	if len(session.Candidates) == 0 {
		// No candidates - use adaptive message
		noNotesMsg := "No notes are available. Please search for something first."
		if g.llmProvider != nil {
			noNotesMsg = g.generateClarifyMessage(ctx, history)
		}
		return &GroundingResult{
			Context:       nil,
			Session:       session,
			ShouldAnswer:  false,
			BrowseMessage: noNotesMsg,
		}, nil
	}

	// Generate adaptive browse message
	browseMsg := g.generateBrowseMessage(ctx, session.Candidates, history)

	return &GroundingResult{
		Context:       nil,
		Session:       session,
		ShouldAnswer:  false,
		BrowseMessage: browseMsg,
	}, nil
}

func (g *Grounder) groundClarify(ctx context.Context, session *store.Session, history []llm.Message) (*GroundingResult, error) {
	clarifyMsg := g.generateClarifyMessage(ctx, history)
	return &GroundingResult{
		Context:       nil,
		Session:       session,
		ShouldAnswer:  false,
		BrowseMessage: clarifyMsg,
	}, nil
}

// groundMetaAnalysis prepares context for history-based questions
func (g *Grounder) groundMetaAnalysis(session *store.Session) (*GroundingResult, error) {
	// For meta-analysis, we don't need note content, just permission to answer from history
	return &GroundingResult{
		Context: &GroundedContext{
			Notes:     []NoteContent{},
			Scope:     intent.ScopeNone,
			FocusedID: "",
			IDs:       []string{},
		},
		Session:      session,
		ShouldAnswer: true,
	}, nil
}

// loadNoteContent loads full note content from database
func (g *Grounder) loadNoteContent(ctx context.Context, uow unitofwork.UnitOfWork, noteID string) (*NoteContent, error) {
	nid, err := uuid.Parse(noteID)
	if err != nil {
		return nil, fmt.Errorf("invalid note ID: %w", err)
	}

	note, err := uow.NoteRepository().FindOne(ctx, specification.ByID{ID: nid})
	if err != nil {
		return nil, fmt.Errorf("note not found: %w", err)
	}
	if note == nil {
		return nil, fmt.Errorf("note not found")
	}

	return &NoteContent{
		ID:      noteID,
		Title:   note.Title,
		Content: lexical.ParseContent(note.Content),
	}, nil
}

// ============================================================================
// ADAPTIVE MESSAGE GENERATION
// These methods generate context-aware, language-adaptive messages using LLM
// English fallbacks are used when LLM is unavailable
// ============================================================================

// detectSpecificMatch checks if the query matches a specific note title
// Returns the index of the matching note, or -1 if no specific match
func (g *Grounder) detectSpecificMatch(query string, candidates []store.Document) int {
	queryLower := strings.ToLower(strings.TrimSpace(query))

	// Check for exact title match
	for i, c := range candidates {
		titleLower := strings.ToLower(c.Title)
		if queryLower == titleLower {
			g.logger.Printf("[SPECIFICITY] Exact title match: '%s'", c.Title)
			return i
		}
	}

	// Check if query contains specific identifiers (Q1, Chapter 3, etc.)
	specificPatterns := []string{"q1", "q2", "q3", "q4", "q5", "chapter", "section", "part "}
	for _, pattern := range specificPatterns {
		if strings.Contains(queryLower, pattern) {
			// Query is specific but doesn't match a single title - check if it narrows to one
			for i, c := range candidates {
				if strings.Contains(strings.ToLower(c.Title), pattern) {
					g.logger.Printf("[SPECIFICITY] Pattern '%s' matches single note: '%s'", pattern, c.Title)
					return i
				}
			}
		}
	}

	// Check if query is a substantial substring of only one title
	matchCount := 0
	matchIdx := -1
	for i, c := range candidates {
		titleLower := strings.ToLower(c.Title)
		if strings.Contains(titleLower, queryLower) && len(queryLower) > 5 {
			matchCount++
			matchIdx = i
		}
	}
	if matchCount == 1 {
		g.logger.Printf("[SPECIFICITY] Query is unique substring of: '%s'", candidates[matchIdx].Title)
		return matchIdx
	}

	return -1 // Ambiguous
}

// generateAmbiguityMessage creates a clarification message using LLM
func (g *Grounder) generateAmbiguityMessage(ctx context.Context, query string, candidates []store.Document, history []llm.Message) string {
	if g.llmProvider == nil {
		return g.fallbackAmbiguityMessage(candidates)
	}

	var candidateList strings.Builder
	for i, c := range candidates {
		candidateList.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}

	prompt := fmt.Sprintf(`Generate a brief clarification message. The user searched for "%s" and found %d notes.

Notes found:
%s

Requirements:
1. Match the user's language
2. Be concise (2-3 sentences max before list)
3. Ask them to pick one or confirm "all"
4. Keep the numbered list format

Respond with ONLY the message:`, query, len(candidates), candidateList.String())

	response, err := g.llmProvider.Generate(ctx, prompt)
	if err != nil {
		g.logger.Printf("[WARN] LLM ambiguity message failed: %v", err)
		return g.fallbackAmbiguityMessage(candidates)
	}

	return response
}

// generateNotFoundMessage creates a not-found message using LLM
func (g *Grounder) generateNotFoundMessage(ctx context.Context, query string, history []llm.Message) string {
	if g.llmProvider == nil {
		return "I couldn't find any notes matching your search. Try different keywords."
	}

	prompt := fmt.Sprintf(`Generate a brief "not found" message. The user searched for "%s" but no notes were found.

Requirements:
1. Match the user's language (detect from the query)
2. Be helpful and suggest trying different keywords
3. Keep it to 1-2 sentences max

Respond with ONLY the message:`, query)

	response, err := g.llmProvider.Generate(ctx, prompt)
	if err != nil {
		g.logger.Printf("[WARN] LLM not-found message failed: %v", err)
		return "I couldn't find any notes matching your search. Try different keywords."
	}

	return response
}

// generateInvalidSelectionMessage creates an invalid selection message using LLM
func (g *Grounder) generateInvalidSelectionMessage(ctx context.Context, maxOptions int, history []llm.Message) string {
	fallback := fmt.Sprintf("Invalid selection. Please choose a number between 1 and %d.", maxOptions)

	if g.llmProvider == nil {
		return fallback
	}

	prompt := fmt.Sprintf(`Generate a brief "invalid selection" message. Valid options are 1 to %d.

Requirements:
1. Match the conversation language
2. Be helpful, not scolding
3. Keep it to 1 sentence

Respond with ONLY the message:`, maxOptions)

	response, err := g.llmProvider.Generate(ctx, prompt)
	if err != nil {
		g.logger.Printf("[WARN] LLM invalid selection message failed: %v", err)
		return fallback
	}

	return response
}

// generateBrowseMessage creates a browse menu message using LLM
func (g *Grounder) generateBrowseMessage(ctx context.Context, candidates []store.Document, history []llm.Message) string {
	if g.llmProvider == nil {
		return g.fallbackBrowseMessage(candidates)
	}

	var candidateList strings.Builder
	for i, c := range candidates {
		candidateList.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}

	prompt := fmt.Sprintf(`Generate a brief message listing available notes.

Notes:
%s

Requirements:
1. Match the conversation language
2. Be concise (1 sentence intro)
3. Keep the numbered list format

Respond with ONLY the message:`, candidateList.String())

	response, err := g.llmProvider.Generate(ctx, prompt)
	if err != nil {
		g.logger.Printf("[WARN] LLM browse message failed: %v", err)
		return g.fallbackBrowseMessage(candidates)
	}

	return response
}

// generateClarifyMessage creates a clarification request using LLM
func (g *Grounder) generateClarifyMessage(ctx context.Context, history []llm.Message) string {
	fallback := "Could you provide more details about what you're looking for?"

	if g.llmProvider == nil {
		return fallback
	}

	prompt := `Generate a brief clarification request asking the user to provide more details.

Requirements:
1. Match the conversation language
2. Be polite and helpful
3. Keep it to 1 sentence

Respond with ONLY the message:`

	response, err := g.llmProvider.Generate(ctx, prompt)
	if err != nil {
		g.logger.Printf("[WARN] LLM clarify message failed: %v", err)
		return fallback
	}

	return response
}

// evaluateRelevance uses LLM to filter candidates by semantic relevance
// Returns 0-based indices of relevant candidates
func (g *Grounder) evaluateRelevance(ctx context.Context, query string, candidates []store.Document) ([]int, error) {
	if g.llmProvider == nil {
		// Fallback: return all indices
		var all []int
		for i := range candidates {
			all = append(all, i)
		}
		return all, nil
	}

	var sb strings.Builder
	for i, c := range candidates {
		preview := c.Content
		// Parse Lexical JSON if possible to give LLM readable text
		if parsed := lexical.ParseContent(preview); parsed != "" {
			preview = parsed
		}

		if len(preview) > 1000 {
			preview = preview[:1000] + "..."
		}
		// Remove newlines for cleaner prompt
		preview = strings.ReplaceAll(preview, "\n", " ")
		sb.WriteString(fmt.Sprintf("%d. Title: %s\n   Preview: %s\n", i+1, c.Title, preview))
	}

	prompt := fmt.Sprintf(`Analyze the relevance of the following notes to the query: "%s"

Notes:
%s

Identify which notes are semantically relevant to the user's request.
Relevance Criteria:
1. The note MUST be about the query topic.
2. Exclude notes that are clearly off-topic or only share a keyword by coincidence (e.g. mentions "exam" but denotes a "class fund").
3. Include notes that are likely relevant.

Respond with ONLY a comma-separated list of the relevant note numbers (e.g., "1, 3").
If none are relevant, respond with "0".`, query, sb.String())

	// Log the prompt for debugging
	g.logger.Printf("[RELEVANCE] Filter Prompt:\n%s", prompt)

	response, err := g.llmProvider.Generate(ctx, prompt, llm.WithTemperature(0.0))
	if err != nil {
		g.logger.Printf("[WARN] LLM relevance eval failed: %v", err)
		return nil, err
	}

	g.logger.Printf("[RELEVANCE] Query: '%s'", query)
	g.logger.Printf("[RELEVANCE] Analyzer Response: '%s'", response)

	// Parse response
	cleanResp := strings.TrimSpace(response)
	cleanResp = strings.Trim(cleanResp, ".") // remove trailing dot if any

	if cleanResp == "0" || strings.EqualFold(cleanResp, "none") {
		return []int{}, nil
	}

	parts := strings.Split(cleanResp, ",")
	var indices []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if val, err := strconv.Atoi(p); err == nil {
			if val > 0 && val <= len(candidates) {
				indices = append(indices, val-1) // Convert 1-based to 0-based
			}
		}
	}

	return indices, nil
}

// Fallback messages (English)
func (g *Grounder) fallbackAmbiguityMessage(candidates []store.Document) string {
	var builder strings.Builder
	builder.WriteString("I found several relevant notes. Which one would you like to focus on?\n")
	for i, c := range candidates {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}
	builder.WriteString("\nOr say 'all' to get information from all of them.")
	return builder.String()
}

func (g *Grounder) fallbackBrowseMessage(candidates []store.Document) string {
	var builder strings.Builder
	builder.WriteString("Here are the available notes:\n")
	for i, c := range candidates {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Title))
	}
	return builder.String()
}
