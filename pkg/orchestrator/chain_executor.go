package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// ChainExecutor handles the execution of task plans with proper sequencing
type ChainExecutor struct {
	orchestrator *Orchestrator
	plan         *TaskPlan
	state        *TaskState
}

// NewChainExecutor creates a new chain executor
func NewChainExecutor(orchestrator *Orchestrator) *ChainExecutor {
	return &ChainExecutor{
		orchestrator: orchestrator,
	}
}

// ExecutePlan executes a complete task plan
func (e *ChainExecutor) ExecutePlan(ctx context.Context, plan *TaskPlan) (*ChainResult, error) {
	logger.Info("🔗 Starting chain execution for plan: %s", plan.Name)

	e.plan = plan
	e.state = e.orchestrator.stateManager.CreateState(plan.Description)

	// Initialize state metadata with plan context
	e.state.Metadata["plan_id"] = plan.ID
	e.state.Metadata["plan_name"] = plan.Name
	e.state.Metadata["task_outputs"] = make(map[string]string)

	startTime := time.Now()
	plan.Status = PlanStatusExecuting

	// Execute tasks based on dependencies
	executedCount := 0
	maxIterations := len(plan.Tasks) * 2 // Safety limit

	for !plan.IsComplete() && executedCount < maxIterations {
		select {
		case <-ctx.Done():
			plan.Status = PlanStatusCancelled
			return e.buildResult(startTime), ctx.Err()
		default:
		}

		// Get next ready task
		nextTask, err := plan.GetNextTask()
		if err != nil {
			// No tasks ready, check if we're stuck
			if plan.HasFailures() {
				plan.Status = PlanStatusFailed
				break
			}
			// This shouldn't happen with proper dependency management
			logger.Error("No tasks ready but plan not complete")
			break
		}

		logger.Info("📌 Executing task: %s (ID: %s)", nextTask.Name, nextTask.ID)

		// Execute the task
		if err := e.executeTask(ctx, nextTask); err != nil {
			logger.Error("Task %s failed: %v", nextTask.ID, err)
			nextTask.Error = err.Error()
			plan.UpdateTaskStatus(nextTask.ID, TaskStatusFailed)

			// Decide whether to continue or fail the plan
			if e.shouldFailPlan(nextTask) {
				plan.Status = PlanStatusFailed
				break
			}
		}

		executedCount++
	}

	// Update final plan status
	plan.updatePlanStatus()

	result := e.buildResult(startTime)
	logger.Info("🏁 Chain execution completed: status=%s, tasks_executed=%d, duration=%v",
		plan.Status, executedCount, result.Duration)

	return result, nil
}

// executeTask executes a single task in the plan
func (e *ChainExecutor) executeTask(ctx context.Context, task *SubTask) error {
	// Update task status
	if err := e.plan.UpdateTaskStatus(task.ID, TaskStatusRunning); err != nil {
		return err
	}

	// Prepare input with context from previous tasks
	input := e.prepareTaskInput(task)

	// Create routing decision for the task
	decision := &RouteDecision{
		TargetAgent: task.AgentType,
		Instruction: input,
		Parameters:  task.Metadata,
	}

	// Get the agent
	agent, err := e.orchestrator.registry.GetAgent(task.AgentType)
	if err != nil {
		return fmt.Errorf("agent not found: %s", task.AgentType)
	}

	// Execute with the agent
	response, err := agent.Execute(ctx, decision, e.state)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Update task with results
	if err := e.plan.UpdateTaskResult(task.ID, response.Response, response.NextAction); err != nil {
		return err
	}

	// Store output in state for use by subsequent tasks
	taskOutputs := e.state.Metadata["task_outputs"].(map[string]string)
	taskOutputs[task.ID] = response.Response

	// Handle NextAction if provided
	if response.NextAction != nil {
		e.handleNextAction(task, response.NextAction)
	}

	// Update task status based on response
	if response.Status == "success" {
		return e.plan.UpdateTaskStatus(task.ID, TaskStatusCompleted)
	} else {
		return e.plan.UpdateTaskStatus(task.ID, TaskStatusFailed)
	}
}

