package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	gh "github.com/google/go-github/v69/github"
)

type PRInfo struct {
	Owner          string
	Repo           string
	PRNumber       int
	BaseRef        string
	HeadRef        string
	BaseSHA        string
	HeadSHA        string
	InstallationID int64
	Action         string // "opened", "synchronize", "comment"
}

type WebhookHandler struct {
	secret string
}

func NewWebhookHandler(secret string) *WebhookHandler {
	return &WebhookHandler{secret: secret}
}

func (h *WebhookHandler) VerifySignature(body []byte, signatureHeader string) bool {
	if signatureHeader == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	expected := "sha256=" + expectedMAC

	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) ([]byte, *PRInfo, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}
	defer r.Body.Close()

	sig := r.Header.Get("X-Hub-Signature-256")
	if !h.VerifySignature(body, sig) {
		return nil, nil, fmt.Errorf("invalid signature")
	}

	eventType := r.Header.Get("X-GitHub-Event")
	switch eventType {
	case "pull_request":
		return h.handlePREvent(body)
	case "issue_comment":
		return h.handleCommentEvent(body)
	default:
		return body, nil, nil
	}
}

func (h *WebhookHandler) handlePREvent(body []byte) ([]byte, *PRInfo, error) {
	event := &gh.PullRequestEvent{}
	if err := json.Unmarshal(body, event); err != nil {
		return nil, nil, fmt.Errorf("unmarshal PR event: %w", err)
	}

	pr := event.GetPullRequest()
	if pr == nil || pr.Base == nil || pr.Head == nil {
		return nil, nil, fmt.Errorf("invalid PR event: missing pull request data")
	}

	if event.GetAction() != "opened" && event.GetAction() != "synchronize" {
		return body, nil, nil
	}

	info := buildPRInfo(event.GetRepo(), pr, event.GetInstallation(), event.GetAction())
	return body, info, nil
}

func (h *WebhookHandler) handleCommentEvent(body []byte) ([]byte, *PRInfo, error) {
	event := &gh.IssueCommentEvent{}
	if err := json.Unmarshal(body, event); err != nil {
		return nil, nil, fmt.Errorf("unmarshal comment event: %w", err)
	}

	// Only handle created comments, not edits or deletions
	if event.GetAction() != "created" {
		return body, nil, nil
	}

	// Must be a PR comment, not a regular issue comment
	issue := event.GetIssue()
	if issue == nil || !issue.IsPullRequest() {
		return body, nil, nil
	}

	// Check for trigger phrase
	comment := event.GetComment()
	if comment == nil || !isReviewTrigger(comment.GetBody()) {
		return body, nil, nil
	}

	// Build PR info from the issue (which is a PR)
	info := &PRInfo{
		Owner:          event.GetRepo().GetOwner().GetLogin(),
		Repo:           event.GetRepo().GetName(),
		PRNumber:       issue.GetNumber(),
		InstallationID: event.GetInstallation().GetID(),
		Action:         "comment",
	}

	return body, info, nil
}

func isReviewTrigger(body string) bool {
	triggers := []string{"@prreviewer-app review", "@pr-reviewer review"}
	for _, t := range triggers {
		if body == t || len(body) >= len(t) && body[:len(t)] == t {
			return true
		}
	}
	return false
}

func buildPRInfo(repo *gh.Repository, pr *gh.PullRequest, install *gh.Installation, action string) *PRInfo {
	return &PRInfo{
		Owner:          repo.GetOwner().GetLogin(),
		Repo:           repo.GetName(),
		PRNumber:       pr.GetNumber(),
		BaseRef:        pr.Base.GetRef(),
		HeadRef:        pr.Head.GetRef(),
		BaseSHA:        pr.Base.GetSHA(),
		HeadSHA:        pr.Head.GetSHA(),
		InstallationID: install.GetID(),
		Action:         action,
	}
}
