package analyzer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type PipelineInput struct {
	Diff           string
	FileContents   map[string]string
	Stage3Eligible bool
}

type Pipeline struct {
	llm         LLMClient
	modelFast   string
	modelPower  string
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

	if input.Stage3Eligible {
		suggestions, err := p.runSuggestions(ctx, input, risks)
		if err != nil {
			result.Suggestions = &SuggestionResult{Error: err}
		} else {
			result.Suggestions = &SuggestionResult{Suggestions: suggestions}
		}
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

func (p *Pipeline) runSuggestions(ctx context.Context, input PipelineInput, risks []Risk) ([]Suggestion, error) {
	prompt := input.buildSuggestionPrompt(risks)
	resp, err := p.llm.Chat(ctx, p.modelPower, systemPromptSuggestion, prompt)
	if err != nil {
		return nil, fmt.Errorf("stage 3 suggestions: %w", err)
	}
	suggestions := parseSuggestionResponse(resp)
	if len(suggestions) == 0 && resp != "" && strings.TrimSpace(resp) != "无" {
		truncated := resp
		if len(truncated) > 300 {
			truncated = truncated[:300] + "..."
		}
		fmt.Printf("[stage3] LLM raw response (no suggestions parsed): %s",
			strings.ReplaceAll(truncated, "\n", "\\n"))
	}
	return suggestions, nil
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

func (input PipelineInput) buildSuggestionPrompt(risks []Risk) string {
	var sb strings.Builder
	sb.WriteString("请对以下 PR 变更中的代码提供行级改进建议：\n\n")

	sb.WriteString("## 变更 Diff\n")
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

func parseRiskResponse(resp string) []Risk {
	if resp == "" {
		return nil
	}

	var risks []Risk
	sections := []struct {
		marker   string
		severity string
	}{
		{"### Critical", "critical"},
		{"### Warning", "warning"},
		{"### Suggestion", "suggestion"},
	}

	for _, sec := range sections {
		content := extractSection(resp, sec.marker)
		if isEmpty(content) {
			continue
		}
		risks = append(risks, parseRiskItems(content, sec.severity)...)
	}

	return risks
}

func extractSection(resp, marker string) string {
	idx := strings.Index(resp, marker)
	if idx < 0 {
		return ""
	}
	content := resp[idx+len(marker):]

	// Stop at next section marker
	for _, m := range []string{"\n### Critical", "\n### Warning", "\n### Suggestion", "\n---"} {
		if i := strings.Index(content, m); i >= 0 {
			content = content[:i]
			break
		}
	}
	return strings.TrimSpace(content)
}

func isEmpty(content string) bool {
	c := strings.TrimSpace(content)
	return c == "" || c == "（无）" || c == "(无)" || c == "无"
}

func parseRiskItems(content, severity string) []Risk {
	var risks []Risk

	// Ensure content starts with a bullet delimiter for consistent splitting
	if strings.HasPrefix(content, "- ") || strings.HasPrefix(content, "* ") {
		content = "\n" + content
	}
	bullets := strings.Split(content, "\n- ")
	if len(bullets) > 1 {
		for _, bullet := range bullets {
			bullet = strings.TrimSpace(bullet)
			if bullet == "" || bullet == "（无）" || bullet == "(无)" {
				continue
			}
			r := parseBulletRisk(bullet, severity)
			if r != nil {
				risks = append(risks, *r)
			}
		}
		return risks
	}

	// Fallback: treat entire section as one risk
	risks = append(risks, Risk{
		Title:       extractTitle(content),
		Severity:    severity,
		Confidence:  extractConfidence(content),
		Description: content,
		File:        extractFile(content),
		Line:        extractLine(content),
	})
	return risks
}

func parseBulletRisk(bullet, severity string) *Risk {
	// Find first newline to split headline from description
	headline := bullet
	description := ""
	idx := strings.Index(bullet, "\n")
	if idx >= 0 {
		headline = bullet[:idx]
		description = strings.TrimSpace(bullet[idx+1:])
	}
	headline = strings.TrimSpace(headline)
	headline = strings.TrimPrefix(headline, "- ")
	headline = strings.TrimPrefix(headline, "* ")
	// Remove any leading newline
	if strings.HasPrefix(headline, "\n") {
		headline = headline[1:]
	}

	fixSuggestion := ""
	if description != "" {
		if i := strings.Index(description, "建议修复"); i >= 0 {
			fixSuggestion = strings.TrimSpace(description[i:])
			fixSuggestion = strings.TrimPrefix(fixSuggestion, "建议修复")
			fixSuggestion = strings.TrimPrefix(fixSuggestion, "：")
			fixSuggestion = strings.TrimPrefix(fixSuggestion, ":")
			fixSuggestion = strings.TrimSpace(fixSuggestion)
			description = strings.TrimSpace(description[:i])
		}
	}

	// Extract title from **...** in headline
	title := extractTitle(headline)
	file := extractFile(headline)
	line := extractLine(headline)
	confidence := extractConfidence(headline)

	return &Risk{
		Title:         title,
		File:          file,
		Line:          line,
		Severity:      severity,
		Confidence:    confidence,
		Description:   description,
		FixSuggestion: fixSuggestion,
	}
}

func extractTitle(text string) string {
	start := strings.Index(text, "**")
	if start < 0 {
		// Use first 60 chars as title
		if len(text) > 60 {
			return text[:60] + "..."
		}
		return text
	}
	end := strings.Index(text[start+2:], "**")
	if end < 0 {
		return text[start+2:]
	}
	return text[start+2 : start+2+end]
}

func extractConfidence(text string) string {
	for _, level := range []string{"high", "medium", "low"} {
		if strings.Contains(strings.ToLower(text), "置信度: "+level) ||
			strings.Contains(strings.ToLower(text), "置信度:"+level) ||
			strings.Contains(strings.ToLower(text), "置信度："+level) {
			return level
		}
	}
	return "medium"
}

func extractFile(text string) string {
	// Look for `file:line` pattern
	start := strings.Index(text, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(text[start+1:], "`")
	if end < 0 {
		return ""
	}
	inner := text[start+1 : start+1+end]

	// Split by : to get file and line
	if colon := strings.LastIndex(inner, ":"); colon >= 0 {
		return inner[:colon]
	}
	return inner
}

func extractLine(text string) int {
	start := strings.Index(text, "`")
	if start < 0 {
		return 0
	}
	end := strings.Index(text[start+1:], "`")
	if end < 0 {
		return 0
	}
	inner := text[start+1 : start+1+end]

	if colon := strings.LastIndex(inner, ":"); colon >= 0 {
		lineStr := inner[colon+1:]
		if n, err := strconv.Atoi(lineStr); err == nil {
			return n
		}
	}
	return 0
}

func parseSuggestionResponse(resp string) []Suggestion {
	if resp == "" || strings.TrimSpace(resp) == "无" {
		return nil
	}

	var suggestions []Suggestion

	// Split by "### " to get file sections
	sections := strings.Split(resp, "\n### ")
	for _, section := range sections {
		section = strings.TrimSpace(section)
		section = strings.TrimPrefix(section, "### ")
		if section == "" {
			continue
		}

		// First line is the file path
		lines := strings.SplitN(section, "\n", 2)
		filePath := strings.TrimSpace(lines[0])
		if filePath == "" || filePath == "无" {
			continue
		}

		body := ""
		if len(lines) > 1 {
			body = lines[1]
		}

		// Parse individual suggestions within this file section
		items := strings.Split(body, "\n- **")
		for _, item := range items {
			item = strings.TrimSpace(item)
			item = strings.TrimPrefix(item, "- **")
			if item == "" || item == "无" {
				continue
			}

			s := parseSuggestionItem(item, filePath)
			if s != nil {
				suggestions = append(suggestions, *s)
			}
		}
	}

	return suggestions
}

func parseSuggestionItem(item, filePath string) *Suggestion {
	// Format: <line>** <description>\n  ```<lang>\n  <code>```
	// Find the line number (first characters until **)
	firstLineEnd := strings.Index(item, "\n")
	headline := item
	rest := ""
	if firstLineEnd >= 0 {
		headline = item[:firstLineEnd]
		rest = strings.TrimSpace(item[firstLineEnd+1:])
	}
	headline = strings.TrimSpace(headline)

	// Extract line number: digits before "**"
	lineNum := 0
	starIdx := strings.Index(headline, "**")
	if starIdx >= 0 {
		lineStr := strings.TrimSpace(headline[:starIdx])
		if n, err := strconv.Atoi(lineStr); err == nil {
			lineNum = n
		}
	} else {
		// Try to parse just a number at start
		parts := strings.Fields(headline)
		if len(parts) > 0 {
			if n, err := strconv.Atoi(parts[0]); err == nil {
				lineNum = n
			}
		}
	}

	// Description: after "** " until code block or end
	descStart := starIdx + 2
	if descStart >= 0 && descStart < len(headline) {
		// Skip "**"
	}

	description := ""
	codeSnippet := ""

	if strings.Contains(rest, "```") {
		// Has code block
		descEnd := strings.Index(rest, "```")
		if descEnd >= 0 {
			description = strings.TrimSpace(rest[:descEnd])
			// Extract code between first and second ```
			codeStart := strings.Index(rest[descEnd:], "\n")
			if codeStart >= 0 {
				codeBody := rest[descEnd+codeStart+1:]
				codeEnd := strings.Index(codeBody, "```")
				if codeEnd >= 0 {
					codeSnippet = strings.TrimSpace(codeBody[:codeEnd])
				}
			}
		}
	} else {
		description = rest
	}

	// If no structured description found, use headline after "** "
	if description == "" && starIdx >= 0 {
		afterStar := headline[starIdx+2:]
		afterStar = strings.TrimSpace(afterStar)
		if afterStar != "" {
			description = afterStar
		}
	}

	return &Suggestion{
		File:        filePath,
		Line:        lineNum,
		Description: description,
		CodeSnippet: codeSnippet,
	}
}

const systemPromptSummary = "你是一个代码评审助手。你的任务是用中文简洁地总结 PR 变更内容。\n" +
	"要求：\n" +
	"1. 2-4 句话概括变更的核心目的\n" +
	"2. 列出受影响的关键模块或文件\n" +
	"3. 用中文回复"

const systemPromptRisk = "你是一个代码安全与质量评审专家。请识别以下 PR 中的潜在风险。\n" +
	"重点关注：\n" +
	"- 安全漏洞（SQL 注入、XSS、密钥泄漏、权限绕过）\n" +
	"- 逻辑错误（空指针、边界条件、错误处理缺失）\n" +
	"- 并发问题（竞态条件、死锁）\n" +
	"- 破坏性变更（接口不兼容、API 签名变更）\n" +
	"\n" +
	"请按以下格式回复：\n" +
	"### Critical\n" +
	"- **标题** `文件:行号` 置信度: high/medium/low\n" +
	"  描述和建议修复\n" +
	"\n" +
	"### Warning\n" +
	"...\n" +
	"\n" +
	"### Suggestion\n" +
	"..."

	const systemPromptSuggestion = "你是一个代码改进顾问。请对以下代码提供具体的优化建议。\n" +
		"重点关注：代码可读性、性能、测试覆盖、最佳实践。\n" +
		"\n" +
		"请按以下格式回复，每个文件一个区块：\n" +
		"\n" +
		"### <文件路径>\n" +
		"- **<行号>** <简短描述>\n" +
		"  ```<语言>\n" +
		"  <改进后的代码示例>\n" +
		"  ```\n" +
		"\n" +
		"如果没有建议，回复'无'。"