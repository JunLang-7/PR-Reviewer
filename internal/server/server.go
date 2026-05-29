package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/junlang/PRReviewer/internal/analyzer"
	"github.com/junlang/PRReviewer/internal/comment"
	prcontext "github.com/junlang/PRReviewer/internal/context"
	"github.com/junlang/PRReviewer/internal/github"
)

type Server struct {
	contextBuilder *prcontext.Builder
	pipeline       *analyzer.Pipeline
	publisher      *comment.Publisher
	webhookHandler *github.WebhookHandler
	webhookSecret  string
}

func New(
	builder *prcontext.Builder,
	pipeline *analyzer.Pipeline,
	pub *comment.Publisher,
	webhookHandler *github.WebhookHandler,
) *Server {
	return &Server{
		contextBuilder: builder,
		pipeline:       pipeline,
		publisher:      pub,
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

func (s *Server) processPR(info *github.PRInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("processing PR #%d (action=%s) in %s/%s", info.PRNumber, info.Action, info.Owner, info.Repo)

	prCtx, err := s.contextBuilder.Build(ctx, info.Owner, info.Repo, info.BaseSHA, info.HeadSHA)
	if err != nil {
		log.Printf("context build error: %v", err)
		return
	}

	result, err := s.pipeline.Run(ctx, analyzer.PipelineInput{
		Diff:           buildDiffString(prCtx),
		FileContents:   prCtx.FileContents,
		Stage3Eligible: prCtx.Stage3Eligible,
	})
	if err != nil {
		log.Printf("pipeline error: %v", err)
	}

	if err := s.publisher.Publish(ctx, info.Owner, info.Repo, info.PRNumber, result); err != nil {
		log.Printf("publish comment error: %v", err)
	}

	log.Printf("PR #%d review complete", info.PRNumber)
}

func buildDiffString(prCtx *prcontext.PRContext) string {
	var result string
	for _, f := range prCtx.DiffFiles {
		result += "--- a/" + f.Path + "\n+++ b/" + f.Path + "\n" + f.Patch + "\n"
	}
	return result
}
