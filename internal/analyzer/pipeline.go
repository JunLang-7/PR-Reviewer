package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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

	risks, err := p.runRiskScan(ctx, input, summary)
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

func (p *Pipeline) runRiskScan(ctx context.Context, input PipelineInput, summary string) ([]Risk, error) {
	prompt := input.buildRiskPrompt(summary)
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

func (input PipelineInput) buildRiskPrompt(summary string) string {
	var sb strings.Builder
	sb.WriteString("请分析以下 PR 变更中的潜在风险：\n\n")

	if summary != "" {
		sb.WriteString("## PR 变更摘要（仅供参考，不得基于此缩小审查范围）\n")
		sb.WriteString("> ")
		sb.WriteString(strings.ReplaceAll(summary, "\n", "\n> "))
		sb.WriteString("\n\n")
	}

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

var fileHeaderRe = regexp.MustCompile(`^\[[^\]]+\]\(ref\)[:：]`)

func parseRiskResponse(resp string) []Risk {
	if resp == "" {
		return nil
	}

	var risks []Risk
	blocks := splitByFileHeader(resp)
	for _, block := range blocks {
		r := parseRiskBlock(block)
		if r != nil {
			risks = append(risks, *r)
		}
	}
	return risks
}

func splitByFileHeader(resp string) []string {
	var blocks []string
	lines := strings.Split(resp, "\n")
	currentStart := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fileHeaderRe.MatchString(trimmed) {
			if i > 0 && currentStart < i {
				blocks = append(blocks, strings.Join(lines[currentStart:i], "\n"))
			}
			currentStart = i
		}
	}
	if currentStart < len(lines) {
		blocks = append(blocks, strings.Join(lines[currentStart:], "\n"))
	}
	return blocks
}

func parseRiskBlock(block string) *Risk {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}

	lines := strings.Split(block, "\n")

	headerIdx := -1
	for i, line := range lines {
		if fileHeaderRe.MatchString(strings.TrimSpace(line)) {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		return nil
	}

	file, lineNum := parseFileHeader(strings.TrimSpace(lines[headerIdx]))

	bodyLines := lines[headerIdx+1:]

	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}

	severity := "warning"
	severityLineIdx := -1
	for i, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		for _, s := range []string{"critical", "warning", "suggestion"} {
			if strings.Contains(trimmed, "严重程度: "+s) || strings.Contains(trimmed, "严重程度："+s) {
				severity = s
				severityLineIdx = i
			}
		}
	}

	bodyEnd := len(bodyLines)
	if severityLineIdx >= 0 {
		bodyEnd = severityLineIdx
	}

	description := strings.Join(bodyLines[:bodyEnd], "\n")

	// Extract FixSuggestion from **建议**： field
	fixSuggestion := ""
	suggestionIdx := -1
	for i, line := range bodyLines[:bodyEnd] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "**建议**：") || strings.HasPrefix(trimmed, "**建议**:") {
			suggestionIdx = i
			suggestionContent := strings.TrimPrefix(trimmed, "**建议**：")
			suggestionContent = strings.TrimPrefix(suggestionContent, "**建议**:")
			fixSuggestion = strings.TrimSpace(suggestionContent)
			// Collect remaining lines after 建议 header
			if i+1 < bodyEnd {
				restLines := bodyLines[i+1 : bodyEnd]
				rest := strings.TrimSpace(strings.Join(restLines, "\n"))
				if rest != "" {
					if fixSuggestion != "" {
						fixSuggestion += "\n"
					}
					fixSuggestion += rest
				}
			}
			break
		}
	}

	// Remove 建议 section from description
	if suggestionIdx >= 0 {
		descLines := bodyLines[:suggestionIdx]
		description = strings.Join(descLines, "\n")
	}

	description = strings.TrimSpace(description)
	if description == "" {
		return nil
	}

	title := file
	firstBodyLine := strings.SplitN(description, "\n", 2)[0]
	if stripped, ok := strings.CutPrefix(firstBodyLine, "**问题**："); ok {
		firstBodyLine = strings.TrimSpace(stripped)
	} else if stripped, ok := strings.CutPrefix(firstBodyLine, "**问题**:"); ok {
		firstBodyLine = strings.TrimSpace(stripped)
	}
	if firstBodyLine != "" && !strings.HasPrefix(firstBodyLine, "**") {
		title = firstBodyLine
	}

	return &Risk{
		Title:         title,
		File:          file,
		Line:          lineNum,
		Severity:      severity,
		Confidence:    "medium",
		Description:   description,
		FixSuggestion: fixSuggestion,
	}
}

func parseFileHeader(line string) (string, int) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "[")
	if idx := strings.Index(line, "](ref)"); idx != -1 {
		line = line[:idx]
	}
	line = strings.TrimSuffix(line, ":")
	line = strings.TrimSuffix(line, "：")
	if idx := strings.LastIndex(line, ":"); idx != -1 {
		path := line[:idx]
		numStr := line[idx+1:]
		if dashIdx := strings.Index(numStr, "-"); dashIdx != -1 {
			numStr = numStr[:dashIdx]
		}
		lineNum, _ := strconv.Atoi(numStr)
		return path, lineNum
	}
	return line, 0
}
