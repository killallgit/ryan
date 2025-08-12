package react

import (
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/tools"
)

// PromptBuilder constructs ReAct prompts
type PromptBuilder struct {
	tools    []tools.Tool
	template string
	mode     Mode
}

// Mode represents the agent's operating mode
type Mode string

const (
	ExecuteMode Mode = "execute"
	PlanMode    Mode = "plan"
)

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		mode:     ExecuteMode,
		template: defaultReActTemplate,
	}
}

// SetTools sets the available tools
func (pb *PromptBuilder) SetTools(toolList []tools.Tool) {
	pb.tools = toolList
}

// SetMode sets the operating mode
func (pb *PromptBuilder) SetMode(mode Mode) {
	pb.mode = mode
}

// Build constructs a prompt from the current state
func (pb *PromptBuilder) Build(state *State) string {
	var prompt strings.Builder

	// Add system instructions based on mode
	if pb.mode == ExecuteMode {
		prompt.WriteString(executeInstructions)
	} else {
		prompt.WriteString(planInstructions)
	}

	// Add tool descriptions
	prompt.WriteString("\n\nAvailable tools:\n")
	for _, tool := range pb.tools {
		prompt.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	// Add ReAct format instructions
	prompt.WriteString("\n" + reactFormatInstructions)

	// Add the question
	prompt.WriteString(fmt.Sprintf("\n\nQuestion: %s\n", state.Input))

	// Add conversation history (previous iterations)
	if len(state.Iterations) > 0 {
		prompt.WriteString("\nPrevious steps:\n")
		for _, iter := range state.Iterations {
			if iter.Thought != "" {
				prompt.WriteString(fmt.Sprintf("Thought: %s\n", iter.Thought))
			}
			if iter.Action != "" {
				prompt.WriteString(fmt.Sprintf("Action: %s\n", iter.Action))
				prompt.WriteString(fmt.Sprintf("Action Input: %s\n", iter.ActionInput))
			}
			if iter.Observation != "" {
				prompt.WriteString(fmt.Sprintf("Observation: %s\n", iter.Observation))
			}
		}
		prompt.WriteString("\nContinue from here:\n")
	}

	return prompt.String()
}

const defaultReActTemplate = `You are a helpful AI assistant with access to tools.`

const executeInstructions = `You are a helpful AI assistant that can use tools to help answer questions and complete tasks.
Always think step-by-step about what you need to do before taking actions.`

const planInstructions = `You are a helpful AI assistant that creates detailed plans for completing tasks.
Think step-by-step about what needs to be done, but do not execute any actions.
Instead, create a comprehensive plan that someone else could follow.`

const reactFormatInstructions = `Use this exact format for your response:

Thought: [Your reasoning about what to do next]
Action: [The tool name to use, or none if you have the final answer]
Action Input: [The input for the tool as JSON or simple text]
Observation: [This will be filled in by the system after tool execution]
... (repeat Thought/Action/Action Input/Observation as needed)
Thought: [Final reasoning]
Final Answer: [Your final response to the user]

Important:
- Always start with a Thought
- If you need to use a tool, specify the Action and Action Input
- Wait for the Observation before continuing
- When you have enough information, provide a Final Answer
- Each response should contain ONLY ONE iteration (one Thought/Action pair OR Final Answer)`
