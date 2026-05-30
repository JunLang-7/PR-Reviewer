package analyzer

import _ "embed"

//go:embed prompts/summary.txt
var systemPromptSummary string

//go:embed prompts/risk.txt
var systemPromptRisk string
