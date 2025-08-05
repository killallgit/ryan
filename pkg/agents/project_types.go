package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusPlanning   ProjectStatus = "planning"
	ProjectStatusInProgress ProjectStatus = "in_progress"
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusFailed     ProjectStatus = "failed"
	ProjectStatusOnHold     ProjectStatus = "on_hold"
)

// SprintStatus represents the status of a sprint
type SprintStatus string

const (
	SprintStatusPlanning  SprintStatus = "planning"
	SprintStatusReady     SprintStatus = "ready"
	SprintStatusActive    SprintStatus = "active"
	SprintStatusCompleted SprintStatus = "completed"
	SprintStatusCancelled SprintStatus = "cancelled"
)

// EpicStatus represents the status of an epic
type EpicStatus string

const (
	EpicStatusTodo       EpicStatus = "todo"
	EpicStatusInProgress EpicStatus = "in_progress"
	EpicStatusDone       EpicStatus = "done"
	EpicStatusBlocked    EpicStatus = "blocked"
)

// StoryStatus represents the status of a user story
type StoryStatus string

const (
	StoryStatusTodo       StoryStatus = "todo"
	StoryStatusInProgress StoryStatus = "in_progress"
	StoryStatusReview     StoryStatus = "review"
	StoryStatusDone       StoryStatus = "done"
	StoryStatusBlocked    StoryStatus = "blocked"
)

// Project represents a complex, multi-phase project
type Project struct {
	ID          string
	Name        string
	Description string
	Epics       []*Epic
	Sprints     []*Sprint
	Context     *ProjectContext
	Status      ProjectStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// Epic represents a large feature or component of work
type Epic struct {
	ID          string
	Title       string
	Description string
	Stories     []*UserStory
	Priority    Priority
	Status      EpicStatus
	StartDate   *time.Time
	EndDate     *time.Time
	Owner       string
}

// Sprint represents a time-boxed iteration of work
type Sprint struct {
	ID        string
	Number    int
	Name      string
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	Status    SprintStatus
	Stories   []*UserStory
	Plans     []*ExecutionPlan
	Capacity  int // in story points
	Velocity  int // actual story points completed
}

// UserStory represents a user-focused piece of functionality
type UserStory struct {
	ID                 string
	Title              string
	Description        string
	AcceptanceCriteria []string
	Points             int // story points for estimation
	Priority           Priority
	Status             StoryStatus
	AssignedAgent      string
	Dependencies       []string
	CompletedAt        *time.Time
}

// ProjectContext extends ExecutionContext for project-level state
type ProjectContext struct {
	*ExecutionContext
	StartTime    time.Time
	Constraints  map[string]interface{}
	Goals        []string
	Stakeholders []string
	Risks        []*Risk
	Decisions    []*Decision
	Metrics      *ProjectMetrics
}

// Risk represents a project risk
type Risk struct {
	ID          string
	Description string
	Impact      string
	Probability string
	Mitigation  string
	Status      string
}

// Decision represents a key project decision
type Decision struct {
	ID          string
	Description string
	Rationale   string
	Timestamp   time.Time
	MadeBy      string
}

// ProjectMetrics tracks project performance metrics
type ProjectMetrics struct {
	PlannedVelocity int
	ActualVelocity  int
	BurndownRate    float64
	DefectRate      float64
	CodeCoverage    float64
	CycleTime       time.Duration
	LeadTime        time.Duration
	BlockedTime     time.Duration
	RetryCount      int
	SuccessRate     float64
}

// SprintResult represents the outcome of a sprint execution
type SprintResult struct {
	SprintID         string
	Success          bool
	CompletedStories []string
	BlockedStories   []string
	Velocity         int
	StartTime        time.Time
	EndTime          time.Time
	Error            error
	Metrics          *SprintMetrics
}

// SprintMetrics contains metrics for a sprint
type SprintMetrics struct {
	PlannedPoints    int
	CompletedPoints  int
	CarryoverPoints  int
	DefectsFound     int
	DefectsResolved  int
	TimeInBlocked    time.Duration
	AverageLeadTime  time.Duration
	TeamSatisfaction float64
}

// ProjectHistory represents archived project information
type ProjectHistory struct {
	Project       *Project
	Result        AgentResult
	ArchivedAt    time.Time
	LessonsLearn  []string
	Retrospective *Retrospective
}

// Retrospective contains retrospective information
type Retrospective struct {
	WhatWentWell     []string
	WhatCouldImprove []string
	ActionItems      []string
	TeamMood         float64
}

// ProjectMonitor handles real-time project monitoring
type ProjectMonitor struct {
	activeMonitors map[string]*Monitor
	alertChannel   chan Alert
	metricsStore   *MetricsStore
	log            *logger.Logger
}

// NewProjectMonitor creates a new project monitor
func NewProjectMonitor() *ProjectMonitor {
	return &ProjectMonitor{
		activeMonitors: make(map[string]*Monitor),
		alertChannel:   make(chan Alert, 100),
		metricsStore:   NewMetricsStore(),
		log:            logger.WithComponent("project_monitor"),
	}
}

// MonitorProject starts monitoring a project
func (pm *ProjectMonitor) MonitorProject(ctx context.Context, project *Project) {
	monitor := &Monitor{
		ProjectID: project.ID,
		StartTime: time.Now(),
		Status:    "active",
	}

	pm.activeMonitors[project.ID] = monitor

	// Start monitoring loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			monitor.Status = "stopped"
			return
		case <-ticker.C:
			pm.checkProjectHealth(project)
		}
	}
}

