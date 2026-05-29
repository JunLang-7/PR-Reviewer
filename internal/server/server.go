package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v69/github"

	"github.com/junlang/PRReviewer/internal/analyzer"
	"github.com/junlang/PRReviewer/internal/comment"
	prcontext "github.com/junlang/PRReviewer/internal/context"
	ghclient "github.com/junlang/PRReviewer/internal/github"
)

type Server struct {
	appClient      *ghclient.Client
	pipeline       *analyzer.Pipeline
	webhookHandler *ghclient.WebhookHandler
}

func New(appClient *ghclient.Client, pipeline *analyzer.Pipeline, webhookHandler *ghclient.WebhookHandler) *Server {
	return &Server{
		appClient:      appClient,
		pipeline:       pipeline,
		webhookHandler: webhookHandler,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	_, info, err := s.webhookHandler.Handle(w, r)
	if err != nil {
		log.Printf("webhook error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if info == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if info.Action != "opened" && info.Action != "synchronize" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	go s.processPR(info)
}

func (s *Server) processPR(info *ghclient.PRInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("processing PR #%d (action=%s) in %s/%s", info.PRNumber, info.Action, info.Owner, info.Repo)

	// Create per-request installation client
	ghClient, err := s.appClient.NewInstallationClient(ctx, info.InstallationID)
	if err != nil {
		log.Printf("create installation client error: %v", err)
		return
	}

	// Build context
	builder := prcontext.NewBuilder(ghClient.Repositories)
	prCtx, err := builder.Build(ctx, info.Owner, info.Repo, info.BaseSHA, info.HeadSHA)
	if err != nil {
		log.Printf("context build error: %v", err)
		s.postErrorComment(ctx, ghClient, info, "获取 PR 上下文失败: "+err.Error())
		return
	}

	// Run pipeline
	result, err := s.pipeline.Run(ctx, analyzer.PipelineInput{
		Diff:           buildDiffString(prCtx),
		FileContents:   prCtx.FileContents,
		Stage3Eligible: prCtx.Stage3Eligible,
	})
	if err != nil {
		log.Printf("pipeline error: %v", err)
	}

	// Publish comment
	publisher := comment.NewPublisher(ghClient.Issues)
	if err := publisher.Publish(ctx, info.Owner, info.Repo, info.PRNumber, result); err != nil {
		log.Printf("publish comment error: %v", err)
	}

	log.Printf("PR #%d review complete", info.PRNumber)
}

func (s *Server) postErrorComment(ctx context.Context, ghClient *github.Client, info *ghclient.PRInfo, msg string) {
	body := "## AI Review Error\n\n> " + msg + "\n\n---\n> AI Reviewer v1"
	comment := &github.IssueComment{Body: &body}
	_, _, err := ghClient.Issues.CreateComment(ctx, info.Owner, info.Repo, info.PRNumber, comment)
	if err != nil {
		log.Printf("post error comment failed: %v", err)
	}
}

func buildDiffString(prCtx *prcontext.PRContext) string {
	var result string
	for _, f := range prCtx.DiffFiles {
		result += "--- a/" + f.Path + "\n+++ b/" + f.Path + "\n" + f.Patch + "\n"
	}
	return result
}
