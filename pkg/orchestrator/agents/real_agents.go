package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/orchestrator"
	ryantools "github.com/killallgit/ryan/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// ToolInterface defines the interface for tools that can be called
type ToolInterface interface {
	Call(ctx context.Context, input string) (string, error)
	Name() string
	Description() string
}

// ToolCallerAgent handles tool execution and function calls
type ToolCallerAgent struct {
	llm             llms.Model
	availableTools  map[string]ToolInterface
	skipPermissions bool
}

// NewToolCallerAgent creates a new tool caller agent
func NewToolCallerAgent(llm llms.Model, skipPermissions bool) *ToolCallerAgent {
	agent := &ToolCallerAgent{
		llm:             llm,
		availableTools:  make(map[string]ToolInterface),
		skipPermissions: skipPermissions,
	}

	// Initialize available tools
	agent.initializeTools()
	return agent
}

// initializeTools sets up the available tools
func (t *ToolCallerAgent) initializeTools() {
	// Add all available tools
	t.availableTools["bash"] = ryantools.NewBashToolWithBypass(t.skipPermissions)
	t.availableTools["file_read"] = ryantools.NewFileReadToolWithBypass(t.skipPermissions)
	t.availableTools["file_write"] = ryantools.NewFileWriteToolWithBypass(t.skipPermissions)
	t.availableTools["git"] = ryantools.NewGitToolWithBypass(t.skipPermissions)
	t.availableTools["search"] = ryantools.NewRipgrepToolWithBypass(t.skipPermissions)
	t.availableTools["web"] = ryantools.NewWebFetchToolWithBypass(t.skipPermissions)

	logger.Info("ToolCallerAgent initialized with %d tools", len(t.availableTools))
}

