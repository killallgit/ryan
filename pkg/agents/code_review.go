package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// CodeReviewAgent performs comprehensive code reviews using context from other agents
type CodeReviewAgent struct {
	reviewEngine  *ReviewEngine
	issueDetector *IssueDetector
	log           *logger.Logger
}

// NewCodeReviewAgent creates a new code review agent
func NewCodeReviewAgent() *CodeReviewAgent {
	return &CodeReviewAgent{
		reviewEngine:  NewReviewEngine(),
		issueDetector: NewIssueDetector(),
		log:           logger.WithComponent("code_review_agent"),
	}
}

// Name returns the agent name
func (c *CodeReviewAgent) Name() string {
	return "code_review"
}

// Description returns the agent description
func (c *CodeReviewAgent) Description() string {
	return "Performs comprehensive code reviews with architectural analysis and best practices"
}

// CanHandle determines if this agent can handle the request
func (c *CodeReviewAgent) CanHandle(request string) (bool, float64) {
	lowerRequest := strings.ToLower(request)

	keywords := []string{
		"code review", "review", "critique", "feedback",
		"improve", "suggestions", "best practices",
	}

	for _, keyword := range keywords {
		if strings.Contains(lowerRequest, keyword) {
			return true, 0.9
		}
	}

	return false, 0.0
}

// Execute performs the code review
func (c *CodeReviewAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	c.log.Info("Executing code review", "prompt", request.Prompt)

	// Get analysis results from context
	analysisResults, err := c.getAnalysisResults(request)
	if err != nil {
		c.log.Warn("No analysis results found, performing basic review", "error", err)
	}

	// Get file contents
	fileContents, err := c.getFileContents(request)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to get file contents",
			Details: err.Error(),
		}, err
	}

	// Perform review
	review := c.performReview(fileContents, analysisResults)

	// Build summary and details
	summary := c.buildSummary(review)
	details := c.buildDetails(review)

	return AgentResult{
		Success: true,
		Summary: summary,
		Details: details,
		Artifacts: map[string]interface{}{
			"review":           review,
			"issues":           review.Issues,
			"suggestions":      review.Suggestions,
			"positive_aspects": review.PositiveAspects,
		},
		Metadata: AgentMetadata{
			AgentName:      c.Name(),
			StartTime:      startTime,
			EndTime:        time.Now(),
			Duration:       time.Since(startTime),
			FilesProcessed: review.FilesReviewed,
		},
	}, nil
}

// getAnalysisResults retrieves code analysis results from context
func (c *CodeReviewAgent) getAnalysisResults(request AgentRequest) (map[string]*FileAnalysis, error) {
	// Check execution context
	if execContext, ok := request.Context["execution_context"].(*ExecutionContext); ok {
		if results, ok := execContext.SharedData["analysis_results"].(map[string]*FileAnalysis); ok {
			return results, nil
		}
	}

	// Check artifacts
	if artifacts, ok := request.Context["artifacts"].(map[string]interface{}); ok {
		if results, ok := artifacts["analysis_results"].(map[string]*FileAnalysis); ok {
			return results, nil
		}
	}

	return nil, fmt.Errorf("no analysis results found")
}

// getFileContents retrieves file contents from context
func (c *CodeReviewAgent) getFileContents(request AgentRequest) (map[string]string, error) {
	// Check execution context
	if execContext, ok := request.Context["execution_context"].(*ExecutionContext); ok {
		if fileContents, ok := execContext.SharedData["file_contents"].(map[string]string); ok {
			return fileContents, nil
		}
	}

	// Check direct context
	if fileContents, ok := request.Context["file_contents"].(map[string]string); ok {
		return fileContents, nil
	}

	return nil, fmt.Errorf("no file contents found in context")
}

// performReview performs the actual code review
func (c *CodeReviewAgent) performReview(fileContents map[string]string, analysisResults map[string]*FileAnalysis) *CodeReview {
	review := &CodeReview{
		ReviewID:        generateID(),
		Timestamp:       time.Now(),
		FilesReviewed:   make([]string, 0, len(fileContents)),
		Issues:          []Issue{},
		Suggestions:     []Suggestion{},
		PositiveAspects: []string{},
		OverallScore:    0,
	}

	// Review each file
	for filePath, content := range fileContents {
		review.FilesReviewed = append(review.FilesReviewed, filePath)

		// Get analysis for this file if available
		var analysis *FileAnalysis
		if analysisResults != nil {
			analysis = analysisResults[filePath]
		}

		// Perform various checks
		fileIssues := c.reviewFile(filePath, content, analysis)
		review.Issues = append(review.Issues, fileIssues...)

		// Generate suggestions
		fileSuggestions := c.generateSuggestions(filePath, content, analysis)
		review.Suggestions = append(review.Suggestions, fileSuggestions...)

		// Note positive aspects
		positives := c.findPositiveAspects(filePath, content, analysis)
		review.PositiveAspects = append(review.PositiveAspects, positives...)
	}

	// Calculate overall score
	review.OverallScore = c.calculateScore(review)

	return review
}

