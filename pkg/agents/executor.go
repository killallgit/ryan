package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// Executor handles parallel and sequential agent execution
type Executor struct {
	orchestrator    *Orchestrator
	workerPool      *WorkerPool
	taskQueue       *TaskQueue
	dependencyGraph *DependencyGraph
	progressTracker *ProgressTracker
	log             *logger.Logger
}

// NewExecutor creates a new executor
func NewExecutor() *Executor {
	return &Executor{
		workerPool:      NewWorkerPool(10), // 10 concurrent workers
		taskQueue:       NewTaskQueue(),
		dependencyGraph: NewDependencyGraph(),
		progressTracker: NewProgressTracker(),
		log:             logger.WithComponent("executor"),
	}
}

// SetOrchestrator sets the orchestrator reference
func (e *Executor) SetOrchestrator(o *Orchestrator) {
	e.orchestrator = o
}

// ExecutePlan executes an execution plan
func (e *Executor) ExecutePlan(ctx context.Context, plan *ExecutionPlan, execContext *ExecutionContext) ([]TaskResult, error) {
	e.log.Info("Executing plan", "plan_id", plan.ID, "tasks", len(plan.Tasks), "stages", len(plan.Stages))

	// Initialize progress tracking
	e.progressTracker.StartPlan(plan.ID, len(plan.Tasks))
	defer e.progressTracker.CompletePlan(plan.ID)

	// Build dependency graph
	e.dependencyGraph.BuildFromPlan(plan)

	// Execute stages in order
	results := make([]TaskResult, 0, len(plan.Tasks))
	resultsMux := &sync.Mutex{}

	for _, stage := range plan.Stages {
		e.log.Debug("Executing stage", "stage_id", stage.ID, "tasks", len(stage.Tasks))

		// Execute tasks in this stage concurrently
		stageResults, err := e.executeStage(ctx, stage, plan, execContext, resultsMux)
		if err != nil {
			return results, fmt.Errorf("stage %s failed: %w", stage.ID, err)
		}

		results = append(results, stageResults...)
	}

	return results, nil
}

// executeStage executes all tasks in a stage concurrently
func (e *Executor) executeStage(ctx context.Context, stage Stage, plan *ExecutionPlan, execContext *ExecutionContext, resultsMux *sync.Mutex) ([]TaskResult, error) {
	var wg sync.WaitGroup
	results := make([]TaskResult, 0, len(stage.Tasks))
	errors := make(chan error, len(stage.Tasks))

	for _, taskID := range stage.Tasks {
		// Find task in plan
		var task *Task
		for i := range plan.Tasks {
			if plan.Tasks[i].ID == taskID {
				task = &plan.Tasks[i]
				break
			}
		}
		if task == nil {
			continue
		}

		wg.Add(1)
		go func(t Task) {
			defer wg.Done()

			result, err := e.executeTask(ctx, t, execContext)
			if err != nil {
				errors <- err
				return
			}

			resultsMux.Lock()
			results = append(results, result)
			resultsMux.Unlock()
		}(*task)
	}

	// Wait for all tasks to complete
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// executeTask executes a single task
func (e *Executor) executeTask(ctx context.Context, task Task, execContext *ExecutionContext) (TaskResult, error) {
	startTime := time.Now()
	e.log.Info("Executing task", "task_id", task.ID, "agent", task.Agent)

	// Update progress
	e.progressTracker.StartTask(task.ID)
	defer e.progressTracker.CompleteTask(task.ID)

	// Get agent
	agent, err := e.orchestrator.GetAgent(task.Agent)
	if err != nil {
		return TaskResult{
			Task: task,
			Result: AgentResult{
				Success: false,
				Summary: fmt.Sprintf("Agent %s not found", task.Agent),
				Details: err.Error(),
			},
			Error: err,
		}, err
	}

	// Prepare agent request with context
	request := task.Request
	request.Context = e.prepareTaskContext(task, execContext)

	// Execute agent
	result, err := agent.Execute(ctx, request)
	if err != nil {
		e.log.Error("Task execution failed", "task_id", task.ID, "error", err)
		return TaskResult{
			Task:   task,
			Result: result,
			Error:  err,
		}, nil // Don't propagate error to allow other tasks to continue
	}

	// Update execution context with results
	e.updateExecutionContext(task, result, execContext)

	// Send progress update
	if execContext.Progress != nil {
		select {
		case execContext.Progress <- ProgressUpdate{
			TaskID:    task.ID,
			Agent:     task.Agent,
			Status:    "completed",
			Timestamp: time.Now(),
		}:
		default:
			// Channel full, skip
		}
	}

	return TaskResult{
		Task:      task,
		Result:    result,
		Error:     nil,
		StartTime: startTime,
		EndTime:   time.Now(),
	}, nil
}

// prepareTaskContext prepares the context for a task
func (e *Executor) prepareTaskContext(task Task, execContext *ExecutionContext) map[string]interface{} {
	context := make(map[string]interface{})

	// Copy base context
	for k, v := range task.Request.Context {
		context[k] = v
	}

	// Add execution context references
	context["execution_context"] = execContext
	context["orchestrator"] = e.orchestrator
	context["shared_data"] = execContext.SharedData
	context["file_context"] = execContext.FileContext

	return context
}

// updateExecutionContext updates the execution context with task results
func (e *Executor) updateExecutionContext(task Task, result AgentResult, execContext *ExecutionContext) {
	// Store result in shared data
	execContext.mu.Lock()
	defer execContext.mu.Unlock()

	if execContext.SharedData == nil {
		execContext.SharedData = make(map[string]interface{})
	}

	// Store task result
	execContext.SharedData[fmt.Sprintf("task_%s_result", task.ID)] = result

	// Update file context if agent processed files
	if len(result.Metadata.FilesProcessed) > 0 {
		for _, file := range result.Metadata.FilesProcessed {
			// Check if file already in context
			found := false
			for _, f := range execContext.FileContext {
				if f.Path == file {
					found = true
					break
				}
			}
			if !found {
				execContext.FileContext = append(execContext.FileContext, FileInfo{
					Path:         file,
					LastModified: time.Now(),
				})
			}
		}
	}

	// Store any artifacts
	if result.Artifacts != nil {
		if execContext.Artifacts == nil {
			execContext.Artifacts = make(map[string]interface{})
		}
		for k, v := range result.Artifacts {
			execContext.Artifacts[fmt.Sprintf("%s_%s", task.Agent, k)] = v
		}
	}
}

// WorkerPool manages concurrent task execution
type WorkerPool struct {
	workers    int
	taskChan   chan func()
	workerWg   sync.WaitGroup
	shutdownCh chan struct{}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	wp := &WorkerPool{
		workers:    workers,
		taskChan:   make(chan func(), workers*2),
		shutdownCh: make(chan struct{}),
	}

	// Start workers
	for i := 0; i < workers; i++ {
		wp.workerWg.Add(1)
		go wp.worker()
	}

	return wp
}

// worker processes tasks from the queue
func (wp *WorkerPool) worker() {
	defer wp.workerWg.Done()

	for {
		select {
		case task := <-wp.taskChan:
			if task != nil {
				task()
			}
		case <-wp.shutdownCh:
			return
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task func()) {
	select {
	case wp.taskChan <- task:
	case <-wp.shutdownCh:
		// Pool is shutting down
	}
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown() {
	close(wp.shutdownCh)
	wp.workerWg.Wait()
}

// TaskQueue manages task queueing
type TaskQueue struct {
	queue []Task
	mu    sync.Mutex
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		queue: make([]Task, 0),
	}
}

// Enqueue adds a task to the queue
func (tq *TaskQueue) Enqueue(task Task) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.queue = append(tq.queue, task)
}

// Dequeue removes and returns a task from the queue
func (tq *TaskQueue) Dequeue() (Task, bool) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if len(tq.queue) == 0 {
		return Task{}, false
	}

	task := tq.queue[0]
	tq.queue = tq.queue[1:]
	return task, true
}

// DependencyGraph manages task dependencies
type DependencyGraph struct {
	dependencies map[string][]string
	mu           sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		dependencies: make(map[string][]string),
	}
}

