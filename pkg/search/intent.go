package search

import (
	"strings"
)

type SearchStrategy string

const (
	StrategyLiteral  SearchStrategy = "literal"
	StrategySemantic SearchStrategy = "semantic"
)

// DetermineStrategy analyzes the query to decide between literal and semantic search
func DetermineStrategy(query string) SearchStrategy {
	query = strings.TrimSpace(query)

	// 1. Check for specific patterns (e.g., "Note/Title", "Tag:...", "id:...")
	// If the user uses a slash, colon, or specific structured separators, they likely want a literal match.
	if strings.ContainsAny(query, "/:=") {
		return StrategyLiteral
	}

	// 2. Check for short specific keywords
	// Very short queries (e.g., "car", "shoes", "RGB") are often literal lookups.
	// We'll set a threshold, e.g., <= 3 characters, or if it looks like a specific code (all caps, numbers).
	if len(query) <= 3 {
		return StrategyLiteral
	}

	// 3. Check if query is enclosed in quotes (explicit literal request)
	if strings.HasPrefix(query, "\"") && strings.HasSuffix(query, "\"") {
		return StrategyLiteral
	}

	// 4. Default to Semantic logic for explorative queries
	return StrategySemantic
}
