package analyzer

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicClient struct {
	client *anthropic.Client
}

func NewAnthropicClient(apiKey string) LLMClient {
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &anthropicClient{client: &c}
}

func (c *anthropicClient) Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
	})
	if err != nil {
		return "", err
	}

	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", nil
}