// checkProjectHealth checks the health of a project
func (pm *ProjectMonitor) checkProjectHealth(project *Project) {
	// Check for blocked stories, delays, etc.
	// This is a placeholder for more sophisticated monitoring
	pm.log.Debug("Checking project health", "project_id", project.ID)
}

// Monitor represents an active project monitor
type Monitor struct {
	ProjectID string
	StartTime time.Time
	Status    string
	Alerts    []Alert
}

// Alert represents a project alert
type Alert struct {
	ID        string
	ProjectID string
	Level     string
	Message   string
	Timestamp time.Time
}

// MetricsStore stores project metrics
type MetricsStore struct {
	metrics map[string]*ProjectMetrics
	mu      sync.RWMutex
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore() *MetricsStore {
	return &MetricsStore{
		metrics: make(map[string]*ProjectMetrics),
	}
}

// ResourceManager manages agent resources and scheduling
type ResourceManager struct {
	agentCapacity    map[string]*AgentCapacity
	taskQueue        *PriorityQueue
	scheduler        *TaskScheduler
	conflictResolver *ConflictResolver
	log              *logger.Logger
}

// NewResourceManager creates a new resource manager
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		agentCapacity:    make(map[string]*AgentCapacity),
		taskQueue:        NewPriorityQueue(),
		scheduler:        NewTaskScheduler(),
		conflictResolver: NewConflictResolver(),
		log:              logger.WithComponent("resource_manager"),
	}
}

// AllocateResources allocates resources for a task
func (rm *ResourceManager) AllocateResources(task Task, agents []Agent) (*ResourceAllocation, error) {
	// Check agent availability
	availableAgents := rm.findAvailableAgents(agents, task)
	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no available agents for task %s", task.ID)
	}

	// Select best agent based on capacity and suitability
	selectedAgent := rm.selectBestAgent(availableAgents, task)

	// Create allocation
	allocation := &ResourceAllocation{
		TaskID:    task.ID,
		AgentName: selectedAgent.Name(),
		StartTime: time.Now(),
		Priority:  task.Priority,
	}

	// Update agent capacity
	rm.updateAgentCapacity(selectedAgent.Name(), -1)

	return allocation, nil
}

// findAvailableAgents finds agents that can handle the task
func (rm *ResourceManager) findAvailableAgents(agents []Agent, task Task) []Agent {
	available := []Agent{}
	for _, agent := range agents {
		if canHandle, _ := agent.CanHandle(task.Request.Prompt); canHandle {
			if capacity, exists := rm.agentCapacity[agent.Name()]; !exists || capacity.Available > 0 {
				available = append(available, agent)
			}
		}
	}
	return available
}

