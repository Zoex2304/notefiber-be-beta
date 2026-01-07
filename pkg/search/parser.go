package search

import (
	"strings"
)

// SearchFilters holds the extracted filters and the remaining clean query
type SearchFilters struct {
	NotebookName string
	NoteTitle    string
	SearchQuery  string // The remaining text to search in Content/Title
}

// ParseQuery extracts slash commands from the raw query string
// Supported:
// /nb:<term> OR /in:<term> -> Filter by Notebook Name
// /note:<term> -> Filter by Note Title
// <text> -> Remaining text is the SearchQuery
func ParseQuery(raw string) SearchFilters {
	filters := SearchFilters{}
	parts := strings.Fields(raw)
	var cleanParts []string

	for _, part := range parts {
		lowerPart := strings.ToLower(part)

		if strings.HasPrefix(lowerPart, "/nb:") {
			filters.NotebookName = strings.TrimPrefix(lowerPart, "/nb:")
		} else if strings.HasPrefix(lowerPart, "/in:") {
			// Alias for /nb:
			filters.NotebookName = strings.TrimPrefix(lowerPart, "/in:")
		} else if strings.HasPrefix(lowerPart, "/note:") {
			filters.NoteTitle = strings.TrimPrefix(lowerPart, "/note:")
		} else {
			cleanParts = append(cleanParts, part)
		}
	}

	filters.SearchQuery = strings.Join(cleanParts, " ")
	return filters
}
