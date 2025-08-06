package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// LLMIntentAnalyzer uses an LLM to analyze user intent
type LLMIntentAnalyzer struct {
	llm llms.Model
	log *logger.Logger
}

// NewLLMIntentAnalyzer creates a new LLM-based intent analyzer
func NewLLMIntentAnalyzer(baseURL, model string) (*LLMIntentAnalyzer, error) {
	log := logger.WithComponent("llm_intent_analyzer")

	// Create Ollama LLM for intent analysis
	llm, err := ollama.New(
		ollama.WithServerURL(baseURL),
		ollama.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama LLM: %w", err)
	}

	return &LLMIntentAnalyzer{
		llm: llm,
		log: log,
	}, nil
}

// AnalyzeIntent uses the LLM to determine the user's intent
func (lia *LLMIntentAnalyzer) AnalyzeIntent(ctx context.Context, userPrompt string) (*Intent, error) {
	// Use a short timeout for intent analysis to avoid blocking
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Create a prompt for the LLM to analyze intent
	intentPrompt := fmt.Sprintf(`Analyze the following user request and determine the primary intent.

User Request: "%s"

Classify the intent as one of the following:
- FILE_OPERATION: User wants to read, write, list, count, or manipulate files
- CODE_ANALYSIS: User wants to analyze code structure, AST, symbols, or patterns
- CODE_REVIEW: User wants code review or quality assessment
- SEARCH: User wants to search for something in files or code
- REFACTOR: User wants to refactor or improve code
- TEST: User wants to run or create tests
- CONVERSATIONAL: User is asking a general question or having a conversation
- TOOL_EXECUTION: User wants to run a command or execute a tool

Also identify if this request needs any tools to be executed.

Respond in the following format:
INTENT: <intent_type>
NEEDS_TOOLS: <yes/no>
CONFIDENCE: <high/medium/low>
REASONING: <brief explanation>`, userPrompt)

	// Call the LLM
	response, err := llms.GenerateFromSinglePrompt(ctx, lia.llm, intentPrompt)
	if err != nil {
		lia.log.Error("Failed to analyze intent with LLM", "error", err)
		// Fall back to pattern matching
		return lia.fallbackAnalyze(userPrompt), nil
	}

	// Parse the LLM response
	intent := lia.parseIntentResponse(response, userPrompt)
	lia.log.Info("LLM intent analysis complete",
		"intent", intent.Primary,
		"confidence", intent.Confidence,
		"needs_tools", intent.NeedsTools)

	return intent, nil
}

// parseIntentResponse parses the LLM's intent analysis response
func (lia *LLMIntentAnalyzer) parseIntentResponse(response, originalPrompt string) *Intent {
	intent := &Intent{
		RawPrompt: originalPrompt,
		Entities:  make(map[string]string),
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "INTENT:") {
			intentType := strings.TrimSpace(strings.TrimPrefix(line, "INTENT:"))
			intent.Primary = lia.mapToIntentType(intentType)
		} else if strings.HasPrefix(line, "NEEDS_TOOLS:") {
			needsTools := strings.TrimSpace(strings.TrimPrefix(line, "NEEDS_TOOLS:"))
			intent.NeedsTools = strings.ToLower(needsTools) == "yes"
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			confidence := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			intent.Confidence = lia.mapConfidence(confidence)
		} else if strings.HasPrefix(line, "REASONING:") {
			intent.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "REASONING:"))
		}
	}

	// Default to generic if no intent found
	if intent.Primary == "" {
		intent.Primary = IntentGeneric
		intent.Confidence = 0.5
	}

	return intent
}

// mapToIntentType maps LLM response to IntentType
func (lia *LLMIntentAnalyzer) mapToIntentType(intentStr string) IntentType {
	switch strings.ToUpper(strings.TrimSpace(intentStr)) {
	case "FILE_OPERATION":
		return IntentFileOperation
	case "CODE_ANALYSIS":
		return IntentAnalysis
	case "CODE_REVIEW":
		return IntentCodeReview
	case "SEARCH":
		return IntentSearch
	case "REFACTOR":
		return IntentRefactor
	case "TEST":
		return IntentTest
	case "CONVERSATIONAL":
		return IntentGeneric
	case "TOOL_EXECUTION":
		return IntentGeneric // Will be handled with tools
	default:
		return IntentGeneric
	}
}

// mapConfidence maps confidence string to float
func (lia *LLMIntentAnalyzer) mapConfidence(confidence string) float64 {
	switch strings.ToLower(confidence) {
	case "high":
		return 0.9
	case "medium":
		return 0.7
	case "low":
		return 0.5
	default:
		return 0.5
	}
}

// fallbackAnalyze provides fallback intent analysis using patterns
func (lia *LLMIntentAnalyzer) fallbackAnalyze(prompt string) *Intent {
	lowerPrompt := strings.ToLower(prompt)

	intent := &Intent{
		RawPrompt:  prompt,
		Entities:   make(map[string]string),
		Confidence: 0.5,
	}

	// Simple pattern matching as fallback
	if strings.Contains(lowerPrompt, "file") || strings.Contains(lowerPrompt, "count") ||
		strings.Contains(lowerPrompt, "list") || strings.Contains(lowerPrompt, "read") {
		intent.Primary = IntentFileOperation
		intent.NeedsTools = true
		intent.Confidence = 0.7
	} else if strings.Contains(lowerPrompt, "analyze") || strings.Contains(lowerPrompt, "code") {
		intent.Primary = IntentAnalysis
		intent.NeedsTools = true
		intent.Confidence = 0.7
	} else if strings.Contains(lowerPrompt, "search") || strings.Contains(lowerPrompt, "find") {
		intent.Primary = IntentSearch
		intent.NeedsTools = true
		intent.Confidence = 0.7
	} else {
		intent.Primary = IntentGeneric
		intent.NeedsTools = false
		intent.Confidence = 0.5
	}

	return intent
}
