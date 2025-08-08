package tokens

import (
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides methods for counting tokens in text
type TokenCounter struct {
	encoder *tiktoken.Tiktoken
	mu      sync.RWMutex
}

// NewTokenCounter creates a new token counter with the specified model
func NewTokenCounter(modelName string) (*TokenCounter, error) {
	// Map model names to encoding names
	encodingName := getEncodingForModel(modelName)

	encoder, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		// Fallback to cl100k_base for most modern models
		encoder, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, err
		}
	}

	return &TokenCounter{
		encoder: encoder,
	}, nil
}

// CountTokens counts the number of tokens in the given text
func (tc *TokenCounter) CountTokens(text string) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	if tc.encoder == nil {
		// Fallback: rough estimation if encoder is not available
		return estimateTokens(text)
	}

	tokens := tc.encoder.Encode(text, nil, nil)
	return len(tokens)
}

// CountMessages counts tokens for a conversation with role-based messages
func (tc *TokenCounter) CountMessages(messages []Message) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	totalTokens := 0
	for _, msg := range messages {
		// Add tokens for role and content
		// Most models add special tokens for message boundaries
		totalTokens += tc.countSingleMessage(msg)
	}

	// Add tokens for message formatting overhead (approximate)
	totalTokens += 3 // Every reply is primed with assistant

	return totalTokens
}

func (tc *TokenCounter) countSingleMessage(msg Message) int {
	tokens := 0

	// Count role tokens (system, user, assistant)
	tokens += tc.CountTokens(msg.Role)

	// Count content tokens
	tokens += tc.CountTokens(msg.Content)

	// Add message boundary tokens (approximate for most models)
	tokens += 4 // <|start|>role<|end|> type markers

	return tokens
}

// Message represents a chat message with role and content
type Message struct {
	Role    string
	Content string
}

// getEncodingForModel returns the appropriate encoding for a model
func getEncodingForModel(modelName string) string {
	modelLower := strings.ToLower(modelName)

	// GPT-4 and GPT-3.5 models
	if strings.Contains(modelLower, "gpt-4") || strings.Contains(modelLower, "gpt-3.5") {
		return "cl100k_base"
	}

	// Older GPT-3 models
	if strings.Contains(modelLower, "davinci") || strings.Contains(modelLower, "curie") {
		return "p50k_base"
	}

	// Code models
	if strings.Contains(modelLower, "code") {
		return "p50k_base"
	}

	// Default to cl100k_base for most modern models
	// This works reasonably well for many models including local ones
	return "cl100k_base"
}

// estimateTokens provides a rough token estimation when encoder is not available
func estimateTokens(text string) int {
	// Rough estimation: ~4 characters per token on average
	// This is a very rough approximation
	words := strings.Fields(text)
	charCount := len(text)

	// Use a combination of word count and character count for estimation
	// Approximate: 1 token per word, or 1 token per 4 characters, whichever is higher
	wordEstimate := len(words)
	charEstimate := charCount / 4

	if wordEstimate > charEstimate {
		return wordEstimate
	}
	return charEstimate
}