// reviewFile performs various checks on a single file
func (c *CodeReviewAgent) reviewFile(filePath, content string, analysis *FileAnalysis) []Issue {
	issues := []Issue{}

	// Check code style
	styleIssues := c.issueDetector.CheckStyle(filePath, content)
	issues = append(issues, styleIssues...)

	// Check complexity
	if analysis != nil {
		for _, fn := range analysis.Functions {
			if c.isFunctionComplex(fn, content) {
				issues = append(issues, Issue{
					File:     filePath,
					Line:     fn.Position,
					Severity: "warning",
					Type:     "complexity",
					Message:  fmt.Sprintf("Function %s appears to be complex and might benefit from refactoring", fn.Name),
				})
			}
		}
	}

	// Check for common issues
	commonIssues := c.issueDetector.CheckCommonIssues(filePath, content)
	issues = append(issues, commonIssues...)

	return issues
}

// generateSuggestions creates improvement suggestions
func (c *CodeReviewAgent) generateSuggestions(filePath, content string, analysis *FileAnalysis) []Suggestion {
	suggestions := []Suggestion{}

	// Architecture suggestions
	if analysis != nil && len(analysis.Types) > 10 {
		suggestions = append(suggestions, Suggestion{
			File:     filePath,
			Type:     "architecture",
			Priority: "medium",
			Message:  "Consider splitting this file into smaller, more focused modules",
			Details:  fmt.Sprintf("This file contains %d type definitions which might indicate it's doing too much", len(analysis.Types)),
		})
	}

	// Testing suggestions
	if strings.HasSuffix(filePath, ".go") && !strings.HasSuffix(filePath, "_test.go") {
		hasTests := false
		// Simple check - in real implementation would check for corresponding test file
		if !hasTests {
			suggestions = append(suggestions, Suggestion{
				File:     filePath,
				Type:     "testing",
				Priority: "high",
				Message:  "Consider adding unit tests for this file",
				Details:  "No corresponding test file found",
			})
		}
	}

	// Documentation suggestions
	if analysis != nil {
		for _, fn := range analysis.Functions {
			if fn.Receiver == "" && !strings.HasPrefix(fn.Name, "new") && !strings.HasPrefix(fn.Name, "New") {
				// Check if function has comment (simplified check)
				if !strings.Contains(content, fmt.Sprintf("// %s", fn.Name)) {
					suggestions = append(suggestions, Suggestion{
						File:     filePath,
						Type:     "documentation",
						Priority: "low",
						Message:  fmt.Sprintf("Consider adding documentation for exported function %s", fn.Name),
						Details:  "Exported functions should have documentation comments",
					})
				}
			}
		}
	}

	return suggestions
}

// findPositiveAspects identifies good practices in the code
func (c *CodeReviewAgent) findPositiveAspects(filePath, content string, analysis *FileAnalysis) []string {
	positives := []string{}

	// Check for good practices
	if strings.Contains(content, "context.Context") {
		positives = append(positives, fmt.Sprintf("%s: Good use of context for cancellation and timeouts", filePath))
	}

	if strings.Contains(content, "defer") {
		positives = append(positives, fmt.Sprintf("%s: Proper use of defer for cleanup", filePath))
	}

	if analysis != nil {
		// Check for interface usage
		interfaceCount := 0
		for _, typ := range analysis.Types {
			if typ.Kind == "interface" {
				interfaceCount++
			}
		}
		if interfaceCount > 0 {
			positives = append(positives, fmt.Sprintf("%s: Good use of interfaces for abstraction", filePath))
		}
	}

	return positives
}

// Helper methods

func (c *CodeReviewAgent) isFunctionComplex(fn Function, content string) bool {
	// Simple complexity check - count lines
	// In real implementation would use cyclomatic complexity
	lines := strings.Split(content, "\n")
	functionLines := 0
	inFunction := false

	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("func %s", fn.Name)) {
			inFunction = true
		}
		if inFunction {
			functionLines++
			if strings.TrimSpace(line) == "}" {
				break
			}
		}
	}

	return functionLines > 50
}

