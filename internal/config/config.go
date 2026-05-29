package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GitHubAppID         int64
	GitHubAppPrivateKey string
	GitHubWebhookSecret string
	AnthropicAPIKey     string
	Port                int
}

func Load() (*Config, error) {
	missing := []string{}

	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		missing = append(missing, "GITHUB_APP_ID")
	}
	privateKey := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if privateKey == "" {
		missing = append(missing, "GITHUB_APP_PRIVATE_KEY")
	}
	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if webhookSecret == "" {
		missing = append(missing, "GITHUB_WEBHOOK_SECRET")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		missing = append(missing, "ANTHROPIC_API_KEY")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing env vars: %v\n\n请复制 .env.example 为 .env 并填入配置:\n  cp .env.example .env", missing)
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}

	return &Config{
		GitHubAppID:         appID,
		GitHubAppPrivateKey: privateKey,
		GitHubWebhookSecret: webhookSecret,
		AnthropicAPIKey:     apiKey,
		Port:                envInt("PORT", 8080),
	}, nil
}

func envInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
