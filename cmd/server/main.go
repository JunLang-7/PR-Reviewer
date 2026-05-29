package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/junlang/PRReviewer/internal/analyzer"
	"github.com/junlang/PRReviewer/internal/config"
	"github.com/junlang/PRReviewer/internal/github"
	"github.com/junlang/PRReviewer/internal/server"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	appClient := github.NewClient(cfg.GitHubAppID, cfg.GitHubAppPrivateKey)
	llmClient := analyzer.NewAnthropicClient(cfg.LLMAPIKey, cfg.LLMBaseURL)
	pipeline := analyzer.NewPipeline(llmClient, cfg.LLMModelFast, cfg.LLMModelPowerful)
	webhookHandler := github.NewWebhookHandler(cfg.GitHubWebhookSecret)

	srv := server.New(appClient, pipeline, webhookHandler)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("PR Reviewer starting on %s (LLM: %s, models: %s/%s)",
		addr, cfg.LLMBaseURL, cfg.LLMModelFast, cfg.LLMModelPowerful)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
