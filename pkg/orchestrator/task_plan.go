package orchestrator

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// TaskPlan represents a structured plan for complex task execution
type TaskPlan struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Tasks        []SubTask              `json:"tasks"`
	Dependencies map[string][]string    `json:"dependencies"` // task_id -> [depends_on_ids]
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Status       PlanStatus             `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// SubTask represents an individual task within a plan
type SubTask struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	AgentType      AgentType              `json:"agent_type"`
	Input          string                 `json:"input"`
	ExpectedOutput string                 `json:"expected_output,omitempty"`
	Status         TaskStatus             `json:"status"`
	Result         string                 `json:"result,omitempty"`
	Error          string                 `json:"error,omitempty"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	NextAction     *RouteDecision         `json:"next_action,omitempty"`
}

// PlanStatus represents the overall status of a task plan
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusReady     PlanStatus = "ready"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
	PlanStatusCancelled PlanStatus = "cancelled"
)

// TaskStatus represents the status of an individual task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusReady     TaskStatus = "ready"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusSkipped   TaskStatus = "skipped"
)

// generateID generates a random ID for tasks and plans
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// NewTaskPlan creates a new task plan
func NewTaskPlan(name, description string) *TaskPlan {
	return &TaskPlan{
		ID:           generateID(),
		Name:         name,
		Description:  description,
		Tasks:        []SubTask{},
		Dependencies: make(map[string][]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       PlanStatusDraft,
		Metadata:     make(map[string]interface{}),
	}
}

// AddTask adds a subtask to the plan
func (p *TaskPlan) AddTask(task SubTask) *TaskPlan {
	if task.ID == "" {
		task.ID = generateID()
	}
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	p.Tasks = append(p.Tasks, task)
	p.UpdatedAt = time.Now()
	return p
}

// AddDependency adds a dependency between tasks
func (p *TaskPlan) AddDependency(taskID string, dependsOnIDs ...string) *TaskPlan {
	if p.Dependencies == nil {
		p.Dependencies = make(map[string][]string)
	}
	p.Dependencies[taskID] = append(p.Dependencies[taskID], dependsOnIDs...)
	p.UpdatedAt = time.Now()
	return p
}

// GetTask retrieves a task by ID
func (p *TaskPlan) GetTask(taskID string) (*SubTask, error) {
	for i := range p.Tasks {
		if p.Tasks[i].ID == taskID {
			return &p.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %s not found", taskID)
}

// UpdateTaskStatus updates the status of a specific task
func (p *TaskPlan) UpdateTaskStatus(taskID string, status TaskStatus) error {
	task, err := p.GetTask(taskID)
	if err != nil {
		return err
	}

	task.Status = status
	now := time.Now()

	switch status {
	case TaskStatusRunning:
		task.StartedAt = &now
	case TaskStatusCompleted, TaskStatusFailed:
		task.CompletedAt = &now
	}

	p.UpdatedAt = time.Now()
	p.updatePlanStatus()
	return nil
}

// UpdateTaskResult updates the result of a completed task
func (p *TaskPlan) UpdateTaskResult(taskID string, result string, nextAction *RouteDecision) error {
	task, err := p.GetTask(taskID)
	if err != nil {
		return err
	}

	task.Result = result
	task.NextAction = nextAction
	p.UpdatedAt = time.Now()
	return nil
}

// GetReadyTasks returns tasks that are ready to execute (dependencies met)
func (p *TaskPlan) GetReadyTasks() []SubTask {
	ready := []SubTask{}

	for _, task := range p.Tasks {
		if task.Status != TaskStatusPending {
			continue
		}

		// Check if all dependencies are completed
		deps, hasDeps := p.Dependencies[task.ID]
		if !hasDeps {
			ready = append(ready, task)
			continue
		}

		allDepsComplete := true
		for _, depID := range deps {
			depTask, err := p.GetTask(depID)
			if err != nil || depTask.Status != TaskStatusCompleted {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			ready = append(ready, task)
		}
	}

	return ready
}

// GetNextTask returns the next task to execute based on dependencies
func (p *TaskPlan) GetNextTask() (*SubTask, error) {
	readyTasks := p.GetReadyTasks()
	if len(readyTasks) == 0 {
		return nil, fmt.Errorf("no tasks ready for execution")
	}
	return &readyTasks[0], nil
}

// IsComplete checks if all tasks in the plan are complete
func (p *TaskPlan) IsComplete() bool {
	for _, task := range p.Tasks {
		if task.Status != TaskStatusCompleted && task.Status != TaskStatusSkipped {
			return false
		}
	}
	return true
}

// HasFailures checks if any task has failed
func (p *TaskPlan) HasFailures() bool {
	for _, task := range p.Tasks {
		if task.Status == TaskStatusFailed {
			return true
		}
	}
	return false
}

// updatePlanStatus updates the overall plan status based on task states
func (p *TaskPlan) updatePlanStatus() {
	if p.HasFailures() {
		p.Status = PlanStatusFailed
		return
	}

	if p.IsComplete() {
		p.Status = PlanStatusCompleted
		return
	}

	// Check if any task is running
	for _, task := range p.Tasks {
		if task.Status == TaskStatusRunning {
			p.Status = PlanStatusExecuting
			return
		}
	}

	// If we have ready tasks, we're ready to execute
	if len(p.GetReadyTasks()) > 0 {
		p.Status = PlanStatusReady
	}
}

// ToJSON serializes the plan to JSON
func (p *TaskPlan) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON deserializes a plan from JSON
func TaskPlanFromJSON(data string) (*TaskPlan, error) {
	var plan TaskPlan
	if err := json.Unmarshal([]byte(data), &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// BuildSequentialPlan creates a simple sequential plan from a list of tasks
func BuildSequentialPlan(name string, tasks ...SubTask) *TaskPlan {
	plan := NewTaskPlan(name, "Sequential execution plan")

	var prevID string
	for i, task := range tasks {
		if task.ID == "" {
			task.ID = fmt.Sprintf("task_%d", i+1)
		}
		plan.AddTask(task)

		// Add dependency on previous task
		if prevID != "" {
			plan.AddDependency(task.ID, prevID)
		}
		prevID = task.ID
	}

	return plan
}

// BuildParallelPlan creates a plan where all tasks can run in parallel
func BuildParallelPlan(name string, tasks ...SubTask) *TaskPlan {
	plan := NewTaskPlan(name, "Parallel execution plan")

	for i, task := range tasks {
		if task.ID == "" {
			task.ID = fmt.Sprintf("task_%d", i+1)
		}
		plan.AddTask(task)
		// No dependencies, all can run in parallel
	}

	return plan
}
