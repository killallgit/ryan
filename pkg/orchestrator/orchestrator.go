package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/llms"
)

// Orchestrator manages routing and coordination between specialized agents
type Orchestrator struct {
	llm           llms.Model
	registry      *AgentRegistry
	stateManager  *StateManager
	feedbackLoop  *FeedbackLoop
	maxIterations int
}

// New creates a new orchestrator instance
func New(llm llms.Model, options ...Option) (*Orchestrator, error) {
	o := &Orchestrator{
		llm:           llm,
		maxIterations: 10,
	}

	// Apply options
	for _, opt := range options {
		if err := opt(o); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Initialize components
	o.registry = NewAgentRegistry()
	o.stateManager = NewStateManager()
	o.feedbackLoop = NewFeedbackLoop(o)

	logger.Info("Orchestrator initialized with %d max iterations", o.maxIterations)
	return o, nil
}

// Option is a functional option for configuring the orchestrator
type Option func(*Orchestrator) error

// WithMaxIterations sets the maximum number of iterations for task execution
func WithMaxIterations(max int) Option {
	return func(o *Orchestrator) error {
		if max <= 0 {
			return fmt.Errorf("max iterations must be positive")
		}
		o.maxIterations = max
		return nil
	}
}

// Execute processes a user query through the orchestrator
func (o *Orchestrator) Execute(ctx context.Context, query string) (*TaskResult, error) {
	logger.Debug("Orchestrator executing query: %s", query)

	// Create initial task state
	state := o.stateManager.CreateState(query)

	// Analyze intent
	intent, err := o.AnalyzeIntent(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze intent: %w", err)
	}
	state.Intent = intent

	logger.Info("Analyzed intent: type=%s, confidence=%.2f", intent.Type, intent.Confidence)

	// Execute with feedback loop
	return o.feedbackLoop.Run(ctx, state)
}

// AnalyzeIntent determines the task type and required capabilities
func (o *Orchestrator) AnalyzeIntent(ctx context.Context, query string) (*TaskIntent, error) {
	logger.Debug("🧠 Analyzing intent for query: %s", query)

	prompt := fmt.Sprintf(`Analyze the following user query and determine the task type and required capabilities.
Respond ONLY with valid JSON, no other text or formatting.

JSON structure required:
{
  "type": "one of: tool_use, code_generation, reasoning, search, planning",
  "confidence": 0.0 to 1.0,
  "required_capabilities": ["list", "of", "required", "capabilities"],
  "reasoning": "brief explanation of your analysis"
}

User Query: %s`, query)

	logger.Debug("📝 Sending intent analysis prompt to LLM")
	response, err := o.llm.Call(ctx, prompt)
	if err != nil {
		logger.Error("❌ LLM call failed for intent analysis: %v", err)
		return nil, fmt.Errorf("failed to call LLM for intent analysis: %w", err)
	}

	logger.Debug("📋 LLM response received: %s", response)

	var result struct {
		Type                 string   `json:"type"`
		Confidence           float64  `json:"confidence"`
		RequiredCapabilities []string `json:"required_capabilities"`
		Reasoning            string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		logger.Warn("⚠️ Failed to parse JSON response, falling back to defaults: %v", err)
		logger.Debug("📄 Raw response was: %s", response)
		// Fallback to basic intent
		fallbackIntent := &TaskIntent{
			Type:                 "reasoning",
			Confidence:           0.5,
			RequiredCapabilities: []string{"general"},
		}
		logger.Info("🎯 Intent analysis fallback: %+v", fallbackIntent)
		return fallbackIntent, nil
	}

	intent := &TaskIntent{
		Type:                 result.Type,
		Confidence:           result.Confidence,
		RequiredCapabilities: result.RequiredCapabilities,
	}

	logger.Info("🎯 Intent analyzed: type=%s, confidence=%.2f, capabilities=%v",
		intent.Type, intent.Confidence, intent.RequiredCapabilities)
	if result.Reasoning != "" {
		logger.Debug("💭 Reasoning: %s", result.Reasoning)
	}

	return intent, nil
}

// Route determines which agent should handle the task
func (o *Orchestrator) Route(ctx context.Context, intent *TaskIntent, state *TaskState) (*RouteDecision, error) {
	logger.Debug("🎯 Routing task based on intent: %s", intent.Type)

	// Determine best agent based on intent
	agentType := o.selectAgentForIntent(intent)
	logger.Info("🤖 Selected agent: %s for intent type: %s", agentType, intent.Type)

	// Check if agent is available
	if !o.registry.HasAgent(agentType) {
		logger.Error("❌ Agent %s is not available", agentType)
		return nil, fmt.Errorf("no agent available for type: %s", agentType)
	}
	logger.Debug("✅ Agent %s is available", agentType)

	// Create routing decision
	decision := &RouteDecision{
		TargetAgent: agentType,
		Instruction: state.Query,
		Parameters:  make(map[string]interface{}),
	}

	// Add specific parameters based on agent type
	switch agentType {
	case AgentToolCaller:
		decision.Tools = o.getAvailableTools()
		decision.ExpectedOutput = OutputFormatJSON
	case AgentCodeGen:
		decision.ExpectedOutput = OutputFormatCode
	case AgentReasoner:
		decision.ExpectedOutput = OutputFormatText
	case AgentSearcher:
		decision.ExpectedOutput = OutputFormatList
	}

	logger.Info("📋 Routing decision created: agent=%s, output_format=%s",
		decision.TargetAgent, decision.ExpectedOutput)
	if len(decision.Tools) > 0 {
		logger.Debug("🔧 Available tools: %v", decision.Tools)
	}

	return decision, nil
}

// ProcessFeedback handles responses from agents and determines next steps
func (o *Orchestrator) ProcessFeedback(ctx context.Context, feedback *AgentResponse, state *TaskState) (*NextStep, error) {
	logger.Debug("🔄 Processing feedback from agent: %s", feedback.AgentType)
	logger.Debug("📊 Agent response status: %s", feedback.Status)
	if feedback.Error != nil {
		logger.Debug("⚠️ Agent reported error: %s", *feedback.Error)
	}
	if len(feedback.ToolsCalled) > 0 {
		logger.Debug("🔧 Agent used %d tools", len(feedback.ToolsCalled))
	}

	// Update state with feedback
	state.History = append(state.History, *feedback)
	logger.Debug("📚 State history now has %d entries", len(state.History))

	// Check if task is complete
	if feedback.Status == "success" && feedback.NextAction == nil {
		logger.Info("✅ Task completed successfully")
		state.Status = StatusCompleted
		return &NextStep{
			Action:   ActionComplete,
			Decision: nil,
		}, nil
	}

	// Check for errors
	if feedback.Status == "failed" {
		logger.Warn("❌ Agent failed, attempting retry")
		state.Status = StatusFailed
		retryDecision := o.createRetryDecision(feedback, state)
		logger.Debug("🔄 Created retry decision for agent: %s", retryDecision.TargetAgent)
		return &NextStep{
			Action:   ActionRetry,
			Decision: retryDecision,
		}, nil
	}

	// Process next action if specified
	if feedback.NextAction != nil {
		logger.Info("➡️ Agent suggested next action: %s", feedback.NextAction.TargetAgent)
		return &NextStep{
			Action:   ActionContinue,
			Decision: feedback.NextAction,
		}, nil
	}

	// Default: mark as complete
	logger.Info("✅ Task completed (default completion)")
	state.Status = StatusCompleted
	return &NextStep{
		Action:   ActionComplete,
		Decision: nil,
	}, nil
}

// selectAgentForIntent determines the best agent type for a given intent
func (o *Orchestrator) selectAgentForIntent(intent *TaskIntent) AgentType {
	switch intent.Type {
	case "tool_use":
		return AgentToolCaller
	case "code_generation":
		return AgentCodeGen
	case "reasoning":
		return AgentReasoner
	case "search":
		return AgentSearcher
	case "planning":
		return AgentPlanner
	default:
		// Default to reasoner for unknown types
		return AgentReasoner
	}
}

// getAvailableTools returns the list of available tools
func (o *Orchestrator) getAvailableTools() []string {
	// This will be populated from config/registry
	return []string{"bash", "file_read", "file_write", "git", "search", "web"}
}

// createRetryDecision creates a retry decision for failed tasks
func (o *Orchestrator) createRetryDecision(feedback *AgentResponse, state *TaskState) *RouteDecision {
	// Simple retry with same agent for now
	// In future, could route to different agent or modify parameters
	return &RouteDecision{
		TargetAgent: feedback.AgentType,
		Instruction: fmt.Sprintf("Previous attempt failed. Please retry: %s", state.Query),
		Parameters: map[string]interface{}{
			"retry_count": len(state.History),
		},
	}
}

// RegisterAgent adds an agent to the registry
func (o *Orchestrator) RegisterAgent(agentType AgentType, agent Agent) error {
	return o.registry.Register(agentType, agent)
}
