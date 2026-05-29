package config

import (
	"os"
	"testing"
)

func TestLoad_AllRequiredVars(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "12345")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "test-key-content")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "test-secret")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitHubAppID != 12345 {
		t.Errorf("expected app ID 12345, got %d", cfg.GitHubAppID)
	}
	if cfg.GitHubAppPrivateKey != "test-key-content" {
		t.Errorf("expected private key 'test-key-content', got %s", cfg.GitHubAppPrivateKey)
	}
	if cfg.GitHubWebhookSecret != "test-secret" {
		t.Errorf("expected webhook secret 'test-secret', got %s", cfg.GitHubWebhookSecret)
	}
	if cfg.AnthropicAPIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got %s", cfg.AnthropicAPIKey)
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "1")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "k")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "s")
	os.Setenv("ANTHROPIC_API_KEY", "a")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	cfg, _ := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "1")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "k")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "s")
	os.Setenv("ANTHROPIC_API_KEY", "a")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("PORT")
	}()

	cfg, _ := Load()
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestLoad_MissingRequiredVar(t *testing.T) {
	tests := []struct {
		name    string
		unset   string
		setVars map[string]string
	}{
		{"missing app ID", "GITHUB_APP_ID", map[string]string{
			"GITHUB_APP_PRIVATE_KEY": "k",
			"GITHUB_WEBHOOK_SECRET":  "s",
			"ANTHROPIC_API_KEY":      "a",
		}},
		{"missing private key", "GITHUB_APP_PRIVATE_KEY", map[string]string{
			"GITHUB_APP_ID":         "1",
			"GITHUB_WEBHOOK_SECRET": "s",
			"ANTHROPIC_API_KEY":     "a",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.setVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.setVars {
					os.Unsetenv(k)
				}
			}()

			// requireEnv panics, so we need recover
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic, got none")
				}
			}()
			Load()
		})
	}
}
