package context

import (
	"context"

	"github.com/google/go-github/v69/github"
)

// RepositoryClient abstracts GitHub API calls needed for context building.
// This allows mocking in tests.
type RepositoryClient interface {
	CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
}

type DiffFile struct {
	Path      string
	Patch     string
	Additions int
	Deletions int
	Changes   int
	Skipped   bool
}

type PRContext struct {
	Owner            string
	Repo             string
	PRNumber         int
	BaseRef          string
	HeadRef          string
	DiffFiles        []DiffFile
	FileContents     map[string]string
	TotalFiles       int
	TotalDiffLines   int
	Stage3Eligible   bool
}

type Builder struct {
	client RepositoryClient
}

func NewBuilder(client RepositoryClient) *Builder {
	return &Builder{client: client}
}

func (b *Builder) eligibleForStage3(files []DiffFile, totalDiffLines int) bool {
	return len(files) <= 5 && totalDiffLines < 500
}

func isBinaryFile(patch string) bool {
	return patch == "" || hasBinaryIndicator(patch)
}

func hasBinaryIndicator(patch string) bool {
	return len(patch) > 0 && (patch[:4] == "Bina" || patch[:6] == "Binary")
}

func diffLines(patch string) int {
	if patch == "" {
		return 0
	}
	lines := 0
	for _, c := range patch {
		if c == '\n' {
			lines++
		}
	}
	return lines
}

func (b *Builder) Build(ctx context.Context, owner, repo, baseSHA, headSHA string) (*PRContext, error) {
	comparison, _, err := b.client.CompareCommits(ctx, owner, repo, baseSHA, headSHA, nil)
	if err != nil {
		return nil, err
	}

	var files []DiffFile
	totalDiff := 0
	for _, f := range comparison.Files {
		patch := f.GetPatch()
		df := DiffFile{
			Path:      f.GetFilename(),
			Patch:     patch,
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Changes:   f.GetChanges(),
			Skipped:   isBinaryFile(patch),
		}
		files = append(files, df)
		totalDiff += diffLines(patch)
	}

	// L1: Fetch full content for changed files (limit 20, skip >500KB)
	fileContents := make(map[string]string)
	for i := range files {
		if len(fileContents) >= 20 || files[i].Skipped {
			continue
		}
		content, err := b.fetchFileContent(ctx, owner, repo, files[i].Path, headSHA)
		if err != nil || len(content) > 500*1024 {
			files[i].Skipped = true
			continue
		}
		fileContents[files[i].Path] = content
	}

	return &PRContext{
		Owner:          owner,
		Repo:           repo,
		DiffFiles:      files,
		FileContents:   fileContents,
		TotalFiles:     len(files),
		TotalDiffLines: totalDiff,
		Stage3Eligible: b.eligibleForStage3(files, totalDiff),
	}, nil
}

func (b *Builder) fetchFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	file, _, _, err := b.client.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", err
	}
	content, err := file.GetContent()
	if err != nil {
		return "", err
	}
	return content, nil
}
