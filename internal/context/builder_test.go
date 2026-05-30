package context

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/go-github/v69/github"
)

type mockRepoClient struct {
	compareResult *github.CommitsComparison
	contents      map[string]*github.RepositoryContent
}

func (m *mockRepoClient) CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error) {
	return m.compareResult, nil, nil
}

func (m *mockRepoClient) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	content, ok := m.contents[path]
	if !ok {
		return nil, nil, nil, &github.ErrorResponse{}
	}
	return content, nil, nil, nil
}

func encodeContent(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestBuilder_Build_BasicPR(t *testing.T) {
	mock := &mockRepoClient{
		compareResult: &github.CommitsComparison{
			Files: []*github.CommitFile{
				{
					Filename:  github.Ptr("main.go"),
					Patch:     github.Ptr("@@ -1,3 +1,4 @@\n package main\n+import \"fmt\"\n func main() {}"),
					Additions: github.Ptr(1),
					Deletions: github.Ptr(0),
					Changes:   github.Ptr(1),
				},
			},
		},
		contents: map[string]*github.RepositoryContent{
			"main.go": {
				Content: github.Ptr(encodeContent("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")),
			},
		},
	}

	b := NewBuilder(mock)
	ctx := context.Background()
	result, err := b.Build(ctx, "owner", "repo", "abc", "def")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", result.TotalFiles)
	}
	if len(result.DiffFiles) != 1 {
		t.Errorf("expected 1 diff file, got %d", len(result.DiffFiles))
	}
	if result.DiffFiles[0].Skipped {
		t.Error("should not skip main.go")
	}
	if result.TotalDiffLines == 0 {
		t.Error("expected non-zero diff lines")
	}
	content, ok := result.FileContents["main.go"]
	if !ok {
		t.Error("expected main.go content")
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestBuilder_Build_SkipsBinaryFiles(t *testing.T) {
	mock := &mockRepoClient{
		compareResult: &github.CommitsComparison{
			Files: []*github.CommitFile{
				{
					Filename:  github.Ptr("image.png"),
					Patch:     github.Ptr(""),
					Additions: github.Ptr(0),
					Deletions: github.Ptr(0),
					Changes:   github.Ptr(0),
				},
			},
		},
		contents: map[string]*github.RepositoryContent{},
	}

	b := NewBuilder(mock)
	result, err := b.Build(context.Background(), "o", "r", "a", "b")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.DiffFiles) != 1 {
		t.Fatalf("expected 1 diff file, got %d", len(result.DiffFiles))
	}
	if !result.DiffFiles[0].Skipped {
		t.Error("binary file should be marked as skipped")
	}
}



func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		patch    string
		expected bool
	}{
		{"", true},
		{"Binary files differ", true},
		{"@@ -1,3 +1,4 @@\n", false},
		{"normal code diff", false},
	}

	for _, tt := range tests {
		t.Run(tt.patch, func(t *testing.T) {
			got := isBinaryFile(tt.patch)
			if got != tt.expected {
				t.Errorf("isBinaryFile(%q) = %v, want %v", tt.patch, got, tt.expected)
			}
		})
	}
}
