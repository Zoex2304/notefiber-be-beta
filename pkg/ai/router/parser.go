package router

import (
	"strings"
)

// Prefix constants - ORDER MATTERS for parsing (check longer prefix first)
const (
	PrefixBypassNuance = "/bypass/nuance:" // Combined: bypass + nuance
	PrefixBypass       = "/bypass"
	PrefixNuance       = "/nuance:"
)

// Mode represents the pipeline routing mode
type Mode string

const (
	ModeRAG          Mode = "RAG"           // Default RAG mode
	ModeBypass       Mode = "BYPASS"        // Pure LLM without RAG
	ModeBypassNuance Mode = "BYPASS_NUANCE" // Pure LLM with nuance injection
	ModeRAGNuance    Mode = "RAG_NUANCE"    // RAG with nuance injection
)

// ParsedPrompt contains routing information extracted from prompt
type ParsedPrompt struct {
	OriginalPrompt string // Full original prompt
	CleanPrompt    string // Prompt without prefix
	Mode           Mode   // BYPASS, BYPASS_NUANCE, RAG_NUANCE, or RAG
	NuanceKey      string // If mode includes nuance, the nuance key (e.g., "engineering")
}

// Parse extracts routing information from prompt
// Supports:
//   - /bypass/nuance:key <prompt> → Bypass pipeline with nuance injection
//   - /bypass <prompt> → Pure LLM mode without RAG
//   - /nuance:key <prompt> → RAG mode with nuance injection
//   - <prompt> → Default RAG mode
func Parse(prompt string) *ParsedPrompt {
	trimmed := strings.TrimSpace(prompt)
	lower := strings.ToLower(trimmed)

	// 1. Check for /bypass/nuance:key (combined prefix - must check FIRST)
	if strings.HasPrefix(lower, PrefixBypassNuance) {
		rest := trimmed[len(PrefixBypassNuance):]
		nuanceKey, cleanPrompt := extractKeyAndPrompt(rest)
		if nuanceKey != "" {
			return &ParsedPrompt{
				OriginalPrompt: prompt,
				CleanPrompt:    cleanPrompt,
				Mode:           ModeBypassNuance,
				NuanceKey:      nuanceKey,
			}
		}
	}

	// 2. Check for /bypass (pure bypass without nuance)
	if strings.HasPrefix(lower, PrefixBypass) {
		rest := trimmed[len(PrefixBypass):]
		if rest == "" || rest[0] == ' ' || rest[0] == '/' {
			// If followed by /nuance:, we already handled above
			if !strings.HasPrefix(strings.ToLower(rest), "/nuance:") {
				return &ParsedPrompt{
					OriginalPrompt: prompt,
					CleanPrompt:    strings.TrimSpace(rest),
					Mode:           ModeBypass,
				}
			}
		}
	}

	// 3. Check for /nuance:key (RAG mode with nuance)
	if strings.HasPrefix(lower, PrefixNuance) {
		rest := trimmed[len(PrefixNuance):]
		nuanceKey, cleanPrompt := extractKeyAndPrompt(rest)
		if nuanceKey != "" {
			return &ParsedPrompt{
				OriginalPrompt: prompt,
				CleanPrompt:    cleanPrompt,
				Mode:           ModeRAGNuance,
				NuanceKey:      nuanceKey,
			}
		}
	}

	// 4. Default: RAG mode
	return &ParsedPrompt{
		OriginalPrompt: prompt,
		CleanPrompt:    prompt,
		Mode:           ModeRAG,
	}
}

// extractKeyAndPrompt splits "key prompt" into (key, prompt)
func extractKeyAndPrompt(rest string) (string, string) {
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx == -1 {
		// No space: entire rest is the nuance key, no prompt
		return strings.ToLower(rest), ""
	}
	// Space found: key before space, prompt after
	return strings.ToLower(rest[:spaceIdx]), strings.TrimSpace(rest[spaceIdx+1:])
}

// IsEmpty returns true if the clean prompt is empty
func (p *ParsedPrompt) IsEmpty() bool {
	return strings.TrimSpace(p.CleanPrompt) == ""
}
