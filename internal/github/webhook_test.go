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

func TestHandle_PREvent_Opened(t *testing.T) {
	event := gh.PullRequestEvent{
		Action: gh.Ptr("opened"),
		Repo:   &gh.Repository{Owner: &gh.User{Login: gh.Ptr("testowner")}, Name: gh.Ptr("testrepo")},
		PullRequest: &gh.PullRequest{
			Number: gh.Ptr(42),
			Base:   &gh.PullRequestBranch{Ref: gh.Ptr("main"), SHA: gh.Ptr("abc123")},
			Head:   &gh.PullRequestBranch{Ref: gh.Ptr("feature"), SHA: gh.Ptr("def456")},
		},
		Installation: &gh.Installation{ID: gh.Ptr(int64(1))},
	}
	body, _ := json.Marshal(event)

	h := NewWebhookHandler("secret")
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", sign(body, "secret"))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected info for opened event")
	}
	if info.Action != "opened" {
		t.Errorf("expected 'opened', got '%s'", info.Action)
	}
	if info.PRNumber != 42 {
		t.Errorf("expected PR 42, got %d", info.PRNumber)
	}
	if info.Owner != "testowner" {
		t.Errorf("expected owner 'testowner', got '%s'", info.Owner)
	}
}

func TestHandle_PREvent_Closed(t *testing.T) {
	event := gh.PullRequestEvent{
		Action: gh.Ptr("closed"),
		Repo:   &gh.Repository{Owner: &gh.User{Login: gh.Ptr("o")}, Name: gh.Ptr("r")},
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
	req.Header.Set("X-Hub-Signature-256", sign(body, "secret"))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Error("expected nil info for closed event")
	}
}

func TestHandle_CommentEvent_Trigger(t *testing.T) {
	raw := `{
		"action": "created",
		"issue": {
			"number": 42,
			"pull_request": {"url": "https://api.github.com/repos/o/r/pulls/42"}
		},
		"comment": {"body": "@prreviewer-app review"},
		"repository": {"owner": {"login": "o"}, "name": "r"},
		"installation": {"id": 1}
	}`

	h := NewWebhookHandler("secret")
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(raw))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	req.Header.Set("X-Hub-Signature-256", sign([]byte(raw), "secret"))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected info for trigger comment")
	}
	if info.Action != "comment" {
		t.Errorf("expected 'comment', got '%s'", info.Action)
	}
	if info.PRNumber != 42 {
		t.Errorf("expected PR 42, got %d", info.PRNumber)
	}
}

func TestHandle_CommentEvent_NotTrigger(t *testing.T) {
	raw := `{
		"action": "created",
		"issue": {
			"number": 1,
			"pull_request": {"url": "https://api.github.com/repos/o/r/pulls/1"}
		},
		"comment": {"body": "looks good to me"},
		"repository": {"owner": {"login": "o"}, "name": "r"},
		"installation": {"id": 1}
	}`

	h := NewWebhookHandler("secret")
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(raw))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	req.Header.Set("X-Hub-Signature-256", sign([]byte(raw), "secret"))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Error("expected nil info for non-trigger comment")
	}
}

func TestHandle_CommentEvent_NotPR(t *testing.T) {
	raw := `{
		"action": "created",
		"issue": {"number": 1},
		"comment": {"body": "@prreviewer-app review"},
		"repository": {"owner": {"login": "o"}, "name": "r"},
		"installation": {"id": 1}
	}`

	h := NewWebhookHandler("secret")
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(raw))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	req.Header.Set("X-Hub-Signature-256", sign([]byte(raw), "secret"))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Error("expected nil info for non-PR issue comment")
	}
}

func TestHandle_NonPREvent(t *testing.T) {
	h := NewWebhookHandler("secret")
	body := []byte(`{"action":"created"}`)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", sign(body, "secret"))

	_, info, err := h.Handle(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Error("expected nil info for non-PR event")
	}
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