// BuildFromPlan builds the dependency graph from an execution plan
func (dg *DependencyGraph) BuildFromPlan(plan *ExecutionPlan) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	dg.dependencies = make(map[string][]string)
	for _, task := range plan.Tasks {
		dg.dependencies[task.ID] = task.Dependencies
	}
}

// CanExecute checks if a task can be executed (all dependencies satisfied)
func (dg *DependencyGraph) CanExecute(taskID string, completed map[string]bool) bool {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps, exists := dg.dependencies[taskID]
	if !exists {
		return true
	}

	for _, dep := range deps {
		if !completed[dep] {
			return false
		}
	}

	return true
}

// ProgressTracker tracks execution progress
type ProgressTracker struct {
	plans map[string]*PlanProgress
	mu    sync.RWMutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		plans: make(map[string]*PlanProgress),
	}
}

// StartPlan starts tracking a plan
func (pt *ProgressTracker) StartPlan(planID string, totalTasks int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.plans[planID] = &PlanProgress{
		PlanID:     planID,
		TotalTasks: totalTasks,
		StartTime:  time.Now(),
		TaskStatus: make(map[string]string),
	}
}

// StartTask marks a task as started
func (pt *ProgressTracker) StartTask(taskID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, progress := range pt.plans {
		progress.TaskStatus[taskID] = "running"
	}
}

// CompleteTask marks a task as completed
func (pt *ProgressTracker) CompleteTask(taskID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, progress := range pt.plans {
		if progress.TaskStatus[taskID] == "running" {
			progress.TaskStatus[taskID] = "completed"
			progress.CompletedTasks++
		}
	}
}

// CompletePlan marks a plan as completed
func (pt *ProgressTracker) CompletePlan(planID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if progress, exists := pt.plans[planID]; exists {
		progress.EndTime = time.Now()
		progress.Completed = true
	}
}

// GetProgress returns the current progress of a plan
func (pt *ProgressTracker) GetProgress(planID string) (*PlanProgress, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress, exists := pt.plans[planID]
	return progress, exists
}

// PlanProgress represents the progress of a plan
type PlanProgress struct {
	PlanID         string
	TotalTasks     int
	CompletedTasks int
	StartTime      time.Time
	EndTime        time.Time
	Completed      bool
	TaskStatus     map[string]string
}