// Execute implements the Agent interface
func (t *ToolCallerAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	logger.Debug("ToolCallerAgent executing: %s", decision.Instruction)

	// Create prompt for tool usage analysis
	toolsJson, _ := json.Marshal(t.getToolDescriptions())
	prompt := fmt.Sprintf(`You are a tool caller agent. Analyze the following instruction and determine which tools to use and how to use them.

Available tools: %s

Instruction: %s

Please provide a structured response with:
1. Which tools to call and with what arguments
2. The execution plan
3. Expected results

Respond in the following JSON format:
{
  "plan": "explanation of what you'll do",
  "tool_calls": [
    {
      "tool": "tool_name",
      "description": "what this call will do",
      "arguments": {"arg1": "value1"}
    }
  ]
}`, toolsJson, decision.Instruction)

	// Get LLM response for planning
	planResponse, err := t.llm.Call(ctx, prompt)
	if err != nil {
		return &orchestrator.AgentResponse{
			AgentType: orchestrator.AgentToolCaller,
			Status:    "failed",
			Error:     &[]string{fmt.Sprintf("Failed to plan tool usage: %v", err)}[0],
			Timestamp: time.Now(),
		}, nil
	}

	// Parse the plan
	var plan struct {
		Plan      string `json:"plan"`
		ToolCalls []struct {
			Tool        string                 `json:"tool"`
			Description string                 `json:"description"`
			Arguments   map[string]interface{} `json:"arguments"`
		} `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(planResponse), &plan); err != nil {
		// If JSON parsing fails, try to extract tool calls from the text response
		logger.Warn("Failed to parse JSON plan, attempting text analysis: %v", err)
		return t.executeWithTextAnalysis(ctx, decision, planResponse)
	}

	// Execute the planned tool calls
	var toolCalls []orchestrator.ToolCall
	var results []string
	var hasError bool
	var errorMsg string

	results = append(results, fmt.Sprintf("Plan: %s\n", plan.Plan))

	for _, toolCall := range plan.ToolCalls {
		logger.Debug("Executing tool: %s with args: %v", toolCall.Tool, toolCall.Arguments)

		tool, exists := t.availableTools[toolCall.Tool]
		if !exists {
			errorMsg = fmt.Sprintf("Tool %s not available", toolCall.Tool)
			hasError = true
			break
		}

		// Execute the tool
		result, err := t.executeTool(ctx, tool, toolCall.Arguments)
		if err != nil {
			errorMsg = fmt.Sprintf("Tool %s failed: %v", toolCall.Tool, err)
			hasError = true
			break
		}

		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      toolCall.Tool,
			Arguments: toolCall.Arguments,
			Result:    result,
		})

		results = append(results, fmt.Sprintf("%s: %s", toolCall.Description, result))
	}

	// Build response
	response := &orchestrator.AgentResponse{
		AgentType:   orchestrator.AgentToolCaller,
		ToolsCalled: toolCalls,
		Timestamp:   time.Now(),
	}

	if hasError {
		response.Status = "failed"
		response.Error = &errorMsg
		response.Response = strings.Join(results, "\n")
	} else {
		response.Status = "success"
		response.Response = strings.Join(results, "\n")
	}

	return response, nil
}

// executeWithTextAnalysis handles cases where JSON parsing failed
func (t *ToolCallerAgent) executeWithTextAnalysis(ctx context.Context, decision *orchestrator.RouteDecision, llmResponse string) (*orchestrator.AgentResponse, error) {
	// Simple heuristic-based tool selection based on instruction content
	instruction := strings.ToLower(decision.Instruction)
	var selectedTool ToolInterface
	var toolName string
	var args map[string]interface{}

	if strings.Contains(instruction, "list") && strings.Contains(instruction, "file") {
		selectedTool = t.availableTools["bash"]
		toolName = "bash"
		args = map[string]interface{}{"command": "ls -la"}
	} else if strings.Contains(instruction, "read") && strings.Contains(instruction, "file") {
		selectedTool = t.availableTools["file_read"]
		toolName = "file_read"
		// Try to extract filename from instruction
		args = map[string]interface{}{"path": "."} // Default to current directory
	} else if strings.Contains(instruction, "search") || strings.Contains(instruction, "find") {
		selectedTool = t.availableTools["search"]
		toolName = "search"
		args = map[string]interface{}{"pattern": ".*"} // Default pattern
	} else {
		// Default to bash for general commands
		selectedTool = t.availableTools["bash"]
		toolName = "bash"
		args = map[string]interface{}{"command": "echo 'Tool analysis needed'"}
	}

	// Execute the selected tool
	result, err := t.executeTool(ctx, selectedTool, args)
	if err != nil {
		errorStr := fmt.Sprintf("Tool execution failed: %v", err)
		return &orchestrator.AgentResponse{
			AgentType: orchestrator.AgentToolCaller,
			Status:    "failed",
			Error:     &errorStr,
			Response:  llmResponse,
			Timestamp: time.Now(),
		}, nil
	}

	return &orchestrator.AgentResponse{
		AgentType: orchestrator.AgentToolCaller,
		Status:    "success",
		Response:  fmt.Sprintf("Executed %s: %s", toolName, result),
		ToolsCalled: []orchestrator.ToolCall{
			{
				Name:      toolName,
				Arguments: args,
				Result:    result,
			},
		},
		Timestamp: time.Now(),
	}, nil
}

// executeTool executes a specific tool with given arguments
func (t *ToolCallerAgent) executeTool(ctx context.Context, tool ToolInterface, args map[string]interface{}) (string, error) {
	// Convert args to the format expected by the tool
	// For now, we'll just pass the arguments as a string representation
	// In a real implementation, we'd parse the arguments properly for each tool

	var input string
	if cmd, ok := args["command"].(string); ok {
		input = cmd
	} else if path, ok := args["path"].(string); ok {
		input = path
	} else if pattern, ok := args["pattern"].(string); ok {
		input = pattern
	} else if query, ok := args["query"].(string); ok {
		input = query
	} else {
		input = fmt.Sprintf("%v", args)
	}

	return tool.Call(ctx, input)
}

// getToolDescriptions returns descriptions of available tools
func (t *ToolCallerAgent) getToolDescriptions() map[string]string {
	return map[string]string{
		"bash":       "Execute shell commands",
		"file_read":  "Read file contents",
		"file_write": "Write content to files",
		"git":        "Git version control operations",
		"search":     "Search through files and code",
		"web":        "Fetch content from web URLs",
	}
}

// GetCapabilities implements the Agent interface
func (t *ToolCallerAgent) GetCapabilities() []string {
	capabilities := make([]string, 0, len(t.availableTools))
	for name := range t.availableTools {
		capabilities = append(capabilities, name)
	}
	return capabilities
}

// GetType implements the Agent interface
func (t *ToolCallerAgent) GetType() orchestrator.AgentType {
	return orchestrator.AgentToolCaller
}

// ReasonerAgent handles complex reasoning and analysis
type ReasonerAgent struct {
	llm llms.Model
}

// NewReasonerAgent creates a new reasoner agent
func NewReasonerAgent(llm llms.Model) *ReasonerAgent {
	return &ReasonerAgent{llm: llm}
}

// Execute implements the Agent interface
func (r *ReasonerAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	logger.Debug("ReasonerAgent executing: %s", decision.Instruction)

	// Create a reasoning-focused prompt
	prompt := fmt.Sprintf(`You are a reasoning agent specialized in analysis, problem-solving, and logical thinking.

Task: %s

Please provide a thorough analysis with:
1. Problem understanding
2. Key considerations
3. Step-by-step reasoning
4. Conclusions and recommendations

Be detailed and explain your reasoning process.`, decision.Instruction)

	response, err := r.llm.Call(ctx, prompt)
	if err != nil {
		errorStr := fmt.Sprintf("Reasoning failed: %v", err)
		return &orchestrator.AgentResponse{
			AgentType: orchestrator.AgentReasoner,
			Status:    "failed",
			Error:     &errorStr,
			Timestamp: time.Now(),
		}, nil
	}

	return &orchestrator.AgentResponse{
		AgentType: orchestrator.AgentReasoner,
		Status:    "success",
		Response:  response,
		Timestamp: time.Now(),
	}, nil
}

// GetCapabilities implements the Agent interface
func (r *ReasonerAgent) GetCapabilities() []string {
	return []string{"reasoning", "analysis", "problem-solving", "logic"}
}

// GetType implements the Agent interface
func (r *ReasonerAgent) GetType() orchestrator.AgentType {
	return orchestrator.AgentReasoner
}

// CodeGenAgent handles code generation tasks
type CodeGenAgent struct {
	llm llms.Model
}

// NewCodeGenAgent creates a new code generation agent
func NewCodeGenAgent(llm llms.Model) *CodeGenAgent {
	return &CodeGenAgent{llm: llm}
}

// Execute implements the Agent interface
func (c *CodeGenAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	logger.Debug("CodeGenAgent executing: %s", decision.Instruction)

	prompt := fmt.Sprintf(`You are a code generation agent specialized in writing high-quality, well-documented code.

Task: %s

Please provide:
1. Clean, well-structured code
2. Appropriate comments and documentation
3. Follow best practices for the language
4. Include error handling where appropriate

Format your response with proper code blocks and explanations.`, decision.Instruction)

	response, err := c.llm.Call(ctx, prompt)
	if err != nil {
		errorStr := fmt.Sprintf("Code generation failed: %v", err)
		return &orchestrator.AgentResponse{
			AgentType: orchestrator.AgentCodeGen,
			Status:    "failed",
			Error:     &errorStr,
			Timestamp: time.Now(),
		}, nil
	}

	return &orchestrator.AgentResponse{
		AgentType: orchestrator.AgentCodeGen,
		Status:    "success",
		Response:  response,
		Timestamp: time.Now(),
	}, nil
}

// GetCapabilities implements the Agent interface
func (c *CodeGenAgent) GetCapabilities() []string {
	return []string{"coding", "programming", "implementation", "refactoring"}
}

// GetType implements the Agent interface
func (c *CodeGenAgent) GetType() orchestrator.AgentType {
	return orchestrator.AgentCodeGen
}

// SearcherAgent handles search and code analysis tasks
type SearcherAgent struct {
	llm        llms.Model
	searchTool ToolInterface
}

// NewSearcherAgent creates a new searcher agent
func NewSearcherAgent(llm llms.Model, skipPermissions bool) *SearcherAgent {
	return &SearcherAgent{
		llm:        llm,
		searchTool: ryantools.NewRipgrepToolWithBypass(skipPermissions),
	}
}

// Execute implements the Agent interface
func (s *SearcherAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	logger.Debug("SearcherAgent executing: %s", decision.Instruction)

	// First, analyze what to search for
	prompt := fmt.Sprintf(`You are a search agent. Analyze the following request and determine the best search strategy.

Request: %s

Determine:
1. What patterns or terms to search for
2. What file types to focus on
3. What search strategy would be most effective

Provide your search plan and then I'll execute it.`, decision.Instruction)

	planResponse, err := s.llm.Call(ctx, prompt)
	if err != nil {
		errorStr := fmt.Sprintf("Search planning failed: %v", err)
		return &orchestrator.AgentResponse{
			AgentType: orchestrator.AgentSearcher,
			Status:    "failed",
			Error:     &errorStr,
			Timestamp: time.Now(),
		}, nil
	}

	// Execute search (simplified for now - in reality would parse the plan)
	searchResult, err := s.searchTool.Call(ctx, decision.Instruction)
	if err != nil {
		logger.Warn("Search tool failed: %v", err)
		searchResult = "Search could not be completed"
	}

	response := fmt.Sprintf("Search Plan:\n%s\n\nSearch Results:\n%s", planResponse, searchResult)

	return &orchestrator.AgentResponse{
		AgentType: orchestrator.AgentSearcher,
		Status:    "success",
		Response:  response,
		ToolsCalled: []orchestrator.ToolCall{
			{
				Name:      "search",
				Arguments: map[string]interface{}{"query": decision.Instruction},
				Result:    searchResult,
			},
		},
		Timestamp: time.Now(),
	}, nil
}

// GetCapabilities implements the Agent interface
func (s *SearcherAgent) GetCapabilities() []string {
	return []string{"search", "find", "locate", "analysis"}
}

// GetType implements the Agent interface
func (s *SearcherAgent) GetType() orchestrator.AgentType {
	return orchestrator.AgentSearcher
}

// PlannerAgent handles task planning and decomposition
type PlannerAgent struct {
	llm llms.Model
}

// NewPlannerAgent creates a new planner agent
func NewPlannerAgent(llm llms.Model) *PlannerAgent {
	return &PlannerAgent{llm: llm}
}

// Execute implements the Agent interface
func (p *PlannerAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	logger.Debug("PlannerAgent executing: %s", decision.Instruction)

	prompt := fmt.Sprintf(`You are a planning agent specialized in breaking down complex tasks into manageable steps.

Task: %s

Please create a detailed plan with:
1. Task analysis and requirements
2. Step-by-step breakdown
3. Dependencies between steps
4. Recommended tools or approaches for each step
5. Success criteria

Also suggest which specialized agents might be needed for each step (tool_caller, code_gen, reasoner, searcher).`, decision.Instruction)

	response, err := p.llm.Call(ctx, prompt)
	if err != nil {
		errorStr := fmt.Sprintf("Planning failed: %v", err)
		return &orchestrator.AgentResponse{
			AgentType: orchestrator.AgentPlanner,
			Status:    "failed",
			Error:     &errorStr,
			Timestamp: time.Now(),
		}, nil
	}

	// For now, don't suggest next actions - let the orchestrator handle the flow
	// In future, could parse the plan and suggest the first step
	return &orchestrator.AgentResponse{
		AgentType: orchestrator.AgentPlanner,
		Status:    "success",
		Response:  response,
		Timestamp: time.Now(),
	}, nil
}

// GetCapabilities implements the Agent interface
func (p *PlannerAgent) GetCapabilities() []string {
	return []string{"planning", "decomposition", "strategy", "organization"}
}

// GetType implements the Agent interface
func (p *PlannerAgent) GetType() orchestrator.AgentType {
	return orchestrator.AgentPlanner
}
