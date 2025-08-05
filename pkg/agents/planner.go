package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/agents/interfaces"
	"github.com/killallgit/ryan/pkg/logger"
)

// Planner analyzes user prompts and creates execution plans
type Planner struct {
	orchestrator        interfaces.OrchestratorInterface
	intentAnalyzer      *IntentAnalyzer
	graphBuilder        *ExecutionGraphBuilder
	optimizer           *PlanOptimizer
	hierarchicalPlanner *HierarchicalPlanner
	log                 *logger.Logger
}

// NewPlanner creates a new execution planner
func NewPlanner() *Planner {
	return &Planner{
		intentAnalyzer:      NewIntentAnalyzer(),
		graphBuilder:        NewExecutionGraphBuilder(),
		optimizer:           NewPlanOptimizer(),
		hierarchicalPlanner: NewHierarchicalPlanner(),
		log:                 logger.WithComponent("planner"),
	}
}

// SetOrchestrator sets the orchestrator reference
func (p *Planner) SetOrchestrator(o interfaces.OrchestratorInterface) {
	p.orchestrator = o
}

// CreateExecutionPlan analyzes the request and creates an optimized execution plan
func (p *Planner) CreateExecutionPlan(ctx context.Context, request string, execContext *ExecutionContext) (*ExecutionPlan, error) {
	p.log.Info("Creating execution plan", "request_preview", truncateString(request, 100))

	// Validate request
	if strings.TrimSpace(request) == "" {
		return nil, fmt.Errorf("empty request")
	}

	// Validate orchestrator is set
	if p.orchestrator == nil {
		return nil, fmt.Errorf("orchestrator not set")
	}

	// Analyze intent
	intent, err := p.intentAnalyzer.Analyze(request)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze intent: %w", err)
	}

	p.log.Debug("Analyzed intent", "primary", intent.Primary, "secondary", intent.Secondary)

	// Build initial execution graph
	graph, err := p.graphBuilder.BuildGraph(intent, p.orchestrator)
	if err != nil {
		return nil, fmt.Errorf("failed to build execution graph: %w", err)
	}

	// Optimize the plan
	plan := p.optimizer.Optimize(graph, execContext)

	p.log.Info("Created execution plan",
		"tasks", len(plan.Tasks),
		"stages", len(plan.Stages),
		"estimated_duration", plan.EstimatedDuration)

	return plan, nil
}

// IntentAnalyzer analyzes user prompts to determine intent
type IntentAnalyzer struct {
	patterns []IntentPattern
	log      *logger.Logger
}

// NewIntentAnalyzer creates a new intent analyzer
func NewIntentAnalyzer() *IntentAnalyzer {
	return &IntentAnalyzer{
		patterns: defaultIntentPatterns(),
		log:      logger.WithComponent("intent_analyzer"),
	}
}

// Analyze extracts intent from a user prompt
func (ia *IntentAnalyzer) Analyze(prompt string) (*Intent, error) {
	lowerPrompt := strings.ToLower(prompt)

	intent := &Intent{
		RawPrompt: prompt,
		Entities:  make(map[string]string),
	}

	// Match against patterns
	for _, pattern := range ia.patterns {
		if pattern.Matches(lowerPrompt) {
			intent.Primary = pattern.IntentType
			intent.Confidence = pattern.GetConfidence(lowerPrompt)

			// Extract entities
			entities := pattern.ExtractEntities(prompt)
			for k, v := range entities {
				intent.Entities[k] = v
			}

			break
		}
	}

	// If no primary intent found, use generic
	if intent.Primary == "" {
		intent.Primary = IntentGeneric
		intent.Confidence = 0.5
	}

	// Look for secondary intents
	intent.Secondary = ia.findSecondaryIntents(lowerPrompt)

	return intent, nil
}

// findSecondaryIntents identifies additional intents in the prompt
func (ia *IntentAnalyzer) findSecondaryIntents(prompt string) []string {
	var secondary []string

	// Check for common secondary patterns
	if strings.Contains(prompt, "and test") || strings.Contains(prompt, "with tests") {
		secondary = append(secondary, "test")
	}
	if strings.Contains(prompt, "document") || strings.Contains(prompt, "with docs") {
		secondary = append(secondary, "document")
	}
	if strings.Contains(prompt, "optimize") || strings.Contains(prompt, "performance") {
		secondary = append(secondary, "optimize")
	}
	if strings.Contains(prompt, "fix") || strings.Contains(prompt, "repair") || strings.Contains(prompt, "and correct") {
		secondary = append(secondary, "fix")
	}
	if strings.Contains(prompt, "and analyze") || strings.Contains(prompt, "analyze") {
		secondary = append(secondary, "analyze")
	}

	return secondary
}

