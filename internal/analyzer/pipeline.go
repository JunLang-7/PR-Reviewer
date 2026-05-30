package analyzer

import (
	"context"
	"fmt"
	"strings"
)

type PipelineInput struct {
	Diff         string
	FileContents map[string]string
}

type Pipeline struct {
	llm        LLMClient
	modelFast  string
	modelPower string
}

func NewPipeline(llm LLMClient, modelFast, modelPower string) *Pipeline {
	return &Pipeline{llm: llm, modelFast: modelFast, modelPower: modelPower}
}

func (p *Pipeline) Run(ctx context.Context, input PipelineInput) (*AnalysisResult, error) {
	result := &AnalysisResult{}

	summary, err := p.runSummary(ctx, input)
	result.Summary = &SummaryResult{Summary: summary}
	if err != nil {
		result.Summary.Error = err
	}

	risks, err := p.runRiskScan(ctx, input)
	if err != nil {
		result.Risks = &RiskResult{Error: err}
	} else {
		result.Risks = &RiskResult{Risks: risks}
	}

	return result, nil
}

func (p *Pipeline) runSummary(ctx context.Context, input PipelineInput) (string, error) {
	prompt := input.buildSummaryPrompt()
	resp, err := p.llm.Chat(ctx, p.modelFast, systemPromptSummary, prompt)
	if err != nil {
		return "", fmt.Errorf("stage 1 summary: %w", err)
	}
	return resp, nil
}

func (p *Pipeline) runRiskScan(ctx context.Context, input PipelineInput) ([]Risk, error) {
	prompt := input.buildRiskPrompt()
	resp, err := p.llm.Chat(ctx, p.modelPower, systemPromptRisk, prompt)
	if err != nil {
		return nil, fmt.Errorf("stage 2 risk scan: %w", err)
	}
	return parseRiskResponse(resp), nil
}

func codeBlock(lang, content string) string {
	return "```" + lang + "\n" + content + "\n```\n"
}

func (input PipelineInput) buildSummaryPrompt() string {
	var sb strings.Builder
	sb.WriteString("请用中文简要总结以下 PR 的变更内容和影响范围：\n\n")

	sb.WriteString("## 变更文件\n")
	for path := range input.FileContents {
		sb.WriteString(fmt.Sprintf("- %s\n", path))
	}

	sb.WriteString("\n## Diff\n")
	sb.WriteString(codeBlock("diff", truncate(input.Diff, 3000)))

	return sb.String()
}

func (input PipelineInput) buildRiskPrompt() string {
	var sb strings.Builder
	sb.WriteString("请分析以下 PR 变更中的潜在风险：\n\n")

	sb.WriteString("## 变更文件 Diff\n")
	sb.WriteString(codeBlock("diff", truncate(input.Diff, 5000)))

	sb.WriteString("\n## 变更文件完整内容\n")
	for path, content := range input.FileContents {
		sb.WriteString(fmt.Sprintf("\n### %s\n", path))
		sb.WriteString(codeBlock("", truncate(content, 3000)))
	}

	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

// parseRiskResponse parses flat Copilot-style comments separated by ----
// Each block: In `file`:\n\n> code\ncomment...\n严重程度: xxx
func parseRiskResponse(resp string) []Risk {
	if resp == "" {
		return nil
	}

	var risks []Risk
	for _, block := range strings.Split(resp, "\n----") {
		block = strings.TrimSpace(block)
		if block == "" || block == "（无）" || block == "(无)" {
			continue
		}
		r := parseRiskBlock(block)
		if r != nil {
			risks = append(risks, *r)
		}
	}
	return risks
}

// parseRiskBlock parses a single comment block:
// In `file.go`:
//
// > code line (only first line has > for reference)
//
// comment text...
// 严重程度: xxx
func parseRiskBlock(block string) *Risk {
	var codeLines []string
	var commentLines []string
	inCode := false

	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip "In `file`:" header
		if strings.HasPrefix(trimmed, "In `") && strings.Contains(trimmed, "`:") {
			continue
		}

		// Code lines: start with >, >-, >+
		if strings.HasPrefix(trimmed, ">") {
			inCode = true
			codeLines = append(codeLines, trimmed)
			continue
		}
		// Blank line: transition from code to comment
		if inCode && trimmed == "" {
			inCode = false
			continue
		}
		if trimmed != "" {
			commentLines = append(commentLines, trimmed)
		}
	}

	comment := strings.TrimSpace(strings.Join(commentLines, "\n"))
	if comment == "" {
		return nil
	}

	// Extract file from first "In `file`:" line
	file := ""
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "In `") {
			file = extractInFile(strings.TrimSpace(line))
			break
		}
	}

	// Extract severity from "严重程度: xxx" at end of comment
	severity := "warning"
	lines := commentLines
	if len(lines) > 0 {
		last := lines[len(lines)-1]
		for _, s := range []string{"critical", "warning", "suggestion"} {
			if strings.Contains(last, "严重程度: "+s) || strings.Contains(last, "严重程度："+s) {
				severity = s
				comment = strings.TrimSpace(strings.Join(lines[:len(lines)-1], "\n"))
				break
			}
		}
	}

	// Title: first line of comment, truncated
	title := file
	if comment != "" {
		firstLine := strings.SplitN(comment, "\n", 2)[0]
		if len(firstLine) > 80 {
			firstLine = firstLine[:80] + "..."
		}
		title = firstLine
	}

	return &Risk{
		Title:         title,
		File:          file,
		Line:          0,
		Severity:      severity,
		Confidence:    "medium",
		Description:   comment,
		FixSuggestion: strings.TrimSpace(strings.Join(codeLines, "\n")),
	}
}

// extractInFile parses "In `file.go`:" -> "file.go"
func extractInFile(line string) string {
	start := strings.Index(line, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], "`")
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}
