package prompt

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tmc/langchaingo/prompts"
)

// ReactPromptTemplate wraps a LangChain prompt template for ReAct agents
type ReactPromptTemplate struct {
	template prompts.PromptTemplate
	mode     string
}

// NewReactPromptTemplate creates a new ReactPromptTemplate with the given template
func NewReactPromptTemplate(template prompts.PromptTemplate, mode string) *ReactPromptTemplate {
	return &ReactPromptTemplate{
		template: template,
		mode:     mode,
	}
}

// LoadReactPrompt loads a ReAct prompt from the prompts directory
func LoadReactPrompt(mode string) (*ReactPromptTemplate, error) {
	// For backward compatibility, map old modes to unified
	if mode == "execute-mode" || mode == "plan-mode" {
		mode = "unified"
	}

	// Construct the path to the prompt file
	promptPath := filepath.Join("prompts", mode, "SYSTEM_PROMPT.md")

	// Read the prompt file
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}

	// Create LangChain prompt template with the required variables
	template := prompts.NewPromptTemplate(
		string(content),
		[]string{"tool_descriptions", "history", "input"},
	)

	return &ReactPromptTemplate{
		template: template,
		mode:     mode,
	}, nil
}

// Format formats the prompt with the given variables
func (r *ReactPromptTemplate) Format(toolDescriptions, history, input string) (string, error) {
	return r.template.Format(map[string]any{
		"tool_descriptions": toolDescriptions,
		"history":           history,
		"input":             input,
	})
}

// GetTemplate returns the underlying LangChain template
func (r *ReactPromptTemplate) GetTemplate() prompts.PromptTemplate {
	return r.template
}

// GetMode returns the mode this prompt is for
func (r *ReactPromptTemplate) GetMode() string {
	return r.mode
}

// DefaultUnifiedPrompt returns a default unified prompt if file loading fails
func DefaultUnifiedPrompt() prompts.PromptTemplate {
	return prompts.NewPromptTemplate(
		`You are a helpful AI assistant that uses the ReAct framework to solve problems. You adapt your approach based on task complexity.

For complex tasks: Plan first, then ask for confirmation before executing.
For simple tasks: Execute directly with clear reasoning.

Available tools:
{{.tool_descriptions}}

Previous conversation:
{{.history}}

Use the ReAct format:
Thought: [Your reasoning]
Action: [Tool name]
Action Input: [Tool input]
Observation: [Tool output]
... (repeat as needed)
Thought: [Final reasoning]
Final Answer: [Your response]

User input: {{.input}}

Begin by analyzing the request complexity.`,
		[]string{"tool_descriptions", "history", "input"},
	)
}

// DefaultExecuteModePrompt returns a default prompt for backward compatibility
func DefaultExecuteModePrompt() prompts.PromptTemplate {
	return DefaultUnifiedPrompt()
}

// DefaultPlanModePrompt returns a default prompt for backward compatibility
func DefaultPlanModePrompt() prompts.PromptTemplate {
	return DefaultUnifiedPrompt()
}