// ExecutionGraphBuilder builds execution graphs from intents
type ExecutionGraphBuilder struct {
	templates map[string]*PlanTemplate
	log       *logger.Logger
}

// NewExecutionGraphBuilder creates a new graph builder
func NewExecutionGraphBuilder() *ExecutionGraphBuilder {
	return &ExecutionGraphBuilder{
		templates: defaultPlanTemplates(),
		log:       logger.WithComponent("graph_builder"),
	}
}

// BuildGraph creates an execution graph from an intent
func (gb *ExecutionGraphBuilder) BuildGraph(intent *Intent, orchestrator OrchestratorInterface) (*ExecutionGraph, error) {
	// Find matching template
	template, exists := gb.templates[string(intent.Primary)]
	if !exists {
		return nil, fmt.Errorf("no template for intent: %s", intent.Primary)
	}

	// Create graph from template
	graph := &ExecutionGraph{
		Nodes: make(map[string]*GraphNode),
		Edges: make(map[string][]string),
	}

	// Build nodes
	for _, taskTemplate := range template.Tasks {
		_, err := orchestrator.GetAgent(taskTemplate.Agent)
		if err != nil {
			gb.log.Warn("Agent not found, skipping", "agent", taskTemplate.Agent)
			continue
		}

		task := &Task{
			ID:           generateID(),
			Agent:        taskTemplate.Agent,
			Priority:     taskTemplate.Priority,
			Dependencies: taskTemplate.Dependencies,
			Request: AgentRequest{
				Prompt:  gb.buildPromptForTask(&taskTemplate, intent),
				Context: make(map[string]interface{}),
			},
		}

		node := &GraphNode{
			ID:           task.ID,
			Task:         task,
			Status:       "pending",
			Dependencies: taskTemplate.Dependencies,
			Dependents:   []string{},
		}

		graph.Nodes[node.ID] = node

		// Build edges based on dependencies
		for _, dep := range taskTemplate.Dependencies {
			graph.Edges[dep] = append(graph.Edges[dep], node.ID)
		}
	}

	return graph, nil
}

