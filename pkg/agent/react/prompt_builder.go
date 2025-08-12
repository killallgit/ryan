package react

import (
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/prompt"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/tools"
)

// PromptBuilder constructs ReAct prompts
type PromptBuilder struct {
	tools          []tools.Tool
	promptTemplate *prompt.ReactPromptTemplate
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	// Load unified prompt
	promptTemplate, err := prompt.LoadReactPrompt("unified")
	if err != nil {
		logger.Warn("Failed to load prompt from file, using default: %v", err)
		promptTemplate = prompt.NewReactPromptTemplate(
			prompt.DefaultUnifiedPrompt(),
			"unified",
		)
	}

	return &PromptBuilder{
		promptTemplate: promptTemplate,
	}
}

// SetTools sets the available tools
func (pb *PromptBuilder) SetTools(toolList []tools.Tool) {
	pb.tools = toolList
}

// SetCustomPrompt sets a custom prompt template
func (pb *PromptBuilder) SetCustomPrompt(customTemplate string) {
	// Create a custom prompt template
	template := prompts.NewPromptTemplate(
		customTemplate,
		[]string{"tool_descriptions", "history", "input"},
	)
	pb.promptTemplate = prompt.NewReactPromptTemplate(template, "custom")
}

// Build constructs a prompt from the current state
func (pb *PromptBuilder) Build(state *State) string {
	// Build tool descriptions
	var toolDescs []string
	for _, tool := range pb.tools {
		toolDescs = append(toolDescs, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}
	toolDescriptions := strings.Join(toolDescs, "\n")

	// Build conversation history from iterations
	var historyLines []string
	for _, iter := range state.Iterations {
		if iter.Thought != "" {
			historyLines = append(historyLines, fmt.Sprintf("Thought: %s", iter.Thought))
		}
		if iter.Action != "" {
			historyLines = append(historyLines, fmt.Sprintf("Action: %s", iter.Action))
			historyLines = append(historyLines, fmt.Sprintf("Action Input: %s", iter.ActionInput))
		}
		if iter.Observation != "" {
			historyLines = append(historyLines, fmt.Sprintf("Observation: %s", iter.Observation))
		}
	}
	history := strings.Join(historyLines, "\n")
	if len(historyLines) > 0 {
		history += "\n\nContinue from here:"
	}

	// Use the loaded prompt template to format the prompt
	formatted, err := pb.promptTemplate.Format(toolDescriptions, history, state.Input)
	if err != nil {
		logger.Error("Failed to format prompt: %v", err)
		// Return a basic fallback
		return fmt.Sprintf("Tools: %s\n\nHistory: %s\n\nInput: %s\n", toolDescriptions, history, state.Input)
	}

	return formatted
}
