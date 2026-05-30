package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gh "github.com/google/go-github/v69/github"

	"github.com/junlang/PRReviewer/internal/analyzer"
	"github.com/junlang/PRReviewer/internal/github"
)

const testSecret = "test-secret"

func signBody(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// --- mocks ---

type mockLLM struct{}

func (m *mockLLM) Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	return "mock AI response", nil
}

type mockGHRepos struct {
	t *testing.T
}

func (m *mockGHRepos) CompareCommits(ctx context.Context, owner, repo, base, head string, opts *gh.ListOptions) (*gh.CommitsComparison, *gh.Response, error) {
	return &gh.CommitsComparison{
		Files: []*gh.CommitFile{
			{Filename: gh.Ptr("main.go"), Patch: gh.Ptr("@@ -1 +1 @@"), Additions: gh.Ptr(1), Deletions: gh.Ptr(0), Changes: gh.Ptr(1)},
		},
	}, nil, nil
}
func (m *mockGHRepos) GetContents(ctx context.Context, owner, repo, path string, opts *gh.RepositoryContentGetOptions) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error) {
	return &gh.RepositoryContent{Content: gh.Ptr("Y29udGVudA==")}, nil, nil, nil
}

type mockGHIssues struct {
	comments []*gh.IssueComment
}

func (m *mockGHIssues) CreateComment(ctx context.Context, owner, repo string, number int, c *gh.IssueComment) (*gh.IssueComment, *gh.Response, error) {
	m.comments = append(m.comments, c)
	return c, nil, nil
}

// mockAppClient mimics just enough of ghclient.Client for tests.
// We inject it as a *github.Client directly since that's all processPR needs.
type mockAppClient struct{}

func (m *mockAppClient) NewInstallationClient(ctx context.Context, installationID int64) (*gh.Client, error) {
	return &gh.Client{}, nil
}

// --- test server ---

func newTestServer(issues *mockGHIssues) (*Server, *mockGHIssues) {
	pipeline := analyzer.NewPipeline(&mockLLM{}, "fast", "power")
	webhookHandler := github.NewWebhookHandler(testSecret)
	// We need a real *ghclient.Client to create the server.
	// For tests we create a minimal one that returns nil clients.
	// processPR is tested via unit tests; integration is for e2e.
	return New(nil, pipeline, webhookHandler), issues
}

func TestServer_Webhook_NonPREvent(t *testing.T) {
	srv, _ := newTestServer(nil)
	handler := srv.Handler()

	body := []byte(`{"action":"created"}`)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", signBody(body))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-PR event, got %d", rec.Code)
	}
}

func TestServer_Webhook_WrongAction(t *testing.T) {
	srv, _ := newTestServer(nil)
	handler := srv.Handler()

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
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signBody(body))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for skipped action, got %d", rec.Code)
	}
}

func TestServer_Webhook_ProcessPR(t *testing.T) {
	srv, _ := newTestServer(nil)
	handler := srv.Handler()

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

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signBody(body))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}

func TestServer_HealthCheck(t *testing.T) {
	srv, _ := newTestServer(nil)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for health check, got %d", rec.Code)
	}
}

