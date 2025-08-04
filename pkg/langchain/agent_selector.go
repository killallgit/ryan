package langchain

import (
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
)

// AgentType represents the type of agent to use
type AgentType int

const (
	// AgentTypeConversational uses ReAct-style conversational agent
	AgentTypeConversational AgentType = iota
	// AgentTypeOllamaFunctions uses native Ollama function calling
	AgentTypeOllamaFunctions
	// AgentTypeOpenAIFunctions uses OpenAI-style function calling
	AgentTypeOpenAIFunctions
	// AgentTypeDirect bypasses agents and uses direct LLM
	AgentTypeDirect
)

// AgentSelector determines which agent type to use based on input and model
type AgentSelector struct {
	toolRegistry *tools.Registry
	modelInfo    models.ModelInfo
	log          *logger.Logger
}

// NewAgentSelector creates a new agent selector
func NewAgentSelector(toolRegistry *tools.Registry, modelName string) *AgentSelector {
	return &AgentSelector{
		toolRegistry: toolRegistry,
		modelInfo:    models.GetModelInfo(modelName),
		log:          logger.WithComponent("agent_selector"),
	}
}

// SelectAgent determines the best agent type for the given input
func (as *AgentSelector) SelectAgent(input string) (AgentType, bool) {
	needsTools := as.analyzeToolNeed(input)

	as.log.Debug("Agent selection analysis",
		"input_length", len(input),
		"needs_tools", needsTools,
		"model", as.modelInfo.Name,
		"tool_compatibility", as.modelInfo.ToolCompatibility.String())

	// If no tools needed, use direct LLM
	if !needsTools {
		return AgentTypeDirect, false
	}

	// If model doesn't support tools, use direct LLM with tool context
	if as.modelInfo.ToolCompatibility == models.ToolCompatibilityNone {
		as.log.Warn("Model doesn't support tool calling, using direct mode",
			"model", as.modelInfo.Name)
		return AgentTypeDirect, false
	}

	// For models with excellent tool support, prefer native function calling
	if as.modelInfo.ToolCompatibility == models.ToolCompatibilityExcellent {
		// Check if it's an Ollama-compatible model
		if as.isOllamaCompatible() {
			as.log.Debug("Selecting Ollama native function calling",
				"model", as.modelInfo.Name)
			return AgentTypeOllamaFunctions, true
		}
		// For OpenAI models, use OpenAI functions agent
		if strings.Contains(strings.ToLower(as.modelInfo.Name), "gpt") {
			return AgentTypeOpenAIFunctions, true
		}
	}

	// Default to conversational agent for other cases
	as.log.Debug("Selecting conversational agent",
		"model", as.modelInfo.Name,
		"reason", "default fallback")
	return AgentTypeConversational, true
}

// analyzeToolNeed determines if the input likely requires tool usage
func (as *AgentSelector) analyzeToolNeed(input string) bool {
	lowerInput := strings.ToLower(input)

	// Tool-indicating keywords and patterns
	toolKeywords := []string{
		// File system operations
		"how many files", "list files", "list the files", "count files", "show files",
		"create file", "write file", "read file", "delete file",
		"what's in", "show me the contents", "open file",

		// Command execution
		"run command", "execute", "terminal", "bash", "shell",
		"docker", "git", "npm", "go run", "python",

		// System information
		"disk usage", "memory usage", "cpu usage", "system info",
		"process", "running", "status",

		// Web operations
		"fetch", "download", "web page", "url", "website",
		"search for", "look up", "find information about",

		// Code operations
		"grep", "search code", "find in files", "locate",
	}

	// Check for tool keywords
	for _, keyword := range toolKeywords {
		if strings.Contains(lowerInput, keyword) {
			as.log.Debug("Tool keyword detected",
				"keyword", keyword,
				"input_snippet", truncateString(input, 50))
			return true
		}
	}

	// Check for question patterns that typically need tools
	questionPatterns := []string{
		"how many", "what is the", "show me", "can you check",
		"what's running", "is there", "do we have",
	}

	for _, pattern := range questionPatterns {
		if strings.HasPrefix(lowerInput, pattern) {
			as.log.Debug("Tool-requiring question pattern detected",
				"pattern", pattern)
			return true
		}
	}

	// Check if tools are explicitly mentioned
	if as.toolRegistry != nil {
		for _, tool := range as.toolRegistry.GetTools() {
			toolName := strings.ToLower(tool.Name())
			if strings.Contains(lowerInput, toolName) {
				as.log.Debug("Explicit tool reference detected",
					"tool", toolName)
				return true
			}
		}
	}

	return false
}

// isOllamaCompatible checks if the model is compatible with Ollama's function calling
func (as *AgentSelector) isOllamaCompatible() bool {
	// Models known to work well with Ollama's function calling
	ollamaModels := []string{
		"llama", "qwen", "mistral", "deepseek", "command-r",
		"granite", "gemma2", "phi3",
	}

	modelLower := strings.ToLower(as.modelInfo.Name)
	for _, model := range ollamaModels {
		if strings.Contains(modelLower, model) {
			return true
		}
	}

	return false
}

// GetRecommendedAgent returns a recommendation string for the selected agent
func (as *AgentSelector) GetRecommendedAgent(agentType AgentType, needsTools bool) string {
	switch agentType {
	case AgentTypeOllamaFunctions:
		return "Native Ollama function calling (recommended for tool usage)"
	case AgentTypeOpenAIFunctions:
		return "OpenAI Functions agent (native function calling)"
	case AgentTypeConversational:
		return "Conversational ReAct agent (may require output processing)"
	case AgentTypeDirect:
		if needsTools {
			return "Direct LLM (tools available but not executable)"
		}
		return "Direct LLM (no tools needed)"
	default:
		return "Unknown agent type"
	}
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
