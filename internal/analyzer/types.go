package analyzer

import "context"

// LLMClient abstracts the Claude API for testing.
type LLMClient interface {
	Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error)
}

type SummaryResult struct {
	Summary       string
	AffectedPaths []string
	Error         error
}

type Risk struct {
	Title         string
	File          string
	Line          int
	Severity      string // "critical", "warning", "suggestion"
	Confidence    string // "high", "medium", "low"
	Description   string
	FixSuggestion string
}

type RiskResult struct {
	Risks []Risk
	Error error
}

type AnalysisResult struct {
	Summary *SummaryResult
	Risks   *RiskResult
}
