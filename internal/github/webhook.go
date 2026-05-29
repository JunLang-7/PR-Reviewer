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
	Action         string // "opened", "synchronize", etc.
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

func (h *WebhookHandler) ParsePREvent(body []byte) (*PRInfo, error) {
	event := &gh.PullRequestEvent{}
	if err := json.Unmarshal(body, event); err != nil {
		return nil, fmt.Errorf("unmarshal PR event: %w", err)
	}

	pr := event.GetPullRequest()
	if pr == nil || pr.Base == nil || pr.Head == nil {
		return nil, fmt.Errorf("invalid PR event: missing pull request data")
	}

	info := &PRInfo{
		Owner:          event.GetRepo().GetOwner().GetLogin(),
		Repo:           event.GetRepo().GetName(),
		PRNumber:       pr.GetNumber(),
		BaseRef:        pr.Base.GetRef(),
		HeadRef:        pr.Head.GetRef(),
		BaseSHA:        pr.Base.GetSHA(),
		HeadSHA:        pr.Head.GetSHA(),
		InstallationID: event.GetInstallation().GetID(),
		Action:         event.GetAction(),
	}
	return info, nil
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
	if eventType != "pull_request" {
		return body, nil, nil // not a PR event, skip
	}

	info, err := h.ParsePREvent(body)
	if err != nil {
		return nil, nil, err
	}

	// Only handle opened and synchronize for v1
	if info.Action != "opened" && info.Action != "synchronize" {
		return body, info, nil // skip, no error
	}

	return body, info, nil
}
