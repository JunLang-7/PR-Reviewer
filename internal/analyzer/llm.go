package analyzer

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicClient struct {
	client *anthropic.Client
}

func NewAnthropicClient(apiKey, baseURL string) LLMClient {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" && baseURL != "https://api.anthropic.com" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	c := anthropic.NewClient(opts...)
	return &anthropicClient{client: &c}
}

func (c *anthropicClient) Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 32768,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
	})

	var sb strings.Builder
	for stream.Next() {
		event := stream.Current()
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			sb.WriteString(event.Delta.Text)
		}
	}
	if err := stream.Err(); err != nil {
		return "", err
	}

	result := sb.String()
	fmt.Printf("[llm] response: streaming, text=%d bytes\n", len(result))
	return result, nil
}
