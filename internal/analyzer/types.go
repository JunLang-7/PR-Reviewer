package analyzer

import "context"

// LLMClient abstracts the Claude API for testing.
type LLMClient interface {
	Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error)
}

// Stage 1 output
type SummaryResult struct {
	Summary       string
	AffectedPaths []string
	Error         error
}

// Stage 2 output
type Risk struct {
	Title       string
	File        string
	Line        int
	Severity    string // "critical", "warning", "suggestion"
	Confidence  string // "high", "medium", "low"
	Description string
	FixSuggestion string
}

type RiskResult struct {
	Risks []Risk
	Error error
}

// Stage 3 output
type Suggestion struct {
	File        string
	Line        int
	Description string
	CodeSnippet string
}

type SuggestionResult struct {
	Suggestions []Suggestion
	Error       error
}

// AnalysisResult collects all pipeline outputs.
type AnalysisResult struct {
	Summary     *SummaryResult
	Risks       *RiskResult
	Suggestions *SuggestionResult
}
