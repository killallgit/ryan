package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/llms"
)

// GeneralAgent handles general conversational requests
type GeneralAgent struct {
	llm   llms.Model
	model string
	log   *logger.Logger
}

// NewGeneralAgent creates a new general purpose conversational agent
func NewGeneralAgent(llm llms.Model, model string) *GeneralAgent {
	return &GeneralAgent{
		llm:   llm,
		model: model,
		log:   logger.WithComponent("general_agent"),
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
// NOTE: With LLM-based intent detection, this method should not do keyword matching.
// The orchestrator's LLM will determine if this agent is appropriate.
func (g *GeneralAgent) CanHandle(request string) (bool, float64) {
	// Always return true with high confidence when asked
	// The orchestrator's LLM has already determined this is the right agent
	return true, 1.0
}

// Execute performs the agent's task
func (g *GeneralAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	g.log.Info("Executing general conversational request", "prompt", request.Prompt)

	var response string
	var err error

	// Try to use LLM if available
	if g.llm != nil {
		response, err = g.generateLLMResponse(ctx, request.Prompt)
		if err != nil {
			g.log.Warn("Failed to generate LLM response, using fallback", "error", err)
			response = g.generateFallbackResponse(request.Prompt)
		}
	} else {
		// No LLM configured, use fallback
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

// generateLLMResponse generates a response using the LLM
func (g *GeneralAgent) generateLLMResponse(ctx context.Context, prompt string) (string, error) {
	if g.llm == nil {
		return "", fmt.Errorf("LLM not configured")
	}

	// Create message for LLM
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	// Generate response using LangChain LLM
	response, err := g.llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate LLM response: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	return response.Choices[0].Content, nil
}

// generateFallbackResponse generates a basic response without LLM
// NOTE: This should only be used when Ollama is completely unavailable.
// No keyword matching should be done here.
func (g *GeneralAgent) generateFallbackResponse(prompt string) string {
	// Simple fallback when LLM is unavailable
	// No keyword matching - the orchestrator's LLM already determined this is a general query
	return fmt.Sprintf("I understand your request: '%s'. However, I'm currently unable to connect to the language model to provide a proper response. Please ensure Ollama is running and try again.", prompt)
}
