package comment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"

	"github.com/junlang/PRReviewer/internal/analyzer"
)

// IssueCommentClient abstracts the GitHub Issues API for posting comments.
type IssueCommentClient interface {
	CreateComment(ctx context.Context, owner, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

type Formatter struct {
	owner    string
	repo     string
	prNumber int
}

func NewFormatter(owner, repo string, prNumber int) *Formatter {
	return &Formatter{owner: owner, repo: repo, prNumber: prNumber}
}

func (f *Formatter) Format(result *analyzer.AnalysisResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("## AI Review\n\n")

	// Summary
	if result.Summary != nil && result.Summary.Error == nil {
		sb.WriteString("**变更概述**: ")
		sb.WriteString(result.Summary.Summary)
		sb.WriteString("\n\n")
	}

	// Risk Scan
	sb.WriteString("---\n\n")
	sb.WriteString("### Risk Scan\n\n")

	if result.Risks != nil && len(result.Risks.Risks) > 0 {
		risksBySeverity := groupBySeverity(result.Risks.Risks)
		for _, severity := range []string{"critical", "warning", "suggestion"} {
			risks, ok := risksBySeverity[severity]
			if !ok || len(risks) == 0 {
				continue
			}
			sb.WriteString(f.severityHeader(severity))
			for _, r := range risks {
				sb.WriteString(f.formatRisk(r))
			}
		}
	} else if result.Risks != nil && result.Risks.Error != nil {
		sb.WriteString("> Risk scan failed: ")
		sb.WriteString(result.Risks.Error.Error())
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("> 未发现明显风险\n\n")
	}

	// Suggestions (Stage 3)
	if result.Suggestions != nil && len(result.Suggestions.Suggestions) > 0 {
		sb.WriteString("---\n\n")
		sb.WriteString("### 优化建议\n\n")
		for _, s := range result.Suggestions.Suggestions {
			sb.WriteString(f.formatSuggestion(s))
		}
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("> AI Reviewer v1 · %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String()
}

func (f *Formatter) severityHeader(severity string) string {
	switch severity {
	case "critical":
		return "#### Critical\n\n"
	case "warning":
		return "#### Warning\n\n"
	case "suggestion":
		return "#### Suggestion\n\n"
	}
	return ""
}

func (f *Formatter) formatRisk(r analyzer.Risk) string {
	link := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s#L%d",
		f.owner, f.repo, "HEAD", r.File, r.Line)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **%s** [`%s:%d`](%s)\n", r.Title, r.File, r.Line, link))
	sb.WriteString(fmt.Sprintf("  > %s\n", r.Description))
	if r.FixSuggestion != "" {
		sb.WriteString(fmt.Sprintf("  >\n  > **建议修复**: %s\n", r.FixSuggestion))
	}
	sb.WriteString(fmt.Sprintf("  > 置信度: %s | 严重度: %s\n\n", r.Confidence, r.Severity))
	return sb.String()
}

func (f *Formatter) formatSuggestion(s analyzer.Suggestion) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **%s:%d**\n", s.File, s.Line))
	sb.WriteString(fmt.Sprintf("  > %s\n", s.Description))
	if s.CodeSnippet != "" {
		sb.WriteString("  >\n  > ```\n")
		sb.WriteString(fmt.Sprintf("  > %s\n", s.CodeSnippet))
		sb.WriteString("  > ```\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func groupBySeverity(risks []analyzer.Risk) map[string][]analyzer.Risk {
	result := make(map[string][]analyzer.Risk)
	for _, r := range risks {
		result[r.Severity] = append(result[r.Severity], r)
	}
	return result
}

type Publisher struct {
	client IssueCommentClient
}

func NewPublisher(client IssueCommentClient) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) Publish(ctx context.Context, owner, repo string, prNumber int, result *analyzer.AnalysisResult) error {
	formatter := NewFormatter(owner, repo, prNumber)
	body := formatter.Format(result)

	comment := &github.IssueComment{Body: &body}
	_, _, err := p.client.CreateComment(ctx, owner, repo, prNumber, comment)
	return err
}
