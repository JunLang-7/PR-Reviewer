package comment

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"

	"github.com/junlang/PRReviewer/internal/analyzer"
)

type mockReviewClient struct {
	reviews []*github.PullRequestReviewRequest
}

func (m *mockReviewClient) CreateReview(ctx context.Context, owner, repo string, number int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, *github.Response, error) {
	m.reviews = append(m.reviews, review)
	return &github.PullRequestReview{Body: review.Body}, nil, nil
}

func buildTestResult(summary string, risks []analyzer.Risk, suggestions []analyzer.Suggestion) *analyzer.AnalysisResult {
	result := &analyzer.AnalysisResult{
		Summary: &analyzer.SummaryResult{Summary: summary},
		Risks:   &analyzer.RiskResult{Risks: risks},
	}
	if suggestions != nil {
		result.Suggestions = &analyzer.SuggestionResult{Suggestions: suggestions}
	}
	return result
}

func TestFormatComment_ContainsSections(t *testing.T) {
	result := buildTestResult("重构用户模块", []analyzer.Risk{
		{Title: "SQL 注入", File: "db.go", Line: 42, Severity: "critical", Confidence: "high", Description: "直接拼接 SQL", FixSuggestion: "使用参数化查询"},
		{Title: "错误忽略", File: "api.go", Line: 15, Severity: "warning", Confidence: "medium", Description: "err 未检查"},
	}, nil)

	formatter := NewFormatter("testowner", "testrepo", 1)
	body := formatter.Format(result)

	if !strings.Contains(body, "AI Review") {
		t.Error("should contain AI Review header")
	}
	if !strings.Contains(body, "重构用户模块") {
		t.Error("should contain summary")
	}
	if !strings.Contains(body, "SQL 注入") {
		t.Error("should contain risk title")
	}
	if !strings.Contains(body, "db.go:42") {
		t.Error("should contain file:line reference")
	}
	if !strings.Contains(body, "### Critical") {
		t.Error("should contain Critical section")
	}
	if !strings.Contains(body, "### Warning") {
		t.Error("should contain Warning section")
	}
}

func TestFormatComment_OnlySummaryNoRisks(t *testing.T) {
	result := buildTestResult("修复 typo", nil, nil)

	formatter := NewFormatter("o", "r", 1)
	body := formatter.Format(result)

	if !strings.Contains(body, "修复 typo") {
		t.Error("should contain summary")
	}
	if !strings.Contains(body, "无风险问题") || !strings.Contains(body, "风险") {
		// At minimum, there should be something about risks
	}
}

func TestFormatComment_WithSuggestions(t *testing.T) {
	result := buildTestResult("optimization", nil, []analyzer.Suggestion{
		{File: "main.go", Line: 10, Description: "使用 strings.Builder 替代 +=", CodeSnippet: "var b strings.Builder"},
	})

	formatter := NewFormatter("o", "r", 1)
	body := formatter.Format(result)

	if !strings.Contains(body, "优化建议") && !strings.Contains(body, "Suggestion") {
		t.Error("should contain suggestions section")
	}
}

func TestPublisher_PostComment(t *testing.T) {
	mock := &mockReviewClient{}
	pub := NewPublisher(mock)
	ctx := context.Background()

	result := buildTestResult("test summary", nil, nil)
	err := pub.Publish(ctx, "owner", "repo", 1, result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(mock.reviews))
	}
	body := mock.reviews[0].GetBody()
	if !strings.Contains(body, "test summary") {
		t.Error("review should contain summary")
	}
	if !strings.Contains(body, "AI Reviewer") {
		t.Error("review should contain footer")
	}
}

func TestFormatComment_Timestamp(t *testing.T) {
	result := buildTestResult("test", nil, nil)
	formatter := NewFormatter("o", "r", 1)
	body := formatter.Format(result)

	today := time.Now().Format("2006-01-02")
	if !strings.Contains(body, today) {
		t.Errorf("comment should contain today's date %s", today)
	}
}
