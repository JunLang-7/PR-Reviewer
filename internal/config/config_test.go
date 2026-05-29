package config

import (
	"os"
	"testing"
)

func TestLoad_AllRequiredVars(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "12345")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "test-key-content")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "test-secret")
	os.Setenv("LLM_API_KEY", "test-api-key")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("LLM_API_KEY")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitHubAppID != 12345 {
		t.Errorf("expected app ID 12345, got %d", cfg.GitHubAppID)
	}
	if cfg.LLMBaseURL != "https://api.anthropic.com" {
		t.Errorf("expected default base URL, got %s", cfg.LLMBaseURL)
	}
}

func TestLoad_BackwardCompat_AnthropicAPIKey(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "1")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "k")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "s")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-old-key")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	cfg, _ := Load()
	if cfg.LLMAPIKey != "sk-ant-old-key" {
		t.Errorf("expected LLMAPIKey from ANTHROPIC_API_KEY, got %s", cfg.LLMAPIKey)
	}
}

func TestLoad_LLMConfigDefaults(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "1")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "k")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "s")
	os.Setenv("LLM_API_KEY", "a")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("LLM_API_KEY")
	}()

	cfg, _ := Load()
	if cfg.LLMBaseURL != "https://api.anthropic.com" {
		t.Errorf("expected default LLM base URL, got %s", cfg.LLMBaseURL)
	}
	if cfg.LLMModelFast != "claude-haiku-4-5" {
		t.Errorf("expected default fast model, got %s", cfg.LLMModelFast)
	}
	if cfg.LLMModelPowerful != "claude-sonnet-4-6" {
		t.Errorf("expected default powerful model, got %s", cfg.LLMModelPowerful)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
}

func TestLoad_CustomLLMConfig(t *testing.T) {
	os.Setenv("GITHUB_APP_ID", "1")
	os.Setenv("GITHUB_APP_PRIVATE_KEY", "k")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "s")
	os.Setenv("LLM_API_KEY", "sk-deepseek")
	os.Setenv("LLM_BASE_URL", "https://api.deepseek.com/anthropic")
	os.Setenv("LLM_MODEL_FAST", "deepseek-v4-flash")
	os.Setenv("LLM_MODEL_POWERFUL", "deepseek-v4-pro")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("GITHUB_APP_ID")
		os.Unsetenv("GITHUB_APP_PRIVATE_KEY")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("LLM_API_KEY")
		os.Unsetenv("LLM_BASE_URL")
		os.Unsetenv("LLM_MODEL_FAST")
		os.Unsetenv("LLM_MODEL_POWERFUL")
		os.Unsetenv("PORT")
	}()

	cfg, _ := Load()
	if cfg.LLMBaseURL != "https://api.deepseek.com/anthropic" {
		t.Errorf("expected DeepSeek base URL, got %s", cfg.LLMBaseURL)
	}
	if cfg.LLMModelFast != "deepseek-v4-flash" {
		t.Errorf("expected deepseek-v4-flash, got %s", cfg.LLMModelFast)
	}
	if cfg.LLMModelPowerful != "deepseek-v4-pro" {
		t.Errorf("expected deepseek-v4-pro, got %s", cfg.LLMModelPowerful)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestLoad_MissingRequiredVar(t *testing.T) {
	tests := []struct {
		name    string
		setVars map[string]string
	}{
		{"missing app ID", map[string]string{
			"GITHUB_APP_PRIVATE_KEY": "k",
			"GITHUB_WEBHOOK_SECRET":  "s",
			"LLM_API_KEY":            "a",
		}},
		{"missing api key", map[string]string{
			"GITHUB_APP_ID":         "1",
			"GITHUB_APP_PRIVATE_KEY": "k",
			"GITHUB_WEBHOOK_SECRET": "s",
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

			cfg, err := Load()
			if err == nil {
				t.Error("expected error for missing env var, got nil")
			}
			if cfg != nil {
				t.Error("expected nil config on error")
			}
		})
	}
}