// buildPromptForTask creates a specific prompt for a task based on intent
func (gb *ExecutionGraphBuilder) buildPromptForTask(task *TaskTemplate, intent *Intent) string {
	prompt := task.PromptTemplate

	// Replace raw_prompt placeholder
	prompt = strings.ReplaceAll(prompt, "{raw_prompt}", intent.RawPrompt)

	// Replace placeholders with entities
	for key, value := range intent.Entities {
		placeholder := fmt.Sprintf("{%s}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	return prompt
}

// PlanOptimizer optimizes execution plans
type PlanOptimizer struct {
	log *logger.Logger
}

// NewPlanOptimizer creates a new plan optimizer
func NewPlanOptimizer() *PlanOptimizer {
	return &PlanOptimizer{
		log: logger.WithComponent("plan_optimizer"),
	}
}

// Optimize converts a graph into an optimized execution plan
func (po *PlanOptimizer) Optimize(graph *ExecutionGraph, context *ExecutionContext) *ExecutionPlan {
	plan := &ExecutionPlan{
		ID:     generateID(),
		Tasks:  make([]Task, 0),
		Stages: make([]Stage, 0),
		Metadata: map[string]interface{}{
			"execution_context": context,
		},
	}

	// Topological sort to determine execution order
	stages := po.topologicalSort(graph)

	// Create stages
	for i, nodeIDs := range stages {
		stage := Stage{
			ID:    fmt.Sprintf("stage-%d", i),
			Tasks: make([]Task, 0),
		}

		// Add tasks for this stage
		for _, nodeID := range nodeIDs {
			node := graph.Nodes[nodeID]
			if node.Task != nil {
				plan.Tasks = append(plan.Tasks, *node.Task)
				stage.Tasks = append(stage.Tasks, *node.Task)
			}
		}

		plan.Stages = append(plan.Stages, stage)
	}

	// Estimate duration
	plan.EstimatedDuration = po.estimateDuration(plan)

	return plan
}

// topologicalSort performs topological sorting on the graph
func (po *PlanOptimizer) topologicalSort(graph *ExecutionGraph) [][]string {
	// Simple level-based topological sort
	levels := make([][]string, 0)
	visited := make(map[string]bool)

	// Find nodes with no dependencies
	for {
		level := make([]string, 0)

		for nodeID, node := range graph.Nodes {
			if visited[nodeID] {
				continue
			}

			// Check if all dependencies are satisfied
			canExecute := true
			for _, dep := range node.Dependencies {
				if !visited[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				level = append(level, nodeID)
			}
		}

		if len(level) == 0 {
			break
		}

		// Mark as visited
		for _, nodeID := range level {
			visited[nodeID] = true
		}

		levels = append(levels, level)
	}

	return levels
}

// estimateDuration estimates the total duration of a plan
func (po *PlanOptimizer) estimateDuration(plan *ExecutionPlan) string {
	// Simple estimation: 2 seconds per task + 1 second per stage
	totalSeconds := len(plan.Tasks)*2 + len(plan.Stages)
	return fmt.Sprintf("%ds", totalSeconds)
}

// Types for intent analysis

type IntentType string

const (
	IntentCodeReview    IntentType = "code_review"
	IntentFileOperation IntentType = "file_operation"
	IntentSearch        IntentType = "search"
	IntentAnalysis      IntentType = "analysis"
	IntentRefactor      IntentType = "refactor"
	IntentTest          IntentType = "test"
	IntentGeneric       IntentType = "generic"
)

type Intent struct {
	Primary    IntentType
	Secondary  []string
	RawPrompt  string
	Entities   map[string]string
	Confidence float64
}

type IntentPattern struct {
	Pattern    string
	IntentType IntentType
	Extract    []string
}

func (ip *IntentPattern) Matches(prompt string) bool {
	return strings.Contains(prompt, ip.Pattern)
}

func (ip *IntentPattern) GetConfidence(prompt string) float64 {
	// Simple confidence based on pattern match
	if strings.Contains(prompt, ip.Pattern) {
		return 0.8
	}
	return 0.0
}

func (ip *IntentPattern) ExtractEntities(prompt string) map[string]string {
	entities := make(map[string]string)

	// Simple entity extraction - would be more sophisticated in practice
	for _, extract := range ip.Extract {
		switch extract {
		case "path":
			// Extract file paths
			words := strings.Fields(prompt)
			for _, word := range words {
				if strings.Contains(word, "/") || strings.HasSuffix(word, ".go") {
					entities["path"] = strings.Trim(word, "\"',.")
					break
				}
			}
		case "target":
			// Extract target for operations
			if idx := strings.Index(prompt, " of "); idx != -1 {
				target := prompt[idx+4:]
				if endIdx := strings.IndexAny(target, " ,;"); endIdx != -1 {
					target = target[:endIdx]
				}
				entities["target"] = strings.TrimSpace(target)
			}
		}
	}

	return entities
}

// Default patterns and templates

func defaultIntentPatterns() []IntentPattern {
	return []IntentPattern{
		{Pattern: "code review", IntentType: IntentCodeReview, Extract: []string{"target", "path"}},
		{Pattern: "review", IntentType: IntentCodeReview, Extract: []string{"target", "path"}},
		{Pattern: "analyze", IntentType: IntentAnalysis, Extract: []string{"target", "path"}},
		{Pattern: "create file", IntentType: IntentFileOperation, Extract: []string{"path"}},
		{Pattern: "read file", IntentType: IntentFileOperation, Extract: []string{"path"}},
		{Pattern: "search for", IntentType: IntentSearch, Extract: []string{"target"}},
		{Pattern: "find", IntentType: IntentSearch, Extract: []string{"target"}},
		{Pattern: "refactor", IntentType: IntentRefactor, Extract: []string{"target", "path"}},
		{Pattern: "test", IntentType: IntentTest, Extract: []string{"target", "path"}},
	}
}

type TaskTemplate struct {
	Agent          string
	PromptTemplate string
	Priority       int
	Dependencies   []string
}

type PlanTemplate struct {
	Intent IntentType
	Tasks  []TaskTemplate
}

func defaultPlanTemplates() map[string]*PlanTemplate {
	return map[string]*PlanTemplate{
		string(IntentCodeReview): {
			Intent: IntentCodeReview,
			Tasks: []TaskTemplate{
				{
					Agent:          "file_operations",
					PromptTemplate: "List and read all files in {target}",
					Priority:       1,
					Dependencies:   []string{},
				},
				{
					Agent:          "code_analysis",
					PromptTemplate: "Analyze code structure and patterns in {target}",
					Priority:       2,
					Dependencies:   []string{},
				},
				{
					Agent:          "code_review",
					PromptTemplate: "Perform comprehensive code review of {target}",
					Priority:       3,
					Dependencies:   []string{},
				},
			},
		},
		string(IntentFileOperation): {
			Intent: IntentFileOperation,
			Tasks: []TaskTemplate{
				{
					Agent:          "file_operations",
					PromptTemplate: "{raw_prompt}",
					Priority:       1,
					Dependencies:   []string{},
				},
			},
		},
		string(IntentSearch): {
			Intent: IntentSearch,
			Tasks: []TaskTemplate{
				{
					Agent:          "search",
					PromptTemplate: "{raw_prompt}",
					Priority:       1,
					Dependencies:   []string{},
				},
			},
		},
		string(IntentAnalysis): {
			Intent: IntentAnalysis,
			Tasks: []TaskTemplate{
				{
					Agent:          "code_analysis",
					PromptTemplate: "{raw_prompt}",
					Priority:       1,
					Dependencies:   []string{},
				},
			},
		},
		string(IntentTest): {
			Intent: IntentTest,
			Tasks: []TaskTemplate{
				{
					Agent:          "dispatcher",
					PromptTemplate: "{raw_prompt}",
					Priority:       1,
					Dependencies:   []string{},
				},
			},
		},
		string(IntentRefactor): {
			Intent: IntentRefactor,
			Tasks: []TaskTemplate{
				{
					Agent:          "code_analysis",
					PromptTemplate: "Analyze code for refactoring: {raw_prompt}",
					Priority:       1,
					Dependencies:   []string{},
				},
				{
					Agent:          "file_operations",
					PromptTemplate: "Apply refactoring changes",
					Priority:       2,
					Dependencies:   []string{},
				},
			},
		},
		string(IntentGeneric): {
			Intent: IntentGeneric,
			Tasks: []TaskTemplate{
				{
					Agent:          "dispatcher",
					PromptTemplate: "{raw_prompt}",
					Priority:       1,
					Dependencies:   []string{},
				},
			},
		},
	}
}

// HierarchicalPlanner handles multi-level planning for complex projects
type HierarchicalPlanner struct {
	projectAnalyzer *ProjectAnalyzer
	epicPlanner     *EpicPlanner
	sprintPlanner   *SprintPlanner
	taskDecomposer  *TaskDecomposer
	log             *logger.Logger
}

// NewHierarchicalPlanner creates a new hierarchical planner
func NewHierarchicalPlanner() *HierarchicalPlanner {
	return &HierarchicalPlanner{
		projectAnalyzer: NewProjectAnalyzer(),
		epicPlanner:     NewEpicPlanner(),
		sprintPlanner:   NewSprintPlanner(),
		taskDecomposer:  NewTaskDecomposer(),
		log:             logger.WithComponent("hierarchical_planner"),
	}
}

// PlanProject creates a hierarchical plan for a project
func (hp *HierarchicalPlanner) PlanProject(ctx context.Context, request string, context *ProjectContext) (*Project, error) {
	hp.log.Info("Creating hierarchical project plan", "request_preview", truncateString(request, 100))

	// Analyze project requirements
	analysis := hp.projectAnalyzer.Analyze(request)

	// Create project structure
	project := &Project{
		ID:          generateID(),
		Name:        analysis.ProjectName,
		Description: request,
		Context:     context,
		Status:      ProjectStatusPlanning,
		CreatedAt:   time.Now(),
	}

	// Plan epics based on analysis
	epics, err := hp.epicPlanner.PlanEpics(analysis, context)
	if err != nil {
		return nil, fmt.Errorf("failed to plan epics: %w", err)
	}
	project.Epics = epics

	// Plan sprints from epics
	sprints, err := hp.sprintPlanner.PlanSprints(epics, analysis, context)
	if err != nil {
		return nil, fmt.Errorf("failed to plan sprints: %w", err)
	}
	project.Sprints = sprints

	hp.log.Info("Created hierarchical project plan",
		"project_id", project.ID,
		"epics", len(epics),
		"sprints", len(sprints))

	return project, nil
}

// ProjectAnalyzer analyzes requests to extract project information
type ProjectAnalyzer struct {
	log *logger.Logger
}

// NewProjectAnalyzer creates a new project analyzer
func NewProjectAnalyzer() *ProjectAnalyzer {
	return &ProjectAnalyzer{
		log: logger.WithComponent("project_analyzer"),
	}
}

// Analyze analyzes a request to extract project information
func (pa *ProjectAnalyzer) Analyze(request string) *ProjectAnalysis {
	analysis := &ProjectAnalysis{
		RawRequest:   request,
		ProjectName:  pa.extractProjectName(request),
		Scope:        pa.analyzeScope(request),
		Complexity:   pa.analyzeComplexity(request),
		Components:   pa.extractComponents(request),
		Requirements: pa.extractRequirements(request),
	}

	return analysis
}

func (pa *ProjectAnalyzer) extractProjectName(request string) string {
	// Extract project name from request
	words := strings.Fields(request)
	if len(words) > 3 {
		return strings.Join(words[:3], " ")
	}
	return "Project"
}

func (pa *ProjectAnalyzer) analyzeScope(request string) ProjectScope {
	// Analyze project scope
	wordCount := len(strings.Fields(request))
	if wordCount > 100 {
		return ProjectScopeLarge
	} else if wordCount > 50 {
		return ProjectScopeMedium
	}
	return ProjectScopeSmall
}

func (pa *ProjectAnalyzer) analyzeComplexity(request string) float64 {
	// Simple complexity analysis based on keywords
	complexity := 0.3 // Base complexity

	complexityKeywords := []string{"integrate", "multiple", "complex", "distributed", "architecture", "system"}
	for _, keyword := range complexityKeywords {
		if strings.Contains(strings.ToLower(request), keyword) {
			complexity += 0.1
		}
	}

	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

func (pa *ProjectAnalyzer) extractComponents(request string) []string {
	// Extract major components from request
	components := []string{}

	// Look for common component patterns
	componentKeywords := []string{"frontend", "backend", "database", "api", "service", "module", "component"}
	lowerRequest := strings.ToLower(request)

	for _, keyword := range componentKeywords {
		if strings.Contains(lowerRequest, keyword) {
			components = append(components, keyword)
		}
	}

	return components
}

func (pa *ProjectAnalyzer) extractRequirements(request string) []string {
	// Extract requirements from request
	requirements := []string{}

	// Look for requirement patterns
	if strings.Contains(strings.ToLower(request), "test") {
		requirements = append(requirements, "testing")
	}
	if strings.Contains(strings.ToLower(request), "document") {
		requirements = append(requirements, "documentation")
	}
	if strings.Contains(strings.ToLower(request), "secure") || strings.Contains(strings.ToLower(request), "security") {
		requirements = append(requirements, "security")
	}

	return requirements
}

// ProjectAnalysis contains the analysis of a project request
type ProjectAnalysis struct {
	RawRequest   string
	ProjectName  string
	Scope        ProjectScope
	Complexity   float64
	Components   []string
	Requirements []string
}

// ProjectScope represents the scope of a project
type ProjectScope string

const (
	ProjectScopeSmall  ProjectScope = "small"
	ProjectScopeMedium ProjectScope = "medium"
	ProjectScopeLarge  ProjectScope = "large"
)

// EpicPlanner plans epics for a project
type EpicPlanner struct {
	log *logger.Logger
}

// NewEpicPlanner creates a new epic planner
func NewEpicPlanner() *EpicPlanner {
	return &EpicPlanner{
		log: logger.WithComponent("epic_planner"),
	}
}

// PlanEpics creates epics based on project analysis
func (ep *EpicPlanner) PlanEpics(analysis *ProjectAnalysis, context *ProjectContext) ([]*Epic, error) {
	epics := []*Epic{}

	// Create epics for each major component
	for i, component := range analysis.Components {
		epic := &Epic{
			ID:          generateID(),
			Title:       fmt.Sprintf("Implement %s", component),
			Description: fmt.Sprintf("Implementation of %s component", component),
			Priority:    Priority(10 - i), // Prioritize in order
			Status:      EpicStatusTodo,
			Stories:     ep.createStoriesForComponent(component),
		}
		epics = append(epics, epic)
	}

	// Add epics for requirements
	for _, requirement := range analysis.Requirements {
		epic := &Epic{
			ID:          generateID(),
			Title:       fmt.Sprintf("Ensure %s", requirement),
			Description: fmt.Sprintf("Implementation of %s requirements", requirement),
			Priority:    PriorityMedium,
			Status:      EpicStatusTodo,
			Stories:     ep.createStoriesForRequirement(requirement),
		}
		epics = append(epics, epic)
	}

	// If no epics created, create a default one
	if len(epics) == 0 {
		epics = append(epics, &Epic{
			ID:          generateID(),
			Title:       "Core Implementation",
			Description: analysis.RawRequest,
			Priority:    PriorityHigh,
			Status:      EpicStatusTodo,
			Stories:     ep.createDefaultStories(),
		})
	}

	return epics, nil
}

func (ep *EpicPlanner) createStoriesForComponent(component string) []*UserStory {
	stories := []*UserStory{
		{
			ID:          generateID(),
			Title:       fmt.Sprintf("Design %s architecture", component),
			Description: fmt.Sprintf("Design and plan the %s component architecture", component),
			Points:      5,
			Priority:    PriorityHigh,
			Status:      StoryStatusTodo,
		},
		{
			ID:          generateID(),
			Title:       fmt.Sprintf("Implement %s core", component),
			Description: fmt.Sprintf("Implement core functionality for %s", component),
			Points:      8,
			Priority:    PriorityHigh,
			Status:      StoryStatusTodo,
		},
		{
			ID:          generateID(),
			Title:       fmt.Sprintf("Test %s", component),
			Description: fmt.Sprintf("Create tests for %s component", component),
			Points:      3,
			Priority:    PriorityMedium,
			Status:      StoryStatusTodo,
		},
	}
	return stories
}

func (ep *EpicPlanner) createStoriesForRequirement(requirement string) []*UserStory {
	stories := []*UserStory{
		{
			ID:          generateID(),
			Title:       fmt.Sprintf("Implement %s", requirement),
			Description: fmt.Sprintf("Ensure %s requirements are met", requirement),
			Points:      5,
			Priority:    PriorityMedium,
			Status:      StoryStatusTodo,
		},
	}
	return stories
}

func (ep *EpicPlanner) createDefaultStories() []*UserStory {
	return []*UserStory{
		{
			ID:          generateID(),
			Title:       "Initial implementation",
			Description: "Implement core functionality",
			Points:      8,
			Priority:    PriorityHigh,
			Status:      StoryStatusTodo,
		},
	}
}

// SprintPlanner plans sprints from epics
type SprintPlanner struct {
	log *logger.Logger
}

// NewSprintPlanner creates a new sprint planner
func NewSprintPlanner() *SprintPlanner {
	return &SprintPlanner{
		log: logger.WithComponent("sprint_planner"),
	}
}

// PlanSprints creates sprints from epics
func (sp *SprintPlanner) PlanSprints(epics []*Epic, analysis *ProjectAnalysis, context *ProjectContext) ([]*Sprint, error) {
	sprints := []*Sprint{}

	// Calculate sprint capacity based on complexity
	sprintCapacity := sp.calculateSprintCapacity(analysis.Complexity)

	// Group stories into sprints
	currentSprint := sp.createSprint(1, sprintCapacity)
	currentCapacity := 0

	for _, epic := range epics {
		for _, story := range epic.Stories {
			if currentCapacity+story.Points > sprintCapacity {
				// Finalize current sprint and start new one
				sprints = append(sprints, currentSprint)
				currentSprint = sp.createSprint(len(sprints)+1, sprintCapacity)
				currentCapacity = 0
			}

			currentSprint.Stories = append(currentSprint.Stories, story)
			currentCapacity += story.Points
		}
	}

	// Add last sprint if it has stories
	if len(currentSprint.Stories) > 0 {
		sprints = append(sprints, currentSprint)
	}

	return sprints, nil
}

func (sp *SprintPlanner) calculateSprintCapacity(complexity float64) int {
	// Base capacity adjusted by complexity
	baseCapacity := 40
	adjustedCapacity := float64(baseCapacity) * (1.0 - complexity*0.3)
	return int(adjustedCapacity)
}

func (sp *SprintPlanner) createSprint(number int, capacity int) *Sprint {
	return &Sprint{
		ID:        generateID(),
		Number:    number,
		Name:      fmt.Sprintf("Sprint %d", number),
		Goal:      fmt.Sprintf("Complete sprint %d objectives", number),
		StartDate: time.Now().Add(time.Duration(number-1) * 14 * 24 * time.Hour), // 2-week sprints
		EndDate:   time.Now().Add(time.Duration(number) * 14 * 24 * time.Hour),
		Status:    SprintStatusPlanning,
		Stories:   []*UserStory{},
		Plans:     []*ExecutionPlan{},
		Capacity:  capacity,
	}
}

// TaskDecomposer decomposes stories into executable tasks
type TaskDecomposer struct {
	log *logger.Logger
}

// NewTaskDecomposer creates a new task decomposer
func NewTaskDecomposer() *TaskDecomposer {
	return &TaskDecomposer{
		log: logger.WithComponent("task_decomposer"),
	}
}

// DecomposeStory breaks down a user story into tasks
func (td *TaskDecomposer) DecomposeStory(story *UserStory) []Task {
	tasks := []Task{}

	// Create tasks based on story type
	if strings.Contains(strings.ToLower(story.Title), "design") {
		tasks = append(tasks, td.createDesignTask(story))
	}
	if strings.Contains(strings.ToLower(story.Title), "implement") {
		tasks = append(tasks, td.createImplementationTask(story))
	}
	if strings.Contains(strings.ToLower(story.Title), "test") {
		tasks = append(tasks, td.createTestTask(story))
	}

	// Default task if none created
	if len(tasks) == 0 {
		tasks = append(tasks, td.createDefaultTask(story))
	}

	return tasks
}

func (td *TaskDecomposer) createDesignTask(story *UserStory) Task {
	return Task{
		ID:    generateID(),
		Agent: "code_analysis",
		Request: AgentRequest{
			Prompt: fmt.Sprintf("Design and analyze architecture for: %s", story.Description),
		},
		Priority: int(story.Priority),
		Timeout:  30 * time.Minute,
	}
}

func (td *TaskDecomposer) createImplementationTask(story *UserStory) Task {
	return Task{
		ID:    generateID(),
		Agent: "dispatcher",
		Request: AgentRequest{
			Prompt: story.Description,
		},
		Priority: int(story.Priority),
		Timeout:  60 * time.Minute,
	}
}

func (td *TaskDecomposer) createTestTask(story *UserStory) Task {
	return Task{
		ID:    generateID(),
		Agent: "dispatcher",
		Request: AgentRequest{
			Prompt: fmt.Sprintf("Create comprehensive tests for: %s", story.Description),
		},
		Priority: int(story.Priority),
		Timeout:  30 * time.Minute,
	}
}

func (td *TaskDecomposer) createDefaultTask(story *UserStory) Task {
	return Task{
		ID:    generateID(),
		Agent: "dispatcher",
		Request: AgentRequest{
			Prompt: story.Description,
		},
		Priority: int(story.Priority),
		Timeout:  45 * time.Minute,
	}
}

// Additional methods for the main Planner to support hierarchical planning

// CreateProjectPlan creates a hierarchical project plan for complex requests
func (p *Planner) CreateProjectPlan(ctx context.Context, request string, projectContext *ProjectContext) (*Project, error) {
	if p.hierarchicalPlanner == nil {
		return nil, fmt.Errorf("hierarchical planner not initialized")
	}

	return p.hierarchicalPlanner.PlanProject(ctx, request, projectContext)
}

// IsProjectLevelRequest determines if a request requires project-level planning
func (p *Planner) IsProjectLevelRequest(request string) bool {
	// Check for project-level indicators
	projectIndicators := []string{
		"project", "system", "application", "feature set",
		"multiple components", "architecture", "full implementation",
	}

	lowerRequest := strings.ToLower(request)
	matchCount := 0

	for _, indicator := range projectIndicators {
		if strings.Contains(lowerRequest, indicator) {
			matchCount++
		}
	}

	// Also check complexity
	wordCount := len(strings.Fields(request))

	// More lenient criteria: 1 strong indicator or long request
	return matchCount >= 1 || wordCount > 50
}
