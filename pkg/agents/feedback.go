package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/killallgit/ryan/pkg/agents/interfaces"
	"github.com/killallgit/ryan/pkg/logger"
)

// FeedbackLoop handles feedback processing and learning
type FeedbackLoop struct {
	orchestrator interfaces.Orchestrator
	validator    *ResultValidator
	corrector    *AutoCorrector
	learner      *PatternLearner
	log          *logger.Logger
}

// NewFeedbackLoop creates a new feedback loop
func NewFeedbackLoop() *FeedbackLoop {
	return &FeedbackLoop{
		validator: NewResultValidator(),
		corrector: NewAutoCorrector(),
		learner:   NewPatternLearner(),
		log:       logger.WithComponent("feedback_loop"),
	}
}

// SetOrchestrator sets the orchestrator reference
func (fl *FeedbackLoop) SetOrchestrator(o interfaces.Orchestrator) {
	fl.orchestrator = o
}

// ProcessFeedback processes feedback from agent execution
func (fl *FeedbackLoop) ProcessFeedback(ctx context.Context, feedback *FeedbackRequest) error {
	fl.log.Info("Processing feedback", "type", feedback.Type, "task_id", feedback.TaskID)

	switch feedback.Type {
	case FeedbackTypeNeedMoreContext:
		return fl.handleNeedMoreContext(ctx, feedback)
	case FeedbackTypeValidationError:
		return fl.handleValidationError(ctx, feedback)
	case FeedbackTypeRetry:
		return fl.handleRetry(ctx, feedback)
	case FeedbackTypeRefine:
		return fl.handleRefine(ctx, feedback)
	case FeedbackTypeCorrection:
		return fl.handleCorrection(ctx, feedback)
	default:
		return fmt.Errorf("unknown feedback type: %s", feedback.Type)
	}
}

// handleNeedMoreContext handles requests for additional context
func (fl *FeedbackLoop) handleNeedMoreContext(ctx context.Context, feedback *FeedbackRequest) error {
	fl.log.Debug("Handling need more context feedback")

	// Extract what context is needed
	contextRequest, ok := feedback.Context["content"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid context request format")
	}

	// Create a new task to gather the requested context
	contextType, _ := contextRequest["type"].(string)
	target, _ := contextRequest["target"].(string)

	// Build a new execution plan for context gathering
	plan := &ExecutionPlan{
		ID: generateID(),
		Metadata: map[string]interface{}{
			"feedback_context": feedback.Context,
		},
		Tasks: []Task{
			{
				ID:    generateID(),
				Agent: "file_operations",
				Request: AgentRequest{
					Prompt: fmt.Sprintf("Gather additional context about %s for %s", target, contextType),
					Context: map[string]interface{}{
						"original_request": feedback.TaskID,
						"context_type":     contextType,
					},
				},
				Priority: 10, // High priority
			},
		},
		Stages: []Stage{
			{
				ID:    "context-gathering",
				Tasks: []Task{}, // Stage tasks are now Task objects, not IDs
			},
		},
	}

	// Execute the context gathering plan
	// Create execution context from feedback context
	execCtx := &ExecutionContext{
		RequestID:  feedback.RequestID,
		SharedData: feedback.Context,
		Options:    map[string]interface{}{},
	}
	results, err := fl.orchestrator.ExecuteWithPlan(ctx, plan, execCtx)
	if err != nil {
		return fmt.Errorf("failed to gather additional context: %w", err)
	}

	// Update the execution context with new information
	if len(results) > 0 {
		// Store the result in the feedback context
		if feedback.Context == nil {
			feedback.Context = make(map[string]interface{})
		}
		feedback.Context["additional_context"] = results[0].Result
	}

	return nil
}

// handleValidationError handles validation errors
func (fl *FeedbackLoop) handleValidationError(ctx context.Context, feedback *FeedbackRequest) error {
	fl.log.Debug("Handling validation error feedback")

	// Analyze the validation error
	validationResult := fl.validator.Analyze(feedback)

	// Attempt automatic correction
	if validationResult.CanAutoCorrect {
		correction := fl.corrector.GenerateCorrection(validationResult)
		if correction != nil {
			// Apply correction by re-executing with modified parameters
			return fl.applyCorrection(ctx, feedback, correction)
		}
	}

	// Learn from the error
	fl.learner.RecordPattern(feedback, validationResult)

	return nil
}

