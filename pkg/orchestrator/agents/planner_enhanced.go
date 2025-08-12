package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/tmc/langchaingo/llms"
)

// EnhancedPlannerAgent creates structured plans with NextAction suggestions
type EnhancedPlannerAgent struct {
	llm llms.Model
}

// NewEnhancedPlannerAgent creates a new enhanced planner agent
func NewEnhancedPlannerAgent(llm llms.Model) *EnhancedPlannerAgent {
	return &EnhancedPlannerAgent{llm: llm}
}

// Execute implements the Agent interface with structured planning
func (p *EnhancedPlannerAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	logger.Debug("EnhancedPlannerAgent executing: %s", decision.Instruction)

	// Check if we should create a structured plan
	if shouldCreateStructuredPlan(decision) {
		return p.executeStructuredPlanning(ctx, decision, state)
	}

	// Fall back to simple planning
	return p.executeSimplePlanning(ctx, decision, state)
}

// executeStructuredPlanning creates a structured TaskPlan
func (p *EnhancedPlannerAgent) executeStructuredPlanning(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	prompt := fmt.Sprintf(`You are a planning agent that creates structured, executable plans.

Task: %s

Create a JSON plan with the following structure:
{
  "name": "Brief plan name",
  "description": "Plan description",
  "tasks": [
    {
      "id": "task_1",
      "name": "Task name",
      "description": "What this task does",
      "agent_type": "tool_caller|code_gen|reasoner|searcher",
      "input": "Specific instruction for the agent",
      "expected_output": "What we expect from this task"
    }
  ],
  "dependencies": {
    "task_2": ["task_1"],  // task_2 depends on task_1
    "task_3": ["task_1", "task_2"]  // task_3 depends on both
  }
}

Important guidelines:
1. Break down the task into 3-7 concrete, actionable steps
2. Choose the right agent for each task:
   - tool_caller: For file operations, bash commands, git operations
   - code_gen: For writing or modifying code
   - reasoner: For analysis, debugging, or decision making
   - searcher: For finding code, documentation, or examples
3. Set up dependencies correctly - tasks that need output from others should depend on them
4. Make inputs specific and actionable
5. The first task should be immediately executable

Task to plan: %s`, decision.Instruction, decision.Instruction)

	response, err := p.llm.Call(ctx, prompt)
	if err != nil {
		return p.createErrorResponse(fmt.Errorf("planning failed: %w", err))
	}

	// Try to parse the JSON plan
	plan, err := p.parseJSONPlan(response)
	if err != nil {
		logger.Warn("Failed to parse structured plan, falling back to text: %v", err)
		return p.createTextResponse(response, decision, state)
	}

	// Store the plan in state metadata
	if state.Metadata == nil {
		state.Metadata = make(map[string]interface{})
	}
	state.Metadata["task_plan"] = plan

	// Create response with NextAction for the first task
	var nextAction *orchestrator.RouteDecision
	if len(plan.Tasks) > 0 {
		firstTask := plan.Tasks[0]
		nextAction = &orchestrator.RouteDecision{
			TargetAgent: firstTask.AgentType,
			Instruction: firstTask.Input,
			Parameters: map[string]interface{}{
				"task_id":   firstTask.ID,
				"task_name": firstTask.Name,
			},
		}
		logger.Info("📋 Created structured plan with %d tasks, suggesting first: %s",
			len(plan.Tasks), firstTask.Name)
	}

	// Format the plan for display
	planSummary := p.formatPlanSummary(plan)

	return &orchestrator.AgentResponse{
		AgentType:  orchestrator.AgentPlanner,
		Status:     "success",
		Response:   planSummary,
		NextAction: nextAction,
		Timestamp:  time.Now(),
	}, nil
}

// executeSimplePlanning creates a simple text-based plan
func (p *EnhancedPlannerAgent) executeSimplePlanning(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	prompt := fmt.Sprintf(`You are a planning agent specialized in breaking down complex tasks.

Task: %s

Create a clear, actionable plan with:
1. Task analysis and requirements
2. Step-by-step breakdown (3-7 steps)
3. Dependencies between steps
4. Recommended agent for each step (tool_caller, code_gen, reasoner, or searcher)
5. Success criteria

After the plan, suggest the FIRST concrete action to take.
Format: "FIRST ACTION: [agent_type] - [specific instruction]"`, decision.Instruction)

	response, err := p.llm.Call(ctx, prompt)
	if err != nil {
		return p.createErrorResponse(fmt.Errorf("planning failed: %w", err))
	}

	// Extract first action if present
	nextAction := p.extractFirstAction(response)

	return &orchestrator.AgentResponse{
		AgentType:  orchestrator.AgentPlanner,
		Status:     "success",
		Response:   response,
		NextAction: nextAction,
		Timestamp:  time.Now(),
	}, nil
}

