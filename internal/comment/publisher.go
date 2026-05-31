package comment

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"

	"github.com/junlang/PRReviewer/internal/analyzer"
	prcontext "github.com/junlang/PRReviewer/internal/context"
)

func getLocation() *time.Location {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.FixedZone(tz, 8*60*60)
	}
	return loc
}

// PRReviewClient abstracts the GitHub Pull Requests API for creating reviews.
type PRReviewClient interface {
	CreateReview(ctx context.Context, owner, repo string, number int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, *github.Response, error)
}

type Formatter struct {
	owner    string
	repo     string
	prNumber int
}

func NewFormatter(owner, repo string, prNumber int) *Formatter {
	return &Formatter{owner: owner, repo: repo, prNumber: prNumber}
}

func (f *Formatter) FormatSummary(result *analyzer.AnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("## AI Review\n\n")

	if result.Summary != nil && result.Summary.Error == nil {
		sb.WriteString("**变更概述**: ")
		sb.WriteString(result.Summary.Summary)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func (f *Formatter) FormatFooter() string {
	return fmt.Sprintf("---\n\n> AI Reviewer v1 · %s\n", time.Now().In(getLocation()).Format("2006-01-02 15:04:05"))
}

func (f *Formatter) Format(result *analyzer.AnalysisResult) string {
	var sb strings.Builder

	sb.WriteString(f.FormatSummary(result))

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

	sb.WriteString(f.FormatFooter())

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
		sb.WriteString(fmt.Sprintf("  >\n  > %s\n", r.FixSuggestion))
	}
	return sb.String()
}

func groupBySeverity(risks []analyzer.Risk) map[string][]analyzer.Risk {
	result := make(map[string][]analyzer.Risk)
	for _, r := range risks {
		result[r.Severity] = append(result[r.Severity], r)
	}
	return result
}

// buildInlineComments converts risks into GitHub inline review comments using
// diff positions. Risks without valid positions are returned as fallback text.
func buildInlineComments(risks []analyzer.Risk, diffFiles []prcontext.DiffFile) (comments []*github.DraftReviewComment, fallbackBody string) {
	var fallbackLines []string

	for _, r := range risks {
		pos := findDiffPosition(diffFiles, r.File, r.Line)
		if pos > 0 {
			body := buildCommentBody(r)
			comments = append(comments, &github.DraftReviewComment{
				Path:     github.Ptr(r.File),
				Position: github.Ptr(pos),
				Body:     github.Ptr(body),
			})
		} else {
			line := fmt.Sprintf("- **%s** [`%s:%d`]\n  > %s", r.Title, r.File, r.Line, r.Description)
			if r.FixSuggestion != "" {
				line += fmt.Sprintf("\n  >\n  > %s", r.FixSuggestion)
			}
			fallbackLines = append(fallbackLines, line)
		}
	}

	if len(fallbackLines) > 0 {
		fallbackBody = "---\n\n### Risk Scan (Fallback)\n\n" + strings.Join(fallbackLines, "\n") + "\n\n"
	}

	return comments, fallbackBody
}

func findDiffPosition(diffFiles []prcontext.DiffFile, file string, line int) int {
	for _, df := range diffFiles {
		if df.Path == file {
			return analyzer.FileLineToPosition(df.Patch, line)
		}
	}
	return 0
}

func buildCommentBody(r analyzer.Risk) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s**\n\n%s", r.Title, r.Description))

	if r.FixSuggestion != "" {
		sb.WriteString(fmt.Sprintf("\n\n**修复建议**：\n%s", r.FixSuggestion))
	}

	return sb.String()
}
type Publisher struct {
	client PRReviewClient
}

func NewPublisher(client PRReviewClient) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) Publish(ctx context.Context, owner, repo string, prNumber int, result *analyzer.AnalysisResult, diffFiles []prcontext.DiffFile) error {
	formatter := NewFormatter(owner, repo, prNumber)

	var risks []analyzer.Risk
	if result.Risks != nil {
		risks = result.Risks.Risks
	}
	comments, fallbackBody := buildInlineComments(risks, diffFiles)

	body := formatter.FormatSummary(result)
	body += fallbackBody
	body += formatter.FormatFooter()

	event := "COMMENT"
	review := &github.PullRequestReviewRequest{
		Body:     &body,
		Event:    &event,
		Comments: comments,
	}
	_, _, err := p.client.CreateReview(ctx, owner, repo, prNumber, review)
	return err
}