// selectBestAgent selects the best agent for a task
func (rm *ResourceManager) selectBestAgent(agents []Agent, task Task) Agent {
	// Simple selection - in practice would be more sophisticated
	bestAgent := agents[0]
	bestScore := 0.0

	for _, agent := range agents {
		_, confidence := agent.CanHandle(task.Request.Prompt)
		if confidence > bestScore {
			bestScore = confidence
			bestAgent = agent
		}
	}

	return bestAgent
}

// updateAgentCapacity updates an agent's capacity
func (rm *ResourceManager) updateAgentCapacity(agentName string, delta int) {
	if capacity, exists := rm.agentCapacity[agentName]; exists {
		capacity.Available += delta
	} else {
		rm.agentCapacity[agentName] = &AgentCapacity{
			Total:     10, // Default capacity
			Available: 10 + delta,
		}
	}
}

// AgentCapacity represents an agent's resource capacity
type AgentCapacity struct {
	Total     int
	Available int
	Reserved  int
}

// ResourceAllocation represents a resource allocation
type ResourceAllocation struct {
	TaskID    string
	AgentName string
	StartTime time.Time
	EndTime   *time.Time
	Priority  int
}

// PriorityQueue manages tasks by priority
type PriorityQueue struct {
	items []PriorityItem
	mu    sync.Mutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items: make([]PriorityItem, 0),
	}
}

// PriorityItem represents an item in the priority queue
type PriorityItem struct {
	Task     Task
	Priority int
}

// TaskScheduler schedules tasks for execution
type TaskScheduler struct {
	schedule map[time.Time][]Task
	mu       sync.RWMutex
}

// NewTaskScheduler creates a new task scheduler
func NewTaskScheduler() *TaskScheduler {
	return &TaskScheduler{
		schedule: make(map[time.Time][]Task),
	}
}

// ConflictResolver resolves resource conflicts
type ConflictResolver struct {
	strategies []ResolutionStrategy
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{
		strategies: defaultResolutionStrategies(),
	}
}

// ResolutionStrategy defines a conflict resolution strategy
type ResolutionStrategy interface {
	CanResolve(conflict ResourceConflict) bool
	Resolve(conflict ResourceConflict) Resolution
}

// ResourceConflict represents a resource conflict
type ResourceConflict struct {
	Tasks     []Task
	Resources []string
	Type      string
}

// Resolution represents a conflict resolution
type Resolution struct {
	Action        string
	ModifiedTasks []Task
	Explanation   string
}

// defaultResolutionStrategies returns default resolution strategies
func defaultResolutionStrategies() []ResolutionStrategy {
	// Placeholder for actual strategies
	return []ResolutionStrategy{}
}

// StrategySelector selects execution strategies
type StrategySelector struct {
	strategies map[string]ExecutionStrategy
	analyzer   *StrategyAnalyzer
	log        *logger.Logger
}

// NewStrategySelector creates a new strategy selector
func NewStrategySelector() *StrategySelector {
	return &StrategySelector{
		strategies: defaultExecutionStrategies(),
		analyzer:   NewStrategyAnalyzer(),
		log:        logger.WithComponent("strategy_selector"),
	}
}

