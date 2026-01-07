package dto

// AIActionPlan represents the strict JSON output from the Planner LLM
type AIActionPlan struct {
	Action      string `json:"action"`       // "SEARCH", "SELECT", "SWITCH", "ANSWER_CURRENT", "ANSWER_ALL", "CLARIFY"
	SearchQuery string `json:"search_query"` // If Action == "SEARCH"
	TargetIndex int    `json:"target_index"` // If Action == "SELECT" or "SWITCH" (0-based)
	Reasoning   string `json:"reasoning"`    // For debugging/logging
}

const (
	PlannerActionSearch        = "SEARCH"
	PlannerActionSelect        = "SELECT"
	PlannerActionSwitch        = "SWITCH"
	PlannerActionAnswerCurrent = "ANSWER_CURRENT"
	PlannerActionAnswerAll     = "ANSWER_ALL" // Aggregate all candidates
	PlannerActionClarify       = "CLARIFY"    // Ask for clarification (LLM decides message)
)
