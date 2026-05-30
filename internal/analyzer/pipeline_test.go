package analyzer

import (
	"context"
	"strings"
	"testing"
)

type mockLLM struct {
	responses map[string]string
}

func (m *mockLLM) Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	for key, resp := range m.responses {
		if strings.Contains(userMessage, key) {
			return resp, nil
		}
	}
	return "mock response", nil
}

func TestPipeline_Run_AllStages(t *testing.T) {
	mock := &mockLLM{
		responses: map[string]string{
			"summary": "这是一个用户注册模块的重构",
			"risk":    "---",
		},
	}

	pipeline := NewPipeline(mock, "fast-model", "power-model")
	result, err := pipeline.Run(context.Background(), PipelineInput{
		Diff:         "mock diff content",
		FileContents: map[string]string{"main.go": "package main"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary == nil {
		t.Error("expected summary result")
	}
	if result.Risks == nil {
		t.Error("expected risk result")
	}
}

func TestPipeline_Run_PartialFailure_Stage2(t *testing.T) {
	mock := &mockLLM{
		responses: map[string]string{
			"summary": "summary only",
		},
	}

	pipeline := NewPipeline(mock, "fast-model", "power-model")
	result, err := pipeline.Run(context.Background(), PipelineInput{
		Diff:         "mock diff",
		FileContents: map[string]string{"a.go": "code"},
	})

	if err != nil {
		t.Fatalf("pipeline should not fail entirely on stage error: %v", err)
	}
	if result.Summary == nil {
		t.Error("summary should still be present")
	}
}

func TestPipelineInput_BuildPrompt_ContainsContext(t *testing.T) {
	input := PipelineInput{
		Diff: "@@ -1,3 +1,5 @@\n+new line\n old line",
		FileContents: map[string]string{
			"main.go": "package main\n\nfunc main() {}\n",
		},
	}

	prompt := input.buildSummaryPrompt()
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt should contain file name")
	}
	if !strings.Contains(prompt, "new line") {
		t.Error("prompt should contain diff content")
	}
}

func TestPipelineInput_BuildRiskPrompt_ContainsCode(t *testing.T) {
	input := PipelineInput{
		Diff: "@@ -1 +1 @@\n-old\n+new",
		FileContents: map[string]string{
			"auth.go": "func login() { password := getUserInput() }",
		},
	}

	prompt := input.buildRiskPrompt()
	if !strings.Contains(prompt, "auth.go") {
		t.Error("risk prompt should contain file name")
	}
	if !strings.Contains(prompt, "password") {
		t.Error("risk prompt should contain file content")
	}
}

func TestParseRiskResponse_Structured(t *testing.T) {
	resp := "### Critical\n- **SQL 注入风险** `db/user.go:42` 置信度: high\n  直接拼接 SQL 查询字符串，存在注入风险\n  建议修复：使用参数化查询\n\n### Warning\n- **错误处理缺失** `api/handler.go:15` 置信度: medium\n  err 返回值未检查\n\n### Suggestion\n（无）"

	risks := parseRiskResponse(resp)
	if len(risks) != 2 {
		t.Fatalf("expected 2 risks, got %d", len(risks))
	}
	if risks[0].Title != "SQL 注入风险" {
		t.Errorf("expected title 'SQL 注入风险', got %q", risks[0].Title)
	}
	if risks[0].Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", risks[0].Severity)
	}
	if risks[0].Confidence != "high" {
		t.Errorf("expected confidence 'high', got %q", risks[0].Confidence)
	}
	if risks[0].File != "db/user.go" {
		t.Errorf("expected file 'db/user.go', got %q", risks[0].File)
	}
	if risks[0].Line != 42 {
		t.Errorf("expected line 42, got %d", risks[0].Line)
	}
	if !strings.Contains(risks[0].FixSuggestion, "参数化查询") {
		t.Errorf("fix suggestion should contain fix, got %q", risks[0].FixSuggestion)
	}
	if risks[1].Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", risks[1].Severity)
	}
}

func TestParseRiskResponse_Empty(t *testing.T) {
	if risks := parseRiskResponse(""); risks != nil {
		t.Error("expected nil for empty response")
	}
}

func TestParseRiskResponse_AllEmpty(t *testing.T) {
	resp := "### Critical\n（无）\n\n### Warning\n无\n\n### Suggestion\n(无)"
	risks := parseRiskResponse(resp)
	if len(risks) != 0 {
		t.Errorf("expected 0 risks, got %d", len(risks))
	}
}

func TestParseRiskResponse_FallbackFormat(t *testing.T) {
	resp := "### Warning\n文档变更无代码风险，本次仅移动了 Markdown 文件。"
	risks := parseRiskResponse(resp)
	if len(risks) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(risks))
	}
	if risks[0].Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", risks[0].Severity)
	}
}

func TestExtractConfidence(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"置信度: high", "high"},
		{"置信度: medium", "medium"},
		{"置信度: low", "low"},
		{"置信度：low", "low"},
		{"no confidence info", "medium"},
	}
	for _, tt := range tests {
		got := extractConfidence(tt.text)
		if got != tt.want {
			t.Errorf("extractConfidence(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestExtractFileLine(t *testing.T) {
	file := extractFile("**title** `handlers/auth.go:153` 置信度: high")
	if file != "handlers/auth.go" {
		t.Errorf("expected 'handlers/auth.go', got %q", file)
	}
	line := extractLine("**title** `handlers/auth.go:153` 置信度: high")
	if line != 153 {
		t.Errorf("expected 153, got %d", line)
	}
}