func (c *CodeReviewAgent) calculateScore(review *CodeReview) float64 {
	// Simple scoring algorithm
	score := 100.0

	// Deduct points for issues
	for _, issue := range review.Issues {
		switch issue.Severity {
		case "error":
			score -= 10
		case "warning":
			score -= 5
		case "info":
			score -= 2
		}
	}

	// Add points for positive aspects
	score += float64(len(review.PositiveAspects)) * 2

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func (c *CodeReviewAgent) buildSummary(review *CodeReview) string {
	return fmt.Sprintf("Reviewed %d files: found %d issues, made %d suggestions. Overall score: %.1f/100",
		len(review.FilesReviewed),
		len(review.Issues),
		len(review.Suggestions),
		review.OverallScore,
	)
}

func (c *CodeReviewAgent) buildDetails(review *CodeReview) string {
	var details []string

	details = append(details, fmt.Sprintf("Code Review Report (ID: %s)", review.ReviewID))
	details = append(details, fmt.Sprintf("Timestamp: %s", review.Timestamp.Format(time.RFC3339)))
	details = append(details, fmt.Sprintf("Overall Score: %.1f/100", review.OverallScore))
	details = append(details, "")

	if len(review.Issues) > 0 {
		details = append(details, "## Issues Found:")
		for _, issue := range review.Issues {
			details = append(details, fmt.Sprintf("- [%s] %s: %s (%s)",
				issue.Severity, issue.File, issue.Message, issue.Type))
		}
		details = append(details, "")
	}

	if len(review.Suggestions) > 0 {
		details = append(details, "## Suggestions:")
		for _, suggestion := range review.Suggestions {
			details = append(details, fmt.Sprintf("- [%s] %s: %s",
				suggestion.Priority, suggestion.File, suggestion.Message))
			if suggestion.Details != "" {
				details = append(details, fmt.Sprintf("  Details: %s", suggestion.Details))
			}
		}
		details = append(details, "")
	}

	if len(review.PositiveAspects) > 0 {
		details = append(details, "## Positive Aspects:")
		for _, positive := range review.PositiveAspects {
			details = append(details, fmt.Sprintf("- %s", positive))
		}
		details = append(details, "")
	}

	return strings.Join(details, "\n")
}

// Supporting types

type CodeReview struct {
	ReviewID        string
	Timestamp       time.Time
	FilesReviewed   []string
	Issues          []Issue
	Suggestions     []Suggestion
	PositiveAspects []string
	OverallScore    float64
}

type Issue struct {
	File     string
	Line     string
	Severity string // error, warning, info
	Type     string // style, complexity, bug, security
	Message  string
}

type Suggestion struct {
	File     string
	Type     string // architecture, testing, documentation, performance
	Priority string // high, medium, low
	Message  string
	Details  string
}

// Helper components

type ReviewEngine struct{}

func NewReviewEngine() *ReviewEngine {
	return &ReviewEngine{}
}

type IssueDetector struct{}

func NewIssueDetector() *IssueDetector {
	return &IssueDetector{}
}

func (i *IssueDetector) CheckStyle(filePath, content string) []Issue {
	issues := []Issue{}

	// Simple style checks
	lines := strings.Split(content, "\n")
	for idx, line := range lines {
		// Check line length
		if len(line) > 120 {
			issues = append(issues, Issue{
				File:     filePath,
				Line:     fmt.Sprintf("%d", idx+1),
				Severity: "info",
				Type:     "style",
				Message:  "Line exceeds 120 characters",
			})
		}

		// Check for TODO comments
		if strings.Contains(line, "TODO") {
			issues = append(issues, Issue{
				File:     filePath,
				Line:     fmt.Sprintf("%d", idx+1),
				Severity: "info",
				Type:     "style",
				Message:  "TODO comment found",
			})
		}
	}

	return issues
}

func (i *IssueDetector) CheckCommonIssues(filePath, content string) []Issue {
	issues := []Issue{}

	// Check for common Go issues
	if strings.HasSuffix(filePath, ".go") {
		// Check for error handling
		if strings.Contains(content, "err != nil") && !strings.Contains(content, "if err != nil") {
			issues = append(issues, Issue{
				File:     filePath,
				Line:     "various",
				Severity: "warning",
				Type:     "bug",
				Message:  "Potential unhandled error",
			})
		}

		// Check for resource leaks
		if strings.Contains(content, "defer") && strings.Contains(content, ".Close()") {
			// Good - has defer close
		} else if strings.Contains(content, ".Open") || strings.Contains(content, "net.Dial") {
			issues = append(issues, Issue{
				File:     filePath,
				Line:     "various",
				Severity: "warning",
				Type:     "bug",
				Message:  "Potential resource leak - ensure resources are properly closed",
			})
		}
	}

	return issues
}
