package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GitHubAppID          int64
	GitHubAppPrivateKey  string
	GitHubWebhookSecret  string
	AnthropicAPIKey      string
	Port                 int
}

func Load() (*Config, error) {
	appID, err := strconv.ParseInt(requireEnv("GITHUB_APP_ID"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}

	cfg := &Config{
		GitHubAppID:         appID,
		GitHubAppPrivateKey: requireEnv("GITHUB_APP_PRIVATE_KEY"),
		GitHubWebhookSecret: requireEnv("GITHUB_WEBHOOK_SECRET"),
		AnthropicAPIKey:     requireEnv("ANTHROPIC_API_KEY"),
		Port:                envInt("PORT", 8080),
	}
	return cfg, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
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
