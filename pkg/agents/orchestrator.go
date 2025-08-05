package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// Orchestrator coordinates agent execution with advanced planning and feedback loops
type Orchestrator struct {
	agents         map[string]Agent
	executor       *Executor
	contextManager *ContextManager
	planner        *Planner
	feedbackLoop   *FeedbackLoop
	log            *logger.Logger
	mu             sync.RWMutex
}

// NewOrchestrator creates a new agent orchestrator
func NewOrchestrator() *Orchestrator {
	o := &Orchestrator{
		agents:         make(map[string]Agent),
		executor:       NewExecutor(),
		contextManager: NewContextManager(),
		planner:        NewPlanner(),
		feedbackLoop:   NewFeedbackLoop(),
		log:            logger.WithComponent("orchestrator"),
	}
	
	// Set up circular references
	o.executor.SetOrchestrator(o)
	o.planner.SetOrchestrator(o)
	o.feedbackLoop.SetOrchestrator(o)
	
	// Register built-in dispatcher agent
	dispatcher := NewDispatcherAgent(o)
	o.RegisterAgent(dispatcher)
	
	return o
}

// RegisterBuiltinAgents registers all built-in agents
func (o *Orchestrator) RegisterBuiltinAgents(toolRegistry *tools.Registry) error {
	// Register file operations agent
	fileOpsAgent := NewFileOperationsAgent(toolRegistry)
	if err := o.RegisterAgent(fileOpsAgent); err != nil {
		return fmt.Errorf("failed to register file operations agent: %w", err)
	}

	// Register code analysis agent
	codeAnalysisAgent := NewCodeAnalysisAgent()
	if err := o.RegisterAgent(codeAnalysisAgent); err != nil {
		return fmt.Errorf("failed to register code analysis agent: %w", err)
	}

	// Register code review agent
	codeReviewAgent := NewCodeReviewAgent()
	if err := o.RegisterAgent(codeReviewAgent); err != nil {
		return fmt.Errorf("failed to register code review agent: %w", err)
	}

	// Register search agent
	searchAgent := NewSearchAgent(toolRegistry)
	if err := o.RegisterAgent(searchAgent); err != nil {
		return fmt.Errorf("failed to register search agent: %w", err)
	}

	o.log.Info("Registered built-in agents", "count", 5) // dispatcher + 4 agents
	return nil
}

// RegisterAgent registers an agent with the orchestrator
func (o *Orchestrator) RegisterAgent(agent Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	name := agent.Name()
	if _, exists := o.agents[name]; exists {
		return fmt.Errorf("agent %s already registered", name)
	}

	o.agents[name] = agent
	o.log.Info("Registered agent", "name", name, "description", agent.Description())
	return nil
}

// Execute analyzes the request and orchestrates agent execution
func (o *Orchestrator) Execute(ctx context.Context, request string, options map[string]interface{}) (AgentResult, error) {
	startTime := time.Now()
	o.log.Info("Executing request", "request_preview", truncateString(request, 100))

	// Create execution context
	execContext := &ExecutionContext{
		SessionID:   generateID(),
		RequestID:   generateID(),
		UserPrompt:  request,
		SharedData:  make(map[string]interface{}),
		FileContext: []FileInfo{},
		Progress:    make(chan ProgressUpdate, 100),
		Options:     options,
	}

	// Plan the execution
	plan, err := o.planner.CreateExecutionPlan(ctx, request, execContext)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to create execution plan",
			Details: err.Error(),
			Metadata: AgentMetadata{
				StartTime: startTime,
				EndTime:   time.Now(),
				Duration:  time.Since(startTime),
			},
		}, err
	}

	o.log.Info("Created execution plan", "tasks", len(plan.Tasks), "stages", len(plan.Stages))

	// Execute the plan
	results, err := o.executor.ExecutePlan(ctx, plan, execContext)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Execution failed",
			Details: err.Error(),
			Metadata: AgentMetadata{
				StartTime: startTime,
				EndTime:   time.Now(),
				Duration:  time.Since(startTime),
			},
		}, err
	}

	// Aggregate results
	finalResult := o.aggregateResults(results, plan, execContext)
	finalResult.Metadata.StartTime = startTime
	finalResult.Metadata.EndTime = time.Now()
	finalResult.Metadata.Duration = time.Since(startTime)

	return finalResult, nil
}

// ExecuteWithPlan executes a pre-built execution plan
func (o *Orchestrator) ExecuteWithPlan(ctx context.Context, plan *ExecutionPlan, execContext *ExecutionContext) ([]TaskResult, error) {
	return o.executor.ExecutePlan(ctx, plan, execContext)
}

// GetAgent retrieves a registered agent by name
func (o *Orchestrator) GetAgent(name string) (Agent, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, exists := o.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}
	return agent, nil
}

// ListAgents returns all registered agents
func (o *Orchestrator) ListAgents() []Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ProcessFeedback handles feedback from agent execution
func (o *Orchestrator) ProcessFeedback(ctx context.Context, feedback *FeedbackRequest) error {
	return o.feedbackLoop.ProcessFeedback(ctx, feedback)
}

// aggregateResults combines multiple task results into a final result
func (o *Orchestrator) aggregateResults(results []TaskResult, plan *ExecutionPlan, ctx *ExecutionContext) AgentResult {
	// Check if all tasks succeeded
	allSuccess := true
	var failedTasks []string
	var successfulTasks []string
	var allDetails []string
	var toolsUsed []string
	var filesProcessed []string
	toolsMap := make(map[string]bool)
	filesMap := make(map[string]bool)

	for _, result := range results {
		if !result.Result.Success {
			allSuccess = false
			failedTasks = append(failedTasks, result.Task.ID)
		} else {
			successfulTasks = append(successfulTasks, result.Task.ID)
		}
		
		// Collect details
		if result.Result.Details != "" {
			allDetails = append(allDetails, fmt.Sprintf("[%s]: %s", result.Task.Agent, result.Result.Details))
		}
		
		// Collect unique tools and files
		for _, tool := range result.Result.Metadata.ToolsUsed {
			toolsMap[tool] = true
		}
		for _, file := range result.Result.Metadata.FilesProcessed {
			filesMap[file] = true
		}
	}

	// Convert maps to slices
	for tool := range toolsMap {
		toolsUsed = append(toolsUsed, tool)
	}
	for file := range filesMap {
		filesProcessed = append(filesProcessed, file)
	}

	// Build summary
	summary := fmt.Sprintf("Executed %d tasks (%d successful, %d failed)", 
		len(results), len(successfulTasks), len(failedTasks))

	// Combine all details
	details := ""
	if len(allDetails) > 0 {
		details = joinStrings(allDetails, "\n\n")
	}

	// Create artifacts map from context
	artifacts := make(map[string]interface{})
	if ctx.SharedData != nil {
		artifacts["shared_data"] = ctx.SharedData
	}
	if len(ctx.FileContext) > 0 {
		artifacts["files"] = ctx.FileContext
	}

	return AgentResult{
		Success: allSuccess,
		Summary: summary,
		Details: details,
		Artifacts: artifacts,
		Metadata: AgentMetadata{
			ToolsUsed:      toolsUsed,
			FilesProcessed: filesProcessed,
		},
	}
}

// Helper functions

func joinStrings(strings []string, separator string) string {
	result := ""
	for i, s := range strings {
		if i > 0 {
			result += separator
		}
		result += s
	}
	return result
}