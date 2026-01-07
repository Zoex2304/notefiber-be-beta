package router

import (
	"testing"
)

func TestParseReferences(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		wantRefCount    int
		wantCleanPrompt string
		wantHasRefs     bool
	}{
		{
			name:            "no references",
			prompt:          "What is machine learning?",
			wantRefCount:    0,
			wantCleanPrompt: "What is machine learning?",
			wantHasRefs:     false,
		},
		{
			name:            "single UUID reference",
			prompt:          "@notes:abc12345-1234-1234-1234-123456789abc Explain this",
			wantRefCount:    1,
			wantCleanPrompt: "Explain this",
			wantHasRefs:     true,
		},
		{
			name:            "quoted title reference",
			prompt:          "@notes:\"Meeting Notes\" Summarize",
			wantRefCount:    1,
			wantCleanPrompt: "Summarize",
			wantHasRefs:     true,
		},
		{
			name:            "wiki-link reference",
			prompt:          "[[Recipe Book]] List ingredients",
			wantRefCount:    1,
			wantCleanPrompt: "List ingredients",
			wantHasRefs:     true,
		},
		{
			name:            "multiple references",
			prompt:          "@notes:abc @notes:\"Title\" [[Wiki]] Compare",
			wantRefCount:    3,
			wantCleanPrompt: "Compare",
			wantHasRefs:     true,
		},
		{
			name:            "partial reference",
			prompt:          "@notes:partial_search Explain",
			wantRefCount:    1,
			wantCleanPrompt: "Explain",
			wantHasRefs:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseReferences(tt.prompt)

			if len(result.References) != tt.wantRefCount {
				t.Errorf("RefCount = %d, want %d", len(result.References), tt.wantRefCount)
			}

			if result.CleanPrompt != tt.wantCleanPrompt {
				t.Errorf("CleanPrompt = %q, want %q", result.CleanPrompt, tt.wantCleanPrompt)
			}

			if result.HasRefs != tt.wantHasRefs {
				t.Errorf("HasRefs = %v, want %v", result.HasRefs, tt.wantHasRefs)
			}
		})
	}
}

func TestDetermineReferenceType(t *testing.T) {
	tests := []struct {
		value    string
		wantType ReferenceType
	}{
		{"abc12345-1234-1234-1234-123456789abc", ReferenceTypeUUID},
		{"My Note Title", ReferenceTypePartial},
		{"partial_search", ReferenceTypePartial},
		{"ABC12345-1234-1234-1234-123456789ABC", ReferenceTypeUUID}, // uppercase UUID
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := determineReferenceType(tt.value)
			if got != tt.wantType {
				t.Errorf("determineReferenceType(%q) = %v, want %v", tt.value, got, tt.wantType)
			}
		})
	}
}

func TestValidateReferences(t *testing.T) {
	// Test within limit
	refs := make([]ParsedReference, 3)
	err := ValidateReferences(refs)
	if err != nil {
		t.Errorf("ValidateReferences with 3 refs should not error, got: %v", err)
	}

	// Test at limit
	refs = make([]ParsedReference, 5)
	err = ValidateReferences(refs)
	if err != nil {
		t.Errorf("ValidateReferences with 5 refs should not error, got: %v", err)
	}

	// Test over limit
	refs = make([]ParsedReference, 6)
	err = ValidateReferences(refs)
	if err == nil {
		t.Error("ValidateReferences with 6 refs should error")
	}
}
