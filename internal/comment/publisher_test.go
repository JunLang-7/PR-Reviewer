package comment

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"

	"github.com/junlang/PRReviewer/internal/analyzer"
	prcontext "github.com/junlang/PRReviewer/internal/context"
)

type mockReviewClient struct {
	reviews []*github.PullRequestReviewRequest
}

func (m *mockReviewClient) CreateReview(ctx context.Context, owner, repo string, number int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, *github.Response, error) {
	m.reviews = append(m.reviews, review)
	return &github.PullRequestReview{Body: review.Body}, nil, nil
}

func buildTestResult(summary string, risks []analyzer.Risk) *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		Summary: &analyzer.SummaryResult{Summary: summary},
		Risks:   &analyzer.RiskResult{Risks: risks},
	}
}

func TestFormatComment_ContainsSections(t *testing.T) {
	result := buildTestResult("重构用户模块", []analyzer.Risk{
		{Title: "SQL 注入", File: "db.go", Line: 42, Severity: "critical", Confidence: "high", Description: "直接拼接 SQL", FixSuggestion: "使用参数化查询"},
		{Title: "错误忽略", File: "api.go", Line: 15, Severity: "warning", Confidence: "medium", Description: "err 未检查"},
	})

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
	result := buildTestResult("修复 typo", nil)
	formatter := NewFormatter("o", "r", 1)
	body := formatter.Format(result)
	if !strings.Contains(body, "修复 typo") {
		t.Error("should contain summary")
	}
}

func TestPublisher_PostComment(t *testing.T) {
	mock := &mockReviewClient{}
	pub := NewPublisher(mock)
	ctx := context.Background()

	result := buildTestResult("test summary", nil)
	err := pub.Publish(ctx, "owner", "repo", 1, result, nil)

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

func TestBuildInlineComments_WithValidPosition(t *testing.T) {
	risks := []analyzer.Risk{
		{Title: "SQL 注入", File: "db.go", Line: 3, Severity: "critical", Description: "直接拼接 SQL", FixSuggestion: "使用参数化查询"},
	}
	diffFiles := []prcontext.DiffFile{
		{Path: "db.go", Patch: "@@ -1,3 +1,4 @@\n line1\n line2\n+line3\n line4"},
	}

	comments, fallback := buildInlineComments(risks, diffFiles)

	if len(comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(comments))
	}
	if fallback != "" {
		t.Error("fallback body should be empty when all risks have valid positions")
	}
	if comments[0].GetPath() != "db.go" {
		t.Errorf("expected path db.go, got %s", comments[0].GetPath())
	}
	if comments[0].GetPosition() != 3 {
		t.Errorf("expected position 3, got %d", comments[0].GetPosition())
	}
	if !strings.Contains(comments[0].GetBody(), "SQL 注入") {
		t.Error("comment body should contain risk title")
	}
	if !strings.Contains(comments[0].GetBody(), "```suggestion") {
		t.Error("comment body should contain suggestion block")
	}
}

func TestBuildInlineComments_NoSuggestedChangeWhenFixSuggestionEmpty(t *testing.T) {
	risks := []analyzer.Risk{
		{Title: "错误忽略", File: "api.go", Line: 3, Severity: "warning", Description: "err 未检查", FixSuggestion: ""},
	}
	diffFiles := []prcontext.DiffFile{
		{Path: "api.go", Patch: "@@ -1,3 +1,4 @@\n line1\n line2\n+line3\n line4"},
	}

	comments, _ := buildInlineComments(risks, diffFiles)

	if len(comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(comments))
	}
	if strings.Contains(comments[0].GetBody(), "```suggestion") {
		t.Error("comment body should NOT contain suggestion block when FixSuggestion is empty")
	}
}

func TestBuildInlineComments_FallbackWhenNoPosition(t *testing.T) {
	risks := []analyzer.Risk{
		{Title: "行号不存在", File: "unknown.go", Line: 999, Severity: "warning", Description: "某问题"},
		{Title: "有效风险", File: "main.go", Line: 3, Severity: "critical", Description: "严重问题"},
	}
	diffFiles := []prcontext.DiffFile{
		{Path: "main.go", Patch: "@@ -1,3 +1,4 @@\n line1\n line2\n+line3\n line4"},
	}

	comments, fallback := buildInlineComments(risks, diffFiles)

	if len(comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(comments))
	}
	if !strings.Contains(fallback, "行号不存在") {
		t.Error("fallback body should contain risk with no diff position")
	}
	if strings.Contains(fallback, "有效风险") {
		t.Error("fallback body should NOT contain risk that has valid inline comment")
	}
}

func TestBuildInlineComments_EmptyRisks(t *testing.T) {
	comments, fallback := buildInlineComments(nil, nil)

	if comments != nil {
		t.Error("expected nil comments for empty risks")
	}
	if fallback != "" {
		t.Error("expected empty fallback for empty risks")
	}
}

func TestPublisher_PostComment_WithDiffFiles(t *testing.T) {
	mock := &mockReviewClient{}
	pub := NewPublisher(mock)
	ctx := context.Background()

	result := buildTestResult("test summary", []analyzer.Risk{
		{Title: "风险1", File: "main.go", Line: 3, Severity: "warning", Description: "问题描述", FixSuggestion: "修复代码"},
	})
	diffFiles := []prcontext.DiffFile{
		{Path: "main.go", Patch: "@@ -1,3 +1,4 @@\n line1\n line2\n+line3\n line4"},
	}

	err := pub.Publish(ctx, "owner", "repo", 1, result, diffFiles)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(mock.reviews))
	}
	if len(mock.reviews[0].Comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(mock.reviews[0].Comments))
	}
	body := mock.reviews[0].GetBody()
	if !strings.Contains(body, "test summary") {
		t.Error("review body should contain summary")
	}
	if !strings.Contains(body, "AI Reviewer") {
		t.Error("review should contain footer")
	}
}

func TestFormatComment_Timestamp(t *testing.T) {
	result := buildTestResult("test", nil)
	formatter := NewFormatter("o", "r", 1)
	body := formatter.Format(result)

	today := time.Now().In(getLocation()).Format("2006-01-02")
	if !strings.Contains(body, today) {
		t.Errorf("comment should contain today's date %s", today)
	}
}
