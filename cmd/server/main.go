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
	"github.com/junlang/PRReviewer/internal/logger"
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
	log.Printf("========================================")
	log.Printf("PR Reviewer 已启动")
	log.Printf("监听地址: %s", addr)
	log.Printf("Webhook:  POST %s/webhook", addr)
	log.Printf("健康检查: GET  %s/health", addr)
	log.Printf("LLM:      %s (fast=%s, power=%s)", cfg.LLMBaseURL, cfg.LLMModelFast, cfg.LLMModelPowerful)
	log.Printf("========================================")

	// Wrap handler with request logger
	handler := logger.RequestLogger(srv.Handler())

	log.Printf("等待 webhook 请求...")
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
