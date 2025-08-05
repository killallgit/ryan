package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// DispatcherAgent is the main entry point that analyzes prompts and dispatches to other agents
type DispatcherAgent struct {
	orchestrator *Orchestrator
	planner      *Planner
	log          *logger.Logger
}

// NewDispatcherAgent creates a new dispatcher agent
func NewDispatcherAgent(orchestrator *Orchestrator) *DispatcherAgent {
	return &DispatcherAgent{
		orchestrator: orchestrator,
		planner:      orchestrator.planner,
		log:          logger.WithComponent("dispatcher_agent"),
	}
}

// Name returns the agent name
func (d *DispatcherAgent) Name() string {
	return "dispatcher"
}

// Description returns the agent description
func (d *DispatcherAgent) Description() string {
	return "Analyzes user prompts and creates execution plans for other agents"
}

// CanHandle determines if this agent can handle the request
func (d *DispatcherAgent) CanHandle(request string) (bool, float64) {
	// Dispatcher can handle any request as the entry point
	return true, 1.0
}

// Execute analyzes the request and creates an execution plan
func (d *DispatcherAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	d.log.Info("Analyzing request", "prompt_preview", truncateString(request.Prompt, 100))

	// Create execution context if not provided
	execContext, ok := request.Context["execution_context"].(*ExecutionContext)
	if !ok {
		execContext = &ExecutionContext{
			SessionID:   generateID(),
			RequestID:   generateID(),
			UserPrompt:  request.Prompt,
			SharedData:  make(map[string]interface{}),
			FileContext: []FileInfo{},
			Progress:    make(chan ProgressUpdate, 100),
			Options:     request.Options,
		}
	}

	// Create execution plan
	plan, err := d.planner.CreateExecutionPlan(ctx, request.Prompt, execContext)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to create execution plan",
			Details: err.Error(),
			Metadata: AgentMetadata{
				AgentName: d.Name(),
				StartTime: startTime,
				EndTime:   time.Now(),
				Duration:  time.Since(startTime),
			},
		}, err
	}

	// Store plan in context
	execContext.SharedData["execution_plan"] = plan

	// Build summary of the plan
	summary := d.buildPlanSummary(plan)

	// Build details
	details := d.buildPlanDetails(plan)

	return AgentResult{
		Success: true,
		Summary: summary,
		Details: details,
		Artifacts: map[string]interface{}{
			"plan":            plan,
			"execution_context": execContext,
		},
		Metadata: AgentMetadata{
			AgentName: d.Name(),
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime),
		},
	}, nil
}

// buildPlanSummary creates a summary of the execution plan
func (d *DispatcherAgent) buildPlanSummary(plan *ExecutionPlan) string {
	agents := make(map[string]int)
	for _, task := range plan.Tasks {
		agents[task.Agent]++
	}

	parts := []string{
		fmt.Sprintf("Created plan with %d tasks", len(plan.Tasks)),
		fmt.Sprintf("across %d stages", len(plan.Stages)),
	}

	// List agents involved
	agentList := []string{}
	for agent, count := range agents {
		agentList = append(agentList, fmt.Sprintf("%s (%d)", agent, count))
	}
	if len(agentList) > 0 {
		parts = append(parts, fmt.Sprintf("using agents: %s", strings.Join(agentList, ", ")))
	}

	return strings.Join(parts, " ")
}

// buildPlanDetails creates detailed description of the execution plan
func (d *DispatcherAgent) buildPlanDetails(plan *ExecutionPlan) string {
	var details []string

	details = append(details, fmt.Sprintf("Execution Plan ID: %s", plan.ID))
	details = append(details, fmt.Sprintf("Estimated Duration: %s", plan.EstimatedDuration))
	details = append(details, "")

	// Detail each stage
	for i, stage := range plan.Stages {
		details = append(details, fmt.Sprintf("Stage %d (%s):", i+1, stage.ID))
		
		for _, taskID := range stage.Tasks {
			// Find task details
			for _, task := range plan.Tasks {
				if task.ID == taskID {
					details = append(details, fmt.Sprintf("  - %s: %s", 
						task.Agent, 
						truncateString(task.Request.Prompt, 60)))
					if len(task.Dependencies) > 0 {
						details = append(details, fmt.Sprintf("    Dependencies: %s", 
							strings.Join(task.Dependencies, ", ")))
					}
					break
				}
			}
		}
		details = append(details, "")
	}

	return strings.Join(details, "\n")
}

