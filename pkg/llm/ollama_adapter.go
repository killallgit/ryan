package llm

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
)

// OllamaAdapter adapts the Ollama client to the Provider interface
type OllamaAdapter struct {
	client *ollama.OllamaClient
	model  string
}

// NewOllamaAdapter creates a new Ollama adapter
func NewOllamaAdapter() (*OllamaAdapter, error) {
	client := ollama.NewClient()
	model := viper.GetString("ollama.default_model")

	return &OllamaAdapter{
		client: client,
		model:  model,
	}, nil
}

// Generate generates a response for the given prompt
func (a *OllamaAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	response, err := a.client.GenerateContent(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("ollama generate error: %w", err)
	}

	if response != nil && response.Choices != nil && len(response.Choices) > 0 {
		return response.Choices[0].Content, nil
	}

	return "", fmt.Errorf("no response from ollama")
}

// GenerateStream generates a streaming response for the given prompt
func (a *OllamaAdapter) GenerateStream(ctx context.Context, prompt string, handler StreamHandler) error {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	streamFunc := func(ctx context.Context, chunk []byte) error {
		return handler.OnChunk(string(chunk))
	}

	_, err := a.client.GenerateContent(ctx, messages,
		llms.WithStreamingFunc(streamFunc))

	if err != nil {
		handler.OnError(err)
		return fmt.Errorf("ollama stream error: %w", err)
	}

	// Call OnComplete when streaming is done
	handler.OnComplete("")
	return nil
}

// GetName returns the provider name
func (a *OllamaAdapter) GetName() string {
	return "ollama"
}

// GetModel returns the current model name
func (a *OllamaAdapter) GetModel() string {
	return a.model
}

// GenerateWithHistory generates a response considering conversation history
func (a *OllamaAdapter) GenerateWithHistory(ctx context.Context, messages []Message) (string, error) {
	// Convert our messages to langchain messages
	lcMessages := make([]llms.MessageContent, 0, len(messages))

	for _, msg := range messages {
		var msgType llms.ChatMessageType
		switch msg.Role {
		case "user":
			msgType = llms.ChatMessageTypeHuman
		case "assistant":
			msgType = llms.ChatMessageTypeAI
		case "system":
			msgType = llms.ChatMessageTypeSystem
		default:
			msgType = llms.ChatMessageTypeGeneric
		}

		lcMessages = append(lcMessages, llms.TextParts(msgType, msg.Content))
	}

	response, err := a.client.GenerateContent(ctx, lcMessages)
	if err != nil {
		return "", fmt.Errorf("ollama generate with history error: %w", err)
	}

	if response != nil && response.Choices != nil && len(response.Choices) > 0 {
		return response.Choices[0].Content, nil
	}

	return "", fmt.Errorf("no response from ollama")
}

// GenerateStreamWithHistory generates a streaming response with history
func (a *OllamaAdapter) GenerateStreamWithHistory(ctx context.Context, messages []Message, handler StreamHandler) error {
	// Convert our messages to langchain messages
	lcMessages := make([]llms.MessageContent, 0, len(messages))

	for _, msg := range messages {
		var msgType llms.ChatMessageType
		switch msg.Role {
		case "user":
			msgType = llms.ChatMessageTypeHuman
		case "assistant":
			msgType = llms.ChatMessageTypeAI
		case "system":
			msgType = llms.ChatMessageTypeSystem
		default:
			msgType = llms.ChatMessageTypeGeneric
		}

		lcMessages = append(lcMessages, llms.TextParts(msgType, msg.Content))
	}

	streamFunc := func(ctx context.Context, chunk []byte) error {
		return handler.OnChunk(string(chunk))
	}

	_, err := a.client.GenerateContent(ctx, lcMessages,
		llms.WithStreamingFunc(streamFunc))

	if err != nil {
		handler.OnError(err)
		return fmt.Errorf("ollama stream with history error: %w", err)
	}

	handler.OnComplete("")
	return nil
}
