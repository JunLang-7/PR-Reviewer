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
			"risk":    "[main.go:1](ref):\n**问题**：缺少错误处理\n严重程度: critical",
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

	prompt := input.buildRiskPrompt("")
	if !strings.Contains(prompt, "auth.go") {
		t.Error("risk prompt should contain file name")
	}
	if !strings.Contains(prompt, "password") {
		t.Error("risk prompt should contain file content")
	}
}

func TestPipelineInput_BuildRiskPrompt_WithSummary(t *testing.T) {
	input := PipelineInput{
		Diff: "@@ -1 +1 @@\n-old\n+new",
		FileContents: map[string]string{
			"auth.go": "func login() {}",
		},
	}

	prompt := input.buildRiskPrompt("本次 PR 重构了认证模块")

	if !strings.Contains(prompt, "本次 PR 重构了认证模块") {
		t.Error("risk prompt should contain summary text")
	}
	if !strings.Contains(prompt, "PR 变更摘要") {
		t.Error("risk prompt should contain summary section header")
	}
	if !strings.Contains(prompt, "仅供参考") {
		t.Error("risk prompt should contain caveat about summary")
	}
	if !strings.Contains(prompt, "auth.go") {
		t.Error("risk prompt should still contain file name after summary")
	}
}

func TestPipelineInput_BuildRiskPrompt_EmptySummary(t *testing.T) {
	input := PipelineInput{
		Diff: "@@ -1 +1 @@\n-old\n+new",
		FileContents: map[string]string{
			"auth.go": "func login() {}",
		},
	}

	prompt := input.buildRiskPrompt("")

	if strings.Contains(prompt, "PR 变更摘要") {
		t.Error("risk prompt should not contain summary section when empty")
	}
	if !strings.Contains(prompt, "auth.go") {
		t.Error("risk prompt should still contain file content")
	}
}

func TestParseRiskResponse_Standard(t *testing.T) {
	resp := "[server/tag.go:23](ref):\n" +
		"**问题**：NewTagServer returns nil auth\n" +
		"**后果**：GetTagList dereferences and panics\n" +
		"**建议**：检查 auth 是否为 nil\n" +
		"严重程度: critical\n" +
		"\n" +
		"[cmd/api/main.go:56](ref):\n" +
		"**问题**：User input not sanitized\n" +
		"**后果**：path traversal risk\n" +
		"**建议**：使用 http.Dir 限制目录访问\n" +
		"严重程度: warning\n"

	risks := parseRiskResponse(resp)
	if len(risks) != 2 {
		t.Fatalf("expected 2 risks, got %d", len(risks))
	}

	if risks[0].File != "server/tag.go" {
		t.Errorf("expected server/tag.go, got %s", risks[0].File)
	}
	if risks[0].Severity != "critical" {
		t.Errorf("expected critical, got %s", risks[0].Severity)
	}
	if !strings.Contains(risks[0].Description, "auth") {
		t.Errorf("expected description about auth, got %s", risks[0].Description)
	}

	if risks[1].File != "cmd/api/main.go" {
		t.Errorf("expected cmd/api/main.go, got %s", risks[1].File)
	}
	if risks[1].Severity != "warning" {
		t.Errorf("expected warning, got %s", risks[1].Severity)
	}
}

func TestParseRiskResponse_Empty(t *testing.T) {
	if risks := parseRiskResponse(""); risks != nil {
		t.Error("expected nil for empty response")
	}
}

func TestParseRiskResponse_WithChineseColon(t *testing.T) {
	resp := "[pkg/bapi/api.go:16](ref)：\n" +
		"**问题**：硬编码密钥\n" +
		"**后果**：凭证泄露\n" +
		"**建议**：通过环境变量注入\n" +
		"严重程度：critical\n"

	risks := parseRiskResponse(resp)
	if len(risks) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(risks))
	}
	if risks[0].Severity != "critical" {
		t.Errorf("expected critical, got %s", risks[0].Severity)
	}
	if risks[0].File != "pkg/bapi/api.go" {
		t.Errorf("expected pkg/bapi/api.go, got %s", risks[0].File)
	}
}

func TestParseRiskResponse_WithNoiseHeaders(t *testing.T) {
	resp := "### 🔴 严重逻辑缺陷\n\n" +
		"[cmd/api/main.go:132](ref):\n" +
		"**问题**：gatewayMux is nil\n" +
		"**后果**：httpMux.Handle panics\n" +
		"**建议**：返回 error 由上层处理\n" +
		"严重程度: critical\n" +
		"\n" +
		"### ✅ 总结\n" +
		"建议在合并前修复所有 critical 问题\n" +
		"[go.mod:3](ref):\n" +
		"**问题**：Go 版本号无效\n" +
		"**后果**：编译失败\n" +
		"**建议**：修正为实际版本\n" +
		"严重程度: warning\n" +
		"\n" +
		"以上是本次评审的全部内容\n"

	risks := parseRiskResponse(resp)
	if len(risks) != 2 {
		t.Fatalf("expected 2 risks, got %d", len(risks))
	}
	if risks[0].File != "cmd/api/main.go" {
		t.Errorf("expected cmd/api/main.go, got %s", risks[0].File)
	}
	if risks[1].File != "go.mod" {
		t.Errorf("expected go.mod, got %s", risks[1].File)
	}
}

func TestParseRiskResponse_MissingField(t *testing.T) {
	resp := "[client/client.go:23](ref):\n" +
		"**问题**：Auth.RequireTransportSecurity 返回 false\n" +
		"**建议**：改用 TLS 传输\n" +
		"严重程度: warning\n"

	risks := parseRiskResponse(resp)
	if len(risks) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(risks))
	}
	if risks[0].Severity != "warning" {
		t.Errorf("expected warning, got %s", risks[0].Severity)
	}
	if !strings.Contains(risks[0].Description, "RequireTransportSecurity") {
		t.Errorf("expected description about auth, got %s", risks[0].Description)
	}
}

func TestParseRiskResponse_EmptyBody(t *testing.T) {
	resp := "[empty.go:1](ref):\n" +
		"严重程度: suggestion\n" +
		"\n" +
		"[valid.go:2](ref):\n" +
		"**问题**：有实际内容\n" +
		"**建议**：修复它\n" +
		"严重程度: warning\n"

	risks := parseRiskResponse(resp)
	if len(risks) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(risks))
	}
	if risks[0].File != "valid.go" {
		t.Errorf("expected valid.go, got %s", risks[0].File)
	}
}
