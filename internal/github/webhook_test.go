package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	gh "github.com/google/go-github/v69/github"
)

func TestVerifySignature_Valid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	h := NewWebhookHandler(secret)
	if !h.VerifySignature(body, expected) {
		t.Error("expected valid signature")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action":"opened"}`)

	h := NewWebhookHandler(secret)
	if h.VerifySignature(body, "sha256=deadbeef") {
		t.Error("expected invalid signature")
	}
}

func TestVerifySignature_EmptyHeader(t *testing.T) {
	h := NewWebhookHandler("secret")
	if h.VerifySignature([]byte("body"), "") {
		t.Error("empty header should be invalid")
	}
}

func TestParsePREvent_Opened(t *testing.T) {
	event := &gh.PullRequestEvent{
		Action: gh.Ptr("opened"),
		Repo: &gh.Repository{
			Owner: &gh.User{Login: gh.Ptr("testowner")},
			Name:  gh.Ptr("testrepo"),
		},
		PullRequest: &gh.PullRequest{
			Number: gh.Ptr(42),
			Base: &gh.PullRequestBranch{
				Ref: gh.Ptr("main"),
				SHA: gh.Ptr("abc123"),
			},
			Head: &gh.PullRequestBranch{
				Ref: gh.Ptr("feature-x"),
				SHA: gh.Ptr("def456"),
			},
		},
		Installation: &gh.Installation{
			ID: gh.Ptr(int64(12345)),
		},
	}

	body, _ := json.Marshal(event)
	h := NewWebhookHandler("secret")
	info, err := h.ParsePREvent(body)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Owner != "testowner" {
		t.Errorf("expected owner testowner, got %s", info.Owner)
	}
	if info.Repo != "testrepo" {
		t.Errorf("expected repo testrepo, got %s", info.Repo)
	}
	if info.PRNumber != 42 {
		t.Errorf("expected PR 42, got %d", info.PRNumber)
	}
	if info.BaseSHA != "abc123" {
		t.Errorf("expected base abc123, got %s", info.BaseSHA)
	}
	if info.HeadSHA != "def456" {
		t.Errorf("expected head def456, got %s", info.HeadSHA)
	}
	if info.InstallationID != 12345 {
		t.Errorf("expected installation 12345, got %d", info.InstallationID)
	}
	if info.Action != "opened" {
		t.Errorf("expected action opened, got %s", info.Action)
	}
}

func TestParsePREvent_InvalidJSON(t *testing.T) {
	h := NewWebhookHandler("secret")
	_, err := h.ParsePREvent([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHandle_NonPREvent(t *testing.T) {
	h := NewWebhookHandler("secret")
	bodyContent := []byte(`{"action":"created"}`)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(bodyContent)))
	req.Header.Set("X-GitHub-Event", "issues")

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(bodyContent)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	body, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil {
		t.Error("expected body to be returned")
	}
	if info != nil {
		t.Error("expected nil info for non-PR event")
	}
}

func TestParsePREvent_ReviewRequested(t *testing.T) {
	// Use raw JSON because gh.PullRequestEvent doesn't include requested_reviewer
	raw := `{
		"action": "review_requested",
		"pull_request": {
			"number": 42,
			"base": {"ref": "main", "sha": "abc123"},
			"head": {"ref": "feature-x", "sha": "def456"}
		},
		"repository": {
			"owner": {"login": "testowner"},
			"name": "testrepo"
		},
		"installation": {"id": 12345},
		"requested_reviewer": {"login": "pr-reviewer[bot]"}
	}`

	h := NewWebhookHandler("secret")
	info, err := h.ParsePREvent([]byte(raw))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Action != "review_requested" {
		t.Errorf("expected action 'review_requested', got '%s'", info.Action)
	}
	if info.RequestedReviewerLogin != "pr-reviewer[bot]" {
		t.Errorf("expected reviewer 'pr-reviewer[bot]', got '%s'", info.RequestedReviewerLogin)
	}
}

func TestParsePREvent_ReviewRequested_NoReviewer(t *testing.T) {
	raw := `{
		"action": "review_requested",
		"pull_request": {
			"number": 1,
			"base": {"ref": "main", "sha": "a"},
			"head": {"ref": "feat", "sha": "b"}
		},
		"repository": {
			"owner": {"login": "o"},
			"name": "r"
		},
		"installation": {"id": 1}
	}`

	h := NewWebhookHandler("secret")
	info, err := h.ParsePREvent([]byte(raw))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.RequestedReviewerLogin != "" {
		t.Errorf("expected empty reviewer, got '%s'", info.RequestedReviewerLogin)
	}
}

func TestHandle_ReviewRequestedAccepted(t *testing.T) {
	raw := `{
		"action": "review_requested",
		"pull_request": {
			"number": 42,
			"base": {"ref": "main", "sha": "abc123"},
			"head": {"ref": "feature-x", "sha": "def456"}
		},
		"repository": {
			"owner": {"login": "o"},
			"name": "r"
		},
		"installation": {"id": 1}
	}`

	h := NewWebhookHandler("secret")
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(raw))
	req.Header.Set("X-GitHub-Event", "pull_request")

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(raw))
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected info for review_requested event")
	}
	if info.Action != "review_requested" {
		t.Errorf("expected action 'review_requested', got '%s'", info.Action)
	}
}

func TestHandle_SkipNonOpenedActions(t *testing.T) {
	event := &gh.PullRequestEvent{
		Action: gh.Ptr("closed"),
		Repo: &gh.Repository{
			Owner: &gh.User{Login: gh.Ptr("o")},
			Name:  gh.Ptr("r"),
		},
		PullRequest: &gh.PullRequest{
			Number: gh.Ptr(1),
			Base:   &gh.PullRequestBranch{Ref: gh.Ptr("main"), SHA: gh.Ptr("a")},
			Head:   &gh.PullRequestBranch{Ref: gh.Ptr("b"), SHA: gh.Ptr("c")},
		},
		Installation: &gh.Installation{ID: gh.Ptr(int64(1))},
	}
	body, _ := json.Marshal(event)

	h := NewWebhookHandler("secret")
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "pull_request")

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	rec := httptest.NewRecorder()
	_, info, err := h.Handle(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Handle returns parsed info for all PR events; caller checks Action to decide whether to process
	if info == nil {
		t.Error("expected info to be returned for PR event")
	}
	if info.Action != "closed" {
		t.Errorf("expected action 'closed', got '%s'", info.Action)
	}
}