// parseJSONPlan attempts to parse a JSON plan from the response
func (p *EnhancedPlannerAgent) parseJSONPlan(response string) (*orchestrator.TaskPlan, error) {
	// Extract JSON from the response (might be wrapped in markdown or text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	// Parse into a temporary structure
	var planData struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Tasks       []struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			Description    string `json:"description"`
			AgentType      string `json:"agent_type"`
			Input          string `json:"input"`
			ExpectedOutput string `json:"expected_output"`
		} `json:"tasks"`
		Dependencies map[string][]string `json:"dependencies"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to TaskPlan
	plan := orchestrator.NewTaskPlan(planData.Name, planData.Description)

	// Add tasks
	for _, taskData := range planData.Tasks {
		task := orchestrator.SubTask{
			ID:             taskData.ID,
			Name:           taskData.Name,
			Description:    taskData.Description,
			AgentType:      p.parseAgentType(taskData.AgentType),
			Input:          taskData.Input,
			ExpectedOutput: taskData.ExpectedOutput,
			Status:         orchestrator.TaskStatusPending,
			Metadata:       make(map[string]interface{}),
		}
		plan.AddTask(task)
	}

	// Add dependencies
	for taskID, deps := range planData.Dependencies {
		plan.AddDependency(taskID, deps...)
	}

	return plan, nil
}

// parseAgentType converts string to AgentType
func (p *EnhancedPlannerAgent) parseAgentType(agentStr string) orchestrator.AgentType {
	switch strings.ToLower(strings.TrimSpace(agentStr)) {
	case "tool_caller", "tool":
		return orchestrator.AgentToolCaller
	case "code_gen", "code", "codegen":
		return orchestrator.AgentCodeGen
	case "reasoner", "reason":
		return orchestrator.AgentReasoner
	case "searcher", "search":
		return orchestrator.AgentSearcher
	case "planner", "plan":
		return orchestrator.AgentPlanner
	default:
		return orchestrator.AgentReasoner // Default fallback
	}
}

// extractFirstAction extracts a NextAction from text response
func (p *EnhancedPlannerAgent) extractFirstAction(response string) *orchestrator.RouteDecision {
	// Look for "FIRST ACTION:" pattern
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToUpper(line), "FIRST ACTION:") {
			// Parse the action
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}

			actionStr := strings.TrimSpace(parts[1])
			// Try to parse "agent_type - instruction" format
			actionParts := strings.SplitN(actionStr, "-", 2)
			if len(actionParts) >= 2 {
				agentType := p.parseAgentType(strings.TrimSpace(actionParts[0]))
				instruction := strings.TrimSpace(actionParts[1])

				return &orchestrator.RouteDecision{
					TargetAgent: agentType,
					Instruction: instruction,
				}
			}
		}
	}

	// Try to infer from the plan content
	if strings.Contains(response, "search") || strings.Contains(response, "find") {
		if idx := strings.Index(response, "Step 1:"); idx != -1 {
			stepOne := response[idx:min(idx+200, len(response))]
			return &orchestrator.RouteDecision{
				TargetAgent: orchestrator.AgentSearcher,
				Instruction: stepOne,
			}
		}
	}

	return nil
}

// formatPlanSummary creates a readable summary of the plan
func (p *EnhancedPlannerAgent) formatPlanSummary(plan *orchestrator.TaskPlan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📋 **Structured Plan: %s**\n\n", plan.Name))
	sb.WriteString(fmt.Sprintf("*%s*\n\n", plan.Description))
	sb.WriteString(fmt.Sprintf("**Tasks (%d):**\n", len(plan.Tasks)))

	for i, task := range plan.Tasks {
		deps := ""
		if taskDeps, hasDeps := plan.Dependencies[task.ID]; hasDeps && len(taskDeps) > 0 {
			deps = fmt.Sprintf(" (depends on: %s)", strings.Join(taskDeps, ", "))
		}

		sb.WriteString(fmt.Sprintf("%d. **%s** [%s]%s\n",
			i+1, task.Name, task.AgentType, deps))
		sb.WriteString(fmt.Sprintf("   - %s\n", task.Description))
		if task.ExpectedOutput != "" {
			sb.WriteString(fmt.Sprintf("   - Expected: %s\n", task.ExpectedOutput))
		}
	}

	return sb.String()
}

// shouldCreateStructuredPlan determines if we should create a structured plan
func shouldCreateStructuredPlan(decision *orchestrator.RouteDecision) bool {
	// Check for keywords that indicate complex multi-step tasks
	keywords := []string{
		"refactor", "implement", "create", "build", "analyze and",
		"step by step", "workflow", "pipeline", "multiple",
	}

	instruction := strings.ToLower(decision.Instruction)
	for _, keyword := range keywords {
		if strings.Contains(instruction, keyword) {
			return true
		}
	}

	// Check if explicitly requested
	if params, ok := decision.Parameters["structured_plan"].(bool); ok && params {
		return true
	}

	return false
}

// createErrorResponse creates an error response
func (p *EnhancedPlannerAgent) createErrorResponse(err error) (*orchestrator.AgentResponse, error) {
	errorStr := err.Error()
	return &orchestrator.AgentResponse{
		AgentType: orchestrator.AgentPlanner,
		Status:    "failed",
		Error:     &errorStr,
		Timestamp: time.Now(),
	}, nil
}

// createTextResponse creates a simple text response
func (p *EnhancedPlannerAgent) createTextResponse(response string, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	nextAction := p.extractFirstAction(response)

	return &orchestrator.AgentResponse{
		AgentType:  orchestrator.AgentPlanner,
		Status:     "success",
		Response:   response,
		NextAction: nextAction,
		Timestamp:  time.Now(),
	}, nil
}

// GetCapabilities implements the Agent interface
func (p *EnhancedPlannerAgent) GetCapabilities() []string {
	return []string{"planning", "decomposition", "strategy", "organization", "structured_planning"}
}

// GetType implements the Agent interface
func (p *EnhancedPlannerAgent) GetType() orchestrator.AgentType {
	return orchestrator.AgentPlanner
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