// handleRetry handles retry requests
func (fl *FeedbackLoop) handleRetry(ctx context.Context, feedback *FeedbackRequest) error {
	fl.log.Debug("Handling retry feedback")

	// Extract retry parameters
	retryParams, ok := feedback.Context["content"].(map[string]interface{})
	if !ok {
		retryParams = make(map[string]interface{})
	}

	// Modify the original request based on feedback
	modifiedRequest := fl.modifyRequestForRetry(feedback, retryParams)

	// Re-execute the task
	agent, err := fl.orchestrator.GetAgent(feedback.TaskID)
	if err != nil {
		return fmt.Errorf("agent not found for retry: %w", err)
	}

	result, err := agent.Execute(ctx, modifiedRequest)
	if err != nil {
		return fmt.Errorf("retry execution failed: %w", err)
	}

	// Update context with retry result
	// Store retry result in feedback context
	if feedback.Context == nil {
		feedback.Context = make(map[string]interface{})
	}
	feedback.Context[fmt.Sprintf("retry_%s_result", feedback.TaskID)] = result

	return nil
}

// handleRefine handles refinement requests
func (fl *FeedbackLoop) handleRefine(ctx context.Context, feedback *FeedbackRequest) error {
	fl.log.Debug("Handling refine feedback")

	// Extract refinement parameters
	refineParams, ok := feedback.Context["content"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid refinement parameters")
	}

	// Create a refinement plan
	plan := fl.createRefinementPlan(feedback, refineParams)

	// Execute refinement
	// Create execution context from feedback context
	execCtx := &ExecutionContext{
		RequestID:  feedback.RequestID,
		SharedData: feedback.Context,
		Options:    map[string]interface{}{},
	}
	results, err := fl.orchestrator.ExecuteWithPlan(ctx, plan, execCtx)
	if err != nil {
		return fmt.Errorf("refinement execution failed: %w", err)
	}

	// Aggregate refined results
	fl.aggregateRefinedResults(execCtx, results)

	return nil
}

// handleCorrection handles correction feedback
func (fl *FeedbackLoop) handleCorrection(ctx context.Context, feedback *FeedbackRequest) error {
	fl.log.Debug("Handling correction feedback")

	// Extract correction parameters
	correctionParams, ok := feedback.Context["content"].(map[string]interface{})
	if !ok {
		correctionParams = make(map[string]interface{})
	}

	// Apply the correction using the validator and corrector
	validationResult := fl.validator.Analyze(feedback)

	if validationResult.CanAutoCorrect {
		correction := fl.corrector.GenerateCorrection(validationResult)
		if correction != nil {
			return fl.applyCorrection(ctx, feedback, correction)
		}
	}

	// If auto-correction isn't possible, create a manual correction task
	plan := fl.createCorrectionPlan(feedback, correctionParams)

	// Execute correction plan
	execCtx := &ExecutionContext{
		RequestID:  feedback.RequestID,
		SharedData: feedback.Context,
		Options:    map[string]interface{}{},
	}

	results, err := fl.orchestrator.ExecuteWithPlan(ctx, plan, execCtx)
	if err != nil {
		return fmt.Errorf("correction execution failed: %w", err)
	}

	// Store correction results
	if feedback.Context == nil {
		feedback.Context = make(map[string]interface{})
	}
	feedback.Context[fmt.Sprintf("correction_%s_result", feedback.TaskID)] = results

	return nil
}

// Helper methods

func (fl *FeedbackLoop) modifyRequestForRetry(feedback *FeedbackRequest, params map[string]interface{}) AgentRequest {
	// Create modified request based on feedback
	return AgentRequest{
		Prompt: fmt.Sprintf("Retry: %v", feedback.Context["content"]),
		Context: map[string]interface{}{
			"retry_attempt": true,
			"original_task": feedback.TaskID,
			"retry_params":  params,
		},
	}
}

func (fl *FeedbackLoop) createRefinementPlan(feedback *FeedbackRequest, params map[string]interface{}) *ExecutionPlan {
	// Create a plan for refinement
	return &ExecutionPlan{
		ID: generateID(),
		Metadata: map[string]interface{}{
			"feedback_context": feedback.Context,
		},
		Tasks:  []Task{}, // Would be populated based on refinement needs
		Stages: []Stage{},
	}
}

func (fl *FeedbackLoop) createCorrectionPlan(feedback *FeedbackRequest, params map[string]interface{}) *ExecutionPlan {
	// Create a plan for correction
	return &ExecutionPlan{
		ID: generateID(),
		Metadata: map[string]interface{}{
			"feedback_context": feedback.Context,
			"correction_type":  "manual",
		},
		Tasks: []Task{
			{
				ID:    generateID(),
				Agent: "code_review", // Use code review agent for corrections
				Request: AgentRequest{
					Prompt: fmt.Sprintf("Apply correction for task %s: %v", feedback.TaskID, feedback.Message),
					Context: map[string]interface{}{
						"original_task":     feedback.TaskID,
						"correction_params": params,
						"feedback_message":  feedback.Message,
					},
				},
				Priority: 10, // High priority
			},
		},
		Stages: []Stage{
			{
				ID:    "correction-application",
				Tasks: []Task{}, // Stage tasks are now Task objects, not IDs
			},
		},
	}
}

