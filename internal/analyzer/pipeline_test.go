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
			"summary":   "这是一个用户注册模块的重构",
			"risk":      "---",
		},
	}

	pipeline := NewPipeline(mock, "fast-model", "power-model")
	result, err := pipeline.Run(context.Background(), PipelineInput{
		Diff:            "mock diff content",
		FileContents:    map[string]string{"main.go": "package main"},
		Stage3Eligible:  true,
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
	if result.Suggestions == nil {
		t.Error("expected suggestion result when eligible")
	}
}

func TestPipeline_Run_SkipsStage3(t *testing.T) {
	mock := &mockLLM{
		responses: map[string]string{
			"summary": "summary text",
			"risk":    "no risks",
		},
	}

	pipeline := NewPipeline(mock, "fast-model", "power-model")
	result, err := pipeline.Run(context.Background(), PipelineInput{
		Diff:           "mock diff",
		FileContents:   map[string]string{"a.go": "code"},
		Stage3Eligible: false,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestions != nil {
		t.Error("should skip stage 3 when not eligible")
	}
}

func TestPipeline_Run_PartialFailure_Stage2(t *testing.T) {
	mock := &mockLLM{
		responses: map[string]string{
			"summary": "summary only",
			// risk intentionally left out → error
		},
	}

	pipeline := NewPipeline(mock, "fast-model", "power-model")
	result, err := pipeline.Run(context.Background(), PipelineInput{
		Diff:           "mock diff",
		FileContents:   map[string]string{"a.go": "code"},
		Stage3Eligible: false,
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
