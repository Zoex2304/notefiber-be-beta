package router

import (
	"regexp"
	"strings"
)

// ReferenceType indicates how the reference was specified
type ReferenceType string

const (
	ReferenceTypeUUID    ReferenceType = "uuid"
	ReferenceTypeTitle   ReferenceType = "title"
	ReferenceTypePartial ReferenceType = "partial"
)

// ParsedReference represents a single note reference extracted from a prompt
type ParsedReference struct {
	Type        ReferenceType // UUID, TITLE, or PARTIAL
	Value       string        // The identifier (UUID string, title, or partial text)
	Syntax      string        // "@notes:" or "[[]]"
	OriginalRaw string        // The original matched text
}

// ReferenceParseResult contains all parsed references and the cleaned prompt
type ReferenceParseResult struct {
	References  []ParsedReference
	CleanPrompt string // Prompt with all references removed
	HasRefs     bool   // Quick check for any references
}

// UUID regex pattern (standard UUID format)
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Reference patterns:
// @notes:uuid           - Direct UUID reference
// @notes:"quoted title" - Quoted title reference
// @notes:partial        - Partial/fuzzy match
// [[Title]]             - Wiki-style title reference
var (
	// @notes:"quoted title" or @notes:value
	atNotesQuotedPattern = regexp.MustCompile(`@notes:"([^"]+)"`)
	atNotesPlainPattern  = regexp.MustCompile(`@notes:(\S+)`)
	// [[Title]]
	wikiLinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
)

// ParseReferences extracts all note references from a prompt.
// Supports:
//   - @notes:uuid         → Direct UUID lookup
//   - @notes:"Note Title" → Quoted title lookup
//   - @notes:partial      → Partial/fuzzy match
//   - [[Note Title]]      → Wiki-link style reference
//
// Returns the list of parsed references and the cleaned prompt.
func ParseReferences(prompt string) *ReferenceParseResult {
	result := &ReferenceParseResult{
		References:  make([]ParsedReference, 0),
		CleanPrompt: prompt,
	}

	// Track all matches for removal
	var allMatches []string

	// 1. Parse @notes:"quoted title" first (higher priority)
	quotedMatches := atNotesQuotedPattern.FindAllStringSubmatch(prompt, -1)
	for _, match := range quotedMatches {
		if len(match) >= 2 {
			ref := ParsedReference{
				Type:        ReferenceTypeTitle,
				Value:       match[1],
				Syntax:      "@notes:",
				OriginalRaw: match[0],
			}
			result.References = append(result.References, ref)
			allMatches = append(allMatches, match[0])
		}
	}

	// Remove quoted matches from prompt before parsing plain @notes:
	tempPrompt := prompt
	for _, match := range allMatches {
		tempPrompt = strings.Replace(tempPrompt, match, "", 1)
	}

	// 2. Parse @notes:value (unquoted)
	plainMatches := atNotesPlainPattern.FindAllStringSubmatch(tempPrompt, -1)
	for _, match := range plainMatches {
		if len(match) >= 2 {
			value := match[1]
			refType := determineReferenceType(value)
			ref := ParsedReference{
				Type:        refType,
				Value:       value,
				Syntax:      "@notes:",
				OriginalRaw: match[0],
			}
			result.References = append(result.References, ref)
			allMatches = append(allMatches, match[0])
		}
	}

	// 3. Parse [[Title]] wiki-links
	wikiMatches := wikiLinkPattern.FindAllStringSubmatch(prompt, -1)
	for _, match := range wikiMatches {
		if len(match) >= 2 {
			ref := ParsedReference{
				Type:        ReferenceTypeTitle,
				Value:       match[1],
				Syntax:      "[[]]",
				OriginalRaw: match[0],
			}
			result.References = append(result.References, ref)
			allMatches = append(allMatches, match[0])
		}
	}

	// 4. Build clean prompt by removing all matches
	cleanPrompt := prompt
	for _, match := range allMatches {
		cleanPrompt = strings.Replace(cleanPrompt, match, "", 1)
	}

	// Normalize whitespace in clean prompt
	cleanPrompt = strings.TrimSpace(cleanPrompt)
	cleanPrompt = regexp.MustCompile(`\s+`).ReplaceAllString(cleanPrompt, " ")

	result.CleanPrompt = cleanPrompt
	result.HasRefs = len(result.References) > 0

	return result
}

// determineReferenceType classifies a reference value
func determineReferenceType(value string) ReferenceType {
	// Check if it's a valid UUID
	if uuidPattern.MatchString(value) {
		return ReferenceTypeUUID
	}
	// Otherwise, treat as partial match
	return ReferenceTypePartial
}

// MaxReferences is the hard limit for references in a single prompt
const MaxReferences = 5

// ValidateReferences checks if references are within limits
func ValidateReferences(refs []ParsedReference) error {
	if len(refs) > MaxReferences {
		return ErrTooManyReferences{}
	}
	return nil
}

// ErrTooManyReferences is returned when more than MaxReferences are provided
type ErrTooManyReferences struct{}

func (e ErrTooManyReferences) Error() string {
	return "too many note references: maximum 5 allowed"
}
