package store

// Document represents a generic note/content structure for the RAG system
type Document struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Session represents the active user session state in memory
type Session struct {
	ID     string `json:"id"` // ChatSessionID
	UserID string `json:"user_id"`
	State  string `json:"state"` // "BROWSING" | "FOCUSED"
	Mode   string `json:"mode"`  // "RAG" | "BYPASS" | "NUANCE" - Pipeline mode for the session

	// THE WAITING ROOM (Candidates found but not yet selected)
	Candidates []Document `json:"candidates"`

	// THE WORKBENCH (The active note being discussed)
	FocusedNote *Document `json:"focused_note"`

	// Metadata for last interaction
	LastQuery string `json:"last_query"`
}

const (
	StateBrowsing = "BROWSING"
	StateFocused  = "FOCUSED"

	// Pipeline modes
	ModeRAG    = "RAG"
	ModeBypass = "BYPASS"
	ModeNuance = "NUANCE"
)
