package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
)

// Planner analyzes user prompts and creates execution plans
type Planner struct {
	orchestrator   *Orchestrator
	intentAnalyzer *IntentAnalyzer
	graphBuilder   *ExecutionGraphBuilder
	optimizer      *PlanOptimizer
	log            *logger.Logger
}

// NewPlanner creates a new execution planner
func NewPlanner() *Planner {
	return &Planner{
		intentAnalyzer: NewIntentAnalyzer(),
		graphBuilder:   NewExecutionGraphBuilder(),
		optimizer:      NewPlanOptimizer(),
		log:            logger.WithComponent("planner"),
	}
}

// SetOrchestrator sets the orchestrator reference
func (p *Planner) SetOrchestrator(o *Orchestrator) {
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
func (gb *ExecutionGraphBuilder) BuildGraph(intent *Intent, orchestrator *Orchestrator) (*ExecutionGraph, error) {
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
		agent, err := orchestrator.GetAgent(taskTemplate.Agent)
		if err != nil {
			gb.log.Warn("Agent not found, skipping", "agent", taskTemplate.Agent)
			continue
		}

		node := &GraphNode{
			ID:           generateID(),
			Agent:        taskTemplate.Agent,
			AgentRef:     agent,
			Priority:     taskTemplate.Priority,
			Dependencies: taskTemplate.Dependencies,
			Request: AgentRequest{
				Prompt:  gb.buildPromptForTask(&taskTemplate, intent),
				Context: make(map[string]interface{}),
			},
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
		ID:      generateID(),
		Context: context,
		Tasks:   make([]Task, 0),
		Stages:  make([]Stage, 0),
	}

	// Topological sort to determine execution order
	stages := po.topologicalSort(graph)

	// Create stages
	for i, nodeIDs := range stages {
		stage := Stage{
			ID:    fmt.Sprintf("stage-%d", i),
			Tasks: make([]string, 0),
		}

		// Create tasks for this stage
		for _, nodeID := range nodeIDs {
			node := graph.Nodes[nodeID]
			task := Task{
				ID:           nodeID,
				Agent:        node.Agent,
				Request:      node.Request,
				Priority:     node.Priority,
				Dependencies: node.Dependencies,
				Stage:        stage.ID,
			}

			plan.Tasks = append(plan.Tasks, task)
			stage.Tasks = append(stage.Tasks, task.ID)
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