func (fl *FeedbackLoop) aggregateRefinedResults(context *ExecutionContext, results []TaskResult) {
	// Note: ExecutionContext doesn't have mutex - synchronization should be handled externally

	refinedData := make(map[string]interface{})
	for _, result := range results {
		if result.Success {
			refinedData[result.Task.ID] = result.Result
		}
	}

	context.Mu.Lock()
	context.SharedData["refined_results"] = refinedData
	context.Mu.Unlock()
}

func (fl *FeedbackLoop) applyCorrection(ctx context.Context, feedback *FeedbackRequest, correction *Correction) error {
	// Apply the correction by modifying and re-executing
	return nil
}

// ResultValidator validates agent results
type ResultValidator struct {
	rules []ValidationRule
	log   *logger.Logger
}

// NewResultValidator creates a new result validator
func NewResultValidator() *ResultValidator {
	return &ResultValidator{
		rules: defaultValidationRules(),
		log:   logger.WithComponent("result_validator"),
	}
}

// Analyze analyzes a feedback request for validation issues
func (rv *ResultValidator) Analyze(feedback *FeedbackRequest) *ValidationResult {
	result := &ValidationResult{
		IsValid:        true,
		Errors:         []string{},
		Warnings:       []string{},
		CanAutoCorrect: false,
	}

	// Apply validation rules
	for _, rule := range rv.rules {
		if rule.Applies(feedback) {
			ruleResult := rule.Validate(feedback)
			if !ruleResult.IsValid {
				result.IsValid = false
				result.Errors = append(result.Errors, ruleResult.Errors...)
			}
			result.Warnings = append(result.Warnings, ruleResult.Warnings...)
			if ruleResult.CanAutoCorrect {
				result.CanAutoCorrect = true
			}
		}
	}

	return result
}

// AutoCorrector generates automatic corrections
type AutoCorrector struct {
	strategies []CorrectionStrategy
	log        *logger.Logger
}

// NewAutoCorrector creates a new auto corrector
func NewAutoCorrector() *AutoCorrector {
	return &AutoCorrector{
		strategies: defaultCorrectionStrategies(),
		log:        logger.WithComponent("auto_corrector"),
	}
}

// GenerateCorrection generates a correction for a validation result
func (ac *AutoCorrector) GenerateCorrection(result *ValidationResult) *Correction {
	for _, strategy := range ac.strategies {
		if strategy.CanHandle(result) {
			return strategy.Generate(result)
		}
	}
	return nil
}

// PatternLearner learns from execution patterns
type PatternLearner struct {
	patterns map[string]*Pattern
	log      *logger.Logger
}

// NewPatternLearner creates a new pattern learner
func NewPatternLearner() *PatternLearner {
	return &PatternLearner{
		patterns: make(map[string]*Pattern),
		log:      logger.WithComponent("pattern_learner"),
	}
}

// RecordPattern records a pattern from feedback
func (pl *PatternLearner) RecordPattern(feedback *FeedbackRequest, result *ValidationResult) {
	// Record patterns for future optimization
	patternKey := fmt.Sprintf("%s_%s", feedback.Type, feedback.TaskID)

	if pattern, exists := pl.patterns[patternKey]; exists {
		pattern.Occurrences++
		pattern.LastSeen = time.Now()
	} else {
		pl.patterns[patternKey] = &Pattern{
			Key:         patternKey,
			Type:        string(feedback.Type),
			Occurrences: 1,
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}
	}
}

// Supporting types

type ValidationResult struct {
	IsValid        bool
	Errors         []string
	Warnings       []string
	CanAutoCorrect bool
	Suggestions    []string
}

type ValidationRule interface {
	Applies(feedback *FeedbackRequest) bool
	Validate(feedback *FeedbackRequest) *ValidationResult
}

type CorrectionStrategy interface {
	CanHandle(result *ValidationResult) bool
	Generate(result *ValidationResult) *Correction
}

type Correction struct {
	Type        string
	Description string
	Actions     []CorrectionAction
}

type CorrectionAction struct {
	Type   string
	Target string
	Value  interface{}
}

type Pattern struct {
	Key         string
	Type        string
	Occurrences int
	FirstSeen   time.Time
	LastSeen    time.Time
	Data        map[string]interface{}
}

// Default implementations

func defaultValidationRules() []ValidationRule {
	return []ValidationRule{
		// Add default validation rules
	}
}

func defaultCorrectionStrategies() []CorrectionStrategy {
	return []CorrectionStrategy{
		// Add default correction strategies
	}
}
