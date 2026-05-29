package analyzer

import (
	"context"
	"fmt"
	"strings"
)

const (
	modelHaiku  = "claude-haiku-4-5"
	modelSonnet = "claude-sonnet-4-6"
)

type PipelineInput struct {
	Diff           string
	FileContents   map[string]string
	Stage3Eligible bool
}

type Pipeline struct {
	llm LLMClient
}

func NewPipeline(llm LLMClient) *Pipeline {
	return &Pipeline{llm: llm}
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
	resp, err := p.llm.Chat(ctx, modelHaiku, systemPromptSummary, prompt)
	if err != nil {
		return "", fmt.Errorf("stage 1 summary: %w", err)
	}
	return resp, nil
}

func (p *Pipeline) runRiskScan(ctx context.Context, input PipelineInput) ([]Risk, error) {
	prompt := input.buildRiskPrompt()
	resp, err := p.llm.Chat(ctx, modelSonnet, systemPromptRisk, prompt)
	if err != nil {
		return nil, fmt.Errorf("stage 2 risk scan: %w", err)
	}
	return parseRiskResponse(resp), nil
}

func (p *Pipeline) runSuggestions(ctx context.Context, input PipelineInput, risks []Risk) ([]Suggestion, error) {
	prompt := input.buildSuggestionPrompt(risks)
	resp, err := p.llm.Chat(ctx, modelSonnet, systemPromptSuggestion, prompt)
	if err != nil {
		return nil, fmt.Errorf("stage 3 suggestions: %w", err)
	}
	return parseSuggestionResponse(resp), nil
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
	sb.WriteString("请对以下 PR 变更提供具体的代码改进建议：\n\n")

	if len(risks) > 0 {
		sb.WriteString("## 已识别的风险\n")
		for _, r := range risks {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s:%d)\n", r.Severity, r.Title, r.File, r.Line))
		}
		sb.WriteString("\n")
	}

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
	if resp == "" || resp == "---" {
		return nil
	}
	return []Risk{
		{
			Title:       "AI 分析暂不可用",
			Severity:    "warning",
			Confidence:  "low",
			Description: resp,
		},
	}
}

func parseSuggestionResponse(resp string) []Suggestion {
	if resp == "" {
		return nil
	}
	return nil
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
	"重点关注：\n" +
	"- 代码可读性和命名\n" +
	"- 性能优化\n" +
	"- 测试覆盖\n" +
	"- 最佳实践\n" +
	"\n" +
	"请提供具体的行级建议和代码示例。"
