package chat

import (
	"time"
)

// ClientType defines the type of chat client to create
type ClientType string

const (
	ClientTypeOriginal  ClientType = "original"
	ClientTypeLangChain ClientType = "langchain"
)

// ClientConfig holds configuration for creating chat clients
type ClientConfig struct {
	BaseURL      string
	Model        string
	Timeout      time.Duration
	ClientType   ClientType
	UseStreaming bool
}

// NewChatClient creates a chat client based on configuration
func NewChatClient(config ClientConfig) (ChatClient, error) {
	switch config.ClientType {
	case ClientTypeLangChain:
		if config.UseStreaming {
			return NewLangChainStreamingClientWithTimeout(config.BaseURL, config.Model, config.Timeout)
		}
		return NewLangChainClientWithTimeout(config.BaseURL, config.Model, config.Timeout)
	case ClientTypeOriginal:
		fallthrough
	default:
		if config.UseStreaming {
			return NewStreamingClientWithTimeout(config.BaseURL, config.Timeout), nil
		}
		return NewClientWithTimeout(config.BaseURL, config.Timeout), nil
	}
}

// NewStreamingChatClient creates a streaming chat client based on configuration
func NewStreamingChatClient(config ClientConfig) (StreamingChatClient, error) {
	switch config.ClientType {
	case ClientTypeLangChain:
		return NewLangChainStreamingClientWithTimeout(config.BaseURL, config.Model, config.Timeout)
	case ClientTypeOriginal:
		fallthrough
	default:
		return NewStreamingClientWithTimeout(config.BaseURL, config.Timeout), nil
	}
}

// DefaultConfig returns a default client configuration
func DefaultConfig() ClientConfig {
	return ClientConfig{
		BaseURL:      "http://localhost:11434",
		Model:        "",
		Timeout:      60 * time.Second,
		ClientType:   ClientTypeOriginal, // Start with original for compatibility
		UseStreaming: false,
	}
}

// LangChainConfig returns a configuration for LangChain Go client
func LangChainConfig(baseURL, model string) ClientConfig {
	return ClientConfig{
		BaseURL:      baseURL,
		Model:        model,
		Timeout:      60 * time.Second,
		ClientType:   ClientTypeLangChain,
		UseStreaming: false,
	}
}

// LangChainStreamingConfig returns a configuration for LangChain Go streaming client
func LangChainStreamingConfig(baseURL, model string) ClientConfig {
	return ClientConfig{
		BaseURL:      baseURL,
		Model:        model,
		Timeout:      60 * time.Second,
		ClientType:   ClientTypeLangChain,
		UseStreaming: true,
	}
}