// prepareTaskInput prepares the input for a task, including context from previous tasks
func (e *ChainExecutor) prepareTaskInput(task *SubTask) string {
	input := task.Input

	// Add context from dependencies if available
	if deps, hasDeps := e.plan.Dependencies[task.ID]; hasDeps {
		taskOutputs := e.state.Metadata["task_outputs"].(map[string]string)

		contextParts := []string{}
		for _, depID := range deps {
			if output, exists := taskOutputs[depID]; exists {
				depTask, _ := e.plan.GetTask(depID)
				if depTask != nil {
					contextParts = append(contextParts,
						fmt.Sprintf("Previous step (%s) output:\n%s", depTask.Name, output))
				}
			}
		}

		if len(contextParts) > 0 {
			input = fmt.Sprintf("%s\n\nContext from previous steps:\n%s",
				input, strings.Join(contextParts, "\n\n"))
		}
	}

	return input
}

// handleNextAction processes a NextAction suggestion from an agent
func (e *ChainExecutor) handleNextAction(currentTask *SubTask, nextAction *RouteDecision) {
	logger.Debug("Agent suggested next action: %s", nextAction.TargetAgent)

	// Check if we can map this to an existing task in the plan
	for i := range e.plan.Tasks {
		task := &e.plan.Tasks[i]
		if task.Status == TaskStatusPending && task.AgentType == nextAction.TargetAgent {
			// Found a matching pending task, update its input if provided
			if nextAction.Instruction != "" {
				task.Input = fmt.Sprintf("%s\n\nAdditional context: %s",
					task.Input, nextAction.Instruction)
			}
			logger.Debug("Mapped NextAction to existing task: %s", task.ID)
			return
		}
	}

	// If no matching task exists, we could dynamically add one (future enhancement)
	logger.Debug("NextAction doesn't match any pending tasks, continuing with plan")
}

// shouldFailPlan determines if a task failure should fail the entire plan
func (e *ChainExecutor) shouldFailPlan(task *SubTask) bool {
	// Check if task is marked as critical
	if critical, ok := task.Metadata["critical"].(bool); ok && critical {
		return true
	}

	// Check if other tasks depend on this one
	for _, t := range e.plan.Tasks {
		if deps, hasDeps := e.plan.Dependencies[t.ID]; hasDeps {
			for _, depID := range deps {
				if depID == task.ID && t.Status == TaskStatusPending {
					// A pending task depends on this failed task
					logger.Warn("Task %s has dependent tasks, failing plan", task.ID)
					return true
				}
			}
		}
	}

	return false
}

// buildResult builds the chain execution result
func (e *ChainExecutor) buildResult(startTime time.Time) *ChainResult {
	completedTasks := 0
	failedTasks := 0

	for _, task := range e.plan.Tasks {
		switch task.Status {
		case TaskStatusCompleted:
			completedTasks++
		case TaskStatusFailed:
			failedTasks++
		}
	}

	return &ChainResult{
		PlanID:         e.plan.ID,
		PlanName:       e.plan.Name,
		Status:         e.plan.Status,
		TotalTasks:     len(e.plan.Tasks),
		CompletedTasks: completedTasks,
		FailedTasks:    failedTasks,
		Duration:       time.Since(startTime),
		TaskResults:    e.plan.Tasks,
		State:          e.state,
	}
}

// ChainResult represents the result of executing a task plan
type ChainResult struct {
	PlanID         string        `json:"plan_id"`
	PlanName       string        `json:"plan_name"`
	Status         PlanStatus    `json:"status"`
	TotalTasks     int           `json:"total_tasks"`
	CompletedTasks int           `json:"completed_tasks"`
	FailedTasks    int           `json:"failed_tasks"`
	Duration       time.Duration `json:"duration"`
	TaskResults    []SubTask     `json:"task_results"`
	State          *TaskState    `json:"state"`
}

// ExecuteChain is a convenience method on the Orchestrator to execute a task plan
func (o *Orchestrator) ExecuteChain(ctx context.Context, plan *TaskPlan) (*ChainResult, error) {
	executor := NewChainExecutor(o)
	return executor.ExecutePlan(ctx, plan)
}
