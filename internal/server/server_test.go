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
	"github.com/junlang/PRReviewer/internal/comment"
	prcontext "github.com/junlang/PRReviewer/internal/context"
	"github.com/junlang/PRReviewer/internal/github"
)

const testSecret = "test-secret"

func signBody(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// --- mocks ---

type mockGHClient struct{}

func (m *mockGHClient) CompareCommits(ctx context.Context, owner, repo, base, head string) (*gh.CommitsComparison, *gh.Response, error) {
	return &gh.CommitsComparison{
		Files: []*gh.CommitFile{
			{Filename: gh.Ptr("main.go"), Patch: gh.Ptr("@@ -1 +1 @@"), Additions: gh.Ptr(1), Deletions: gh.Ptr(0), Changes: gh.Ptr(1)},
		},
	}, nil, nil
}
func (m *mockGHClient) GetContents(ctx context.Context, owner, repo, path string, opts *gh.RepositoryContentGetOptions) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error) {
	return &gh.RepositoryContent{Content: gh.Ptr("Y29udGVudA==")}, nil, nil, nil
}

type mockLLM struct{}

func (m *mockLLM) Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	return "mock AI response", nil
}

type mockCommentClient struct {
	comments []*gh.IssueComment
}

func (m *mockCommentClient) CreateComment(ctx context.Context, owner, repo string, number int, c *gh.IssueComment) (*gh.IssueComment, *gh.Response, error) {
	m.comments = append(m.comments, c)
	return c, nil, nil
}

func newTestServer() (*Server, *mockCommentClient) {
	mockCC := &mockCommentClient{}
	pub := comment.NewPublisher(mockCC)
	srv := New(
		prcontext.NewBuilder(&mockGHClient{}),
		analyzer.NewPipeline(&mockLLM{}, "fast", "power"),
		pub,
		github.NewWebhookHandler(testSecret),
	)
	return srv, mockCC
}

func TestServer_Webhook_NonPREvent(t *testing.T) {
	srv, _ := newTestServer()
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
	srv, _ := newTestServer()
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
	srv, _ := newTestServer()
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
	srv, _ := newTestServer()
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for health check, got %d", rec.Code)
	}
}
