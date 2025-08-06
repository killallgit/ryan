package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/agents/interfaces"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// Orchestrator coordinates agent execution with advanced planning and feedback loops
type Orchestrator struct {
	agents         map[string]interfaces.Agent
	executor       *Executor
	contextManager *ContextManager
	planner        *Planner
	feedbackLoop   *FeedbackLoop
	toolRegistry   *tools.Registry
	log            *logger.Logger
	mu             sync.RWMutex
}

// NewOrchestrator creates a new agent orchestrator
func NewOrchestrator() *Orchestrator {
	o := &Orchestrator{
		agents:         make(map[string]interfaces.Agent),
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

	// Register ScrumMaster agent for complex project management
	scrumMaster := NewScrumMaster(o)
	if err := o.RegisterAgent(scrumMaster); err != nil {
		return fmt.Errorf("failed to register scrummaster agent: %w", err)
	}

	o.log.Info("Registered built-in agents", "count", 6) // dispatcher + 5 agents
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

	// Check if this is a project-level request that should be handled by ScrumMaster
	if o.planner.IsProjectLevelRequest(request) {
		o.log.Info("Detected project-level request, routing to ScrumMaster")
		if scrumMaster, err := o.GetAgent("scrummaster"); err == nil {
			agentRequest := AgentRequest{
				Prompt:  request,
				Options: options,
			}
			return scrumMaster.Execute(ctx, agentRequest)
		}
	}

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
func (o *Orchestrator) GetAgent(name string) (interfaces.Agent, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, exists := o.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}
	return agent, nil
}

// ListAgents returns all registered agents
func (o *Orchestrator) ListAgents() []interfaces.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]interfaces.Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	return agents
}

// GetToolRegistry returns the tool registry
func (o *Orchestrator) GetToolRegistry() *tools.Registry {
	return o.toolRegistry
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
		// Cast result to AgentResult
		agentResult, ok := result.Result.(AgentResult)
		if !ok {
			allSuccess = false
			failedTasks = append(failedTasks, result.Task.ID)
			continue
		}

		if !agentResult.Success {
			allSuccess = false
			failedTasks = append(failedTasks, result.Task.ID)
		} else {
			successfulTasks = append(successfulTasks, result.Task.ID)
		}

		// Collect details
		if agentResult.Details != "" {
			allDetails = append(allDetails, fmt.Sprintf("[%s]: %s", result.Task.Agent, agentResult.Details))
		}

		// Collect unique tools and files
		// TODO: Add ToolsUsed and FilesProcessed to metadata when needed
		/*
			for _, tool := range agentResult.Metadata.ToolsUsed {
				toolsMap[tool] = true
			}
			for _, file := range agentResult.Metadata.FilesProcessed {
				filesMap[file] = true
			}
		*/
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
	ctx.Mu.RLock()
	if ctx.SharedData != nil {
		// Create a copy to avoid race conditions
		sharedDataCopy := make(map[string]interface{})
		for k, v := range ctx.SharedData {
			sharedDataCopy[k] = v
		}
		artifacts["shared_data"] = sharedDataCopy
	}
	ctx.Mu.RUnlock()
	if len(ctx.FileContext) > 0 {
		artifacts["files"] = ctx.FileContext
	}

	return AgentResult{
		Success:   allSuccess,
		Summary:   summary,
		Details:   details,
		Artifacts: artifacts,
		Metadata:  AgentMetadata{
			// ToolsUsed:      toolsUsed, // TODO: Add to metadata
			// FilesProcessed: filesProcessed, // TODO: Add to metadata
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