// SelectStrategy selects the best execution strategy
func (ss *StrategySelector) SelectStrategy(project *Project, context *ProjectContext) ExecutionStrategy {
	// Analyze project characteristics
	characteristics := ss.analyzer.AnalyzeProject(project)

	// Select best matching strategy
	var bestStrategy ExecutionStrategy
	bestScore := 0.0

	for _, strategy := range ss.strategies {
		score := strategy.ScoreForProject(characteristics)
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	return bestStrategy
}

// ExecutionStrategy defines an execution strategy
type ExecutionStrategy interface {
	Name() string
	ScoreForProject(characteristics ProjectCharacteristics) float64
	Apply(project *Project) *Project
}

// StrategyAnalyzer analyzes projects for strategy selection
type StrategyAnalyzer struct{}

// NewStrategyAnalyzer creates a new strategy analyzer
func NewStrategyAnalyzer() *StrategyAnalyzer {
	return &StrategyAnalyzer{}
}

// AnalyzeProject analyzes project characteristics
func (sa *StrategyAnalyzer) AnalyzeProject(project *Project) ProjectCharacteristics {
	return ProjectCharacteristics{
		Complexity:    sa.calculateComplexity(project),
		Size:          len(project.Epics),
		Urgency:       sa.calculateUrgency(project),
		ResourceNeeds: sa.calculateResourceNeeds(project),
	}
}

// calculateComplexity calculates project complexity
func (sa *StrategyAnalyzer) calculateComplexity(project *Project) float64 {
	// Simple complexity calculation
	totalStories := 0
	for _, epic := range project.Epics {
		totalStories += len(epic.Stories)
	}
	return float64(totalStories) / 10.0
}

// calculateUrgency calculates project urgency
func (sa *StrategyAnalyzer) calculateUrgency(project *Project) float64 {
	// Placeholder for urgency calculation
	return 0.5
}

// calculateResourceNeeds calculates resource needs
func (sa *StrategyAnalyzer) calculateResourceNeeds(project *Project) float64 {
	// Placeholder for resource needs calculation
	return 0.5
}

// ProjectCharacteristics represents project characteristics
type ProjectCharacteristics struct {
	Complexity    float64
	Size          int
	Urgency       float64
	ResourceNeeds float64
}

// defaultExecutionStrategies returns default execution strategies
func defaultExecutionStrategies() map[string]ExecutionStrategy {
	return map[string]ExecutionStrategy{
		"waterfall": &WaterfallStrategy{},
		"agile":     &AgileStrategy{},
		"hybrid":    &HybridStrategy{},
		"rapid":     &RapidStrategy{},
	}
}

// WaterfallStrategy implements waterfall execution
type WaterfallStrategy struct{}

func (ws *WaterfallStrategy) Name() string { return "waterfall" }

func (ws *WaterfallStrategy) ScoreForProject(c ProjectCharacteristics) float64 {
	// Waterfall is good for well-defined, low-complexity projects
	if c.Complexity < 0.3 {
		return 0.8
	}
	return 0.3
}

func (ws *WaterfallStrategy) Apply(project *Project) *Project {
	// Sequential execution of all phases
	return project
}

// AgileStrategy implements agile execution
type AgileStrategy struct{}

func (as *AgileStrategy) Name() string { return "agile" }

func (as *AgileStrategy) ScoreForProject(c ProjectCharacteristics) float64 {
	// Agile is good for complex, iterative projects
	if c.Complexity > 0.5 {
		return 0.9
	}
	return 0.5
}

func (as *AgileStrategy) Apply(project *Project) *Project {
	// Iterative execution with feedback loops
	return project
}

// HybridStrategy implements hybrid execution
type HybridStrategy struct{}

func (hs *HybridStrategy) Name() string { return "hybrid" }

func (hs *HybridStrategy) ScoreForProject(c ProjectCharacteristics) float64 {
	// Hybrid is good for medium complexity
	if c.Complexity > 0.3 && c.Complexity < 0.7 {
		return 0.8
	}
	return 0.4
}

func (hs *HybridStrategy) Apply(project *Project) *Project {
	// Mix of sequential and iterative execution
	return project
}

// RapidStrategy implements rapid execution
type RapidStrategy struct{}

func (rs *RapidStrategy) Name() string { return "rapid" }

func (rs *RapidStrategy) ScoreForProject(c ProjectCharacteristics) float64 {
	// Rapid is good for urgent, small projects
	if c.Urgency > 0.7 && c.Size < 3 {
		return 0.9
	}
	return 0.2
}

func (rs *RapidStrategy) Apply(project *Project) *Project {
	// Fast, parallel execution
	return project
}
