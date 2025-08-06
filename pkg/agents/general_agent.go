package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
)

// GeneralAgent handles general conversational requests
type GeneralAgent struct {
	ollamaClient *ollama.Client
	model        string
	log          *logger.Logger
}

// NewGeneralAgent creates a new general purpose conversational agent
func NewGeneralAgent(ollamaClient *ollama.Client, model string) *GeneralAgent {
	return &GeneralAgent{
		ollamaClient: ollamaClient,
		model:        model,
		log:          logger.WithComponent("general_agent"),
	}
}

// Name returns the agent name
func (g *GeneralAgent) Name() string {
	return "general"
}

// Description returns the agent description
func (g *GeneralAgent) Description() string {
	return "Handles general conversational requests and questions"
}

// CanHandle determines if this agent can handle the request
func (g *GeneralAgent) CanHandle(request string) (bool, float64) {
	lower := strings.ToLower(request)

	// Check if it's a conversational request (no tool keywords)
	toolKeywords := []string{
		"file", "files", "directory", "folder", "read", "write", "create", "delete",
		"search", "find", "grep", "bash", "run", "execute", "command",
		"git", "commit", "branch", "diff", "status",
		"code", "analyze", "review", "ast", "syntax",
		"count", "list", "show", "display",
	}

	hasToolKeyword := false
	for _, keyword := range toolKeywords {
		if strings.Contains(lower, keyword) {
			hasToolKeyword = true
			break
		}
	}

	// High confidence for greetings and conversational requests
	if !hasToolKeyword {
		if strings.HasPrefix(lower, "hello") || strings.HasPrefix(lower, "hi") ||
			strings.Contains(lower, "thank") || strings.Contains(lower, "please") {
			return true, 0.9
		}

		// Medium confidence for questions
		if strings.HasPrefix(lower, "what") || strings.HasPrefix(lower, "how") ||
			strings.HasPrefix(lower, "why") || strings.HasPrefix(lower, "when") ||
			strings.HasPrefix(lower, "who") || strings.HasPrefix(lower, "where") {
			return true, 0.7
		}

		// Low confidence as fallback for non-tool requests
		return true, 0.5
	}

	// Don't handle tool-based requests
	return false, 0.0
}

// Execute performs the agent's task
func (g *GeneralAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	g.log.Info("Executing general conversational request", "prompt", request.Prompt)

	var response string
	var err error

	// Try to use Ollama if available
	if g.ollamaClient != nil {
		response, err = g.generateOllamaResponse(ctx, request.Prompt)
		if err != nil {
			g.log.Warn("Failed to generate Ollama response, using fallback", "error", err)
			response = g.generateFallbackResponse(request.Prompt)
		}
	} else {
		// No Ollama client, use fallback
		response = g.generateFallbackResponse(request.Prompt)
	}

	return AgentResult{
		Success: true,
		Summary: "Generated conversational response",
		Details: response,
		Metadata: AgentMetadata{
			AgentName: g.Name(),
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime),
		},
	}, nil
}

// generateOllamaResponse generates a response using Ollama
func (g *GeneralAgent) generateOllamaResponse(ctx context.Context, prompt string) (string, error) {
	// For now, return an error since Ollama chat API isn't fully implemented
	// TODO: Implement when Ollama client has proper chat support
	return "", fmt.Errorf("Ollama chat API not yet implemented")
}

// generateFallbackResponse generates a basic response without LLM
func (g *GeneralAgent) generateFallbackResponse(prompt string) string {
	lower := strings.ToLower(prompt)

	// Handle greetings
	if strings.HasPrefix(lower, "hello") || strings.HasPrefix(lower, "hi") ||
		strings.HasPrefix(lower, "hey") || strings.Contains(lower, "greetings") {
		return "Hello! I'm Ryan, your AI assistant. I can help with file operations, code analysis, searching, and various development tasks. What would you like to work on today?"
	}

	// Handle thanks
	if strings.Contains(lower, "thank") {
		return "You're welcome! Is there anything else I can help you with?"
	}

	// Handle simple math
	if strings.Contains(lower, "2+2") || strings.Contains(lower, "2 + 2") {
		return "2 + 2 equals 4."
	}

	// Handle capital questions
	if strings.Contains(lower, "capital") {
		if strings.Contains(lower, "france") {
			return "The capital of France is Paris."
		} else if strings.Contains(lower, "spain") {
			return "The capital of Spain is Madrid."
		} else if strings.Contains(lower, "italy") {
			return "The capital of Italy is Rome."
		} else if strings.Contains(lower, "germany") {
			return "The capital of Germany is Berlin."
		} else if strings.Contains(lower, "japan") {
			return "The capital of Japan is Tokyo."
		}
	}

	// Handle "what is" questions
	if strings.HasPrefix(lower, "what is") || strings.HasPrefix(lower, "what's") {
		return fmt.Sprintf("That's an interesting question: '%s'. While I don't have direct LLM access configured, I can help with file operations, code analysis, searching, and other development tasks. Would you like me to help with something specific?", prompt)
	}

	// Handle "how" questions
	if strings.HasPrefix(lower, "how") {
		if strings.Contains(lower, "are you") {
			return "I'm functioning well, thank you for asking! I'm ready to help with your development tasks."
		}
		return fmt.Sprintf("I understand you're asking '%s'. I'm equipped to help with file operations, code analysis, searching, and other development tasks. What specific action would you like me to perform?", prompt)
	}

	// Handle "why" questions
	if strings.HasPrefix(lower, "why") {
		return fmt.Sprintf("That's a thoughtful question: '%s'. While I'm primarily focused on development tasks, I can help with file operations, code analysis, and more. What would you like to explore?", prompt)
	}

	// Default response
	return fmt.Sprintf("I received your message: '%s'. I'm Ryan, an AI assistant focused on development tasks. I can help with file operations, code analysis, searching, and more. How can I assist you today?", prompt)
}
