package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/junlang/PRReviewer/internal/analyzer"
	"github.com/junlang/PRReviewer/internal/comment"
	prcontext "github.com/junlang/PRReviewer/internal/context"
	"github.com/junlang/PRReviewer/internal/config"
	"github.com/junlang/PRReviewer/internal/github"
	"github.com/junlang/PRReviewer/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Initialize components
	ghAppClient := github.NewClient(cfg.GitHubAppID, cfg.GitHubAppPrivateKey)
	llmClient := analyzer.NewAnthropicClient(cfg.AnthropicAPIKey)
	pipeline := analyzer.NewPipeline(llmClient)
	webhookHandler := github.NewWebhookHandler(cfg.GitHubWebhookSecret)

	// The server creates per-request GitHub clients from installation IDs.
	// For demo, wire a basic server that will get full client injection later.
	_ = ghAppClient
	_ = pipeline
	_ = prcontext.NewBuilder
	_ = comment.NewPublisher

	srv := server.New(nil, nil, nil, webhookHandler)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("PR Reviewer starting on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
