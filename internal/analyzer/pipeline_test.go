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
			"risk":    "In `main.go`:\n\n>  func main() {}\n缺少错误处理\n严重程度: critical\n----",
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

func TestParseRiskResponse_Flat(t *testing.T) {
	resp := "In `server/tag.go`:\n\n" +
		">  func NewTagServer() *TagServer {\n" +
		"     return &TagServer{}\n" +
		"   }\n" +
		"NewTagServer returns nil auth, dereferences and panics.\n" +
		"严重程度: critical\n" +
		"----\n\n" +
		"In `pkg/bapi/api.go`:\n\n" +
		"> - app_key = value[0]\n" +
		">+ if value, ok := md[\"app_key\"]; ok {\n" +
		">+     appKey = value[0]\n" +
		">+ }\n" +
		"Reading metadata without checking slice length.\n" +
		"严重程度: critical\n" +
		"----\n\n" +
		"In `cmd/api/main.go`:\n\n" +
		">  p := strings.TrimPrefix(r.URL.Path, \"/swagger/\")\n" +
		"User input not sanitized, path traversal risk.\n" +
		"严重程度: warning\n" +
		"----"

	risks := parseRiskResponse(resp)
	if len(risks) != 3 {
		t.Fatalf("expected 3 risks, got %d", len(risks))
	}

	if risks[0].Severity != "critical" {
		t.Errorf("expected critical, got %s", risks[0].Severity)
	}
	if risks[0].File != "server/tag.go" {
		t.Errorf("expected server/tag.go, got %s", risks[0].File)
	}
	if !strings.Contains(risks[0].Description, "auth") {
		t.Errorf("expected description about auth, got %s", risks[0].Description)
	}

	if risks[1].File != "pkg/bapi/api.go" {
		t.Errorf("expected pkg/bapi/api.go, got %s", risks[1].File)
	}
	if !strings.Contains(risks[1].FixSuggestion, "app_key") {
		t.Errorf("expected code block with app_key, got %s", risks[1].FixSuggestion)
	}

	if risks[2].Severity != "warning" {
		t.Errorf("expected warning, got %s", risks[2].Severity)
	}
	if risks[2].File != "cmd/api/main.go" {
		t.Errorf("expected cmd/api/main.go, got %s", risks[2].File)
	}
}

func TestParseRiskResponse_Empty(t *testing.T) {
	if risks := parseRiskResponse(""); risks != nil {
		t.Error("expected nil for empty response")
	}
}

func TestExtractInFile(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"In `server/tag.go`:", "server/tag.go"},
		{"In `pkg/bapi/api.go`:", "pkg/bapi/api.go"},
		{"no backticks", ""},
	}
	for _, tt := range tests {
		got := extractInFile(tt.line)
		if got != tt.want {
			t.Errorf("extractInFile(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}
