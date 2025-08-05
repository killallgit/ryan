package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// SearchAgent handles code search operations
type SearchAgent struct {
	toolRegistry *tools.Registry
	log          *logger.Logger
}

// NewSearchAgent creates a new search agent
func NewSearchAgent(toolRegistry *tools.Registry) *SearchAgent {
	return &SearchAgent{
		toolRegistry: toolRegistry,
		log:          logger.WithComponent("search_agent"),
	}
}

// Name returns the agent name
func (s *SearchAgent) Name() string {
	return "search"
}

// Description returns the agent description
func (s *SearchAgent) Description() string {
	return "Searches for code patterns, symbols, and text across files"
}

// CanHandle determines if this agent can handle the request
func (s *SearchAgent) CanHandle(request string) (bool, float64) {
	lowerRequest := strings.ToLower(request)

	keywords := []string{
		"search", "find", "grep", "locate",
		"where is", "look for", "search for",
	}

	for _, keyword := range keywords {
		if strings.Contains(lowerRequest, keyword) {
			return true, 0.8
		}
	}

	return false, 0.0
}

// Execute performs the search operation
func (s *SearchAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	s.log.Info("Executing search", "prompt", request.Prompt)

	// Extract search pattern
	pattern := s.extractSearchPattern(request.Prompt)
	if pattern == "" {
		return AgentResult{
			Success: false,
			Summary: "No search pattern found",
			Details: "Please specify what to search for",
		}, nil
	}

	// Determine search scope
	scope := s.determineScope(request)

	// Use grep tool
	grepTool, exists := s.toolRegistry.Get("grep")
	if !exists {
		return AgentResult{
			Success: false,
			Summary: "Search tool not available",
			Details: "The grep tool is not registered",
		}, fmt.Errorf("grep tool not available")
	}

	// Execute search
	result, err := grepTool.Execute(ctx, map[string]interface{}{
		"pattern":     pattern,
		"path":        scope.Path,
		"glob":        scope.Glob,
		"output_mode": "content",
		"-n":          true, // Show line numbers
		"-i":          scope.CaseInsensitive,
		"head_limit":  100, // Limit results
	})

	if err != nil {
		return AgentResult{
			Success: false,
			Summary: fmt.Sprintf("Search failed for pattern: %s", pattern),
			Details: err.Error(),
		}, err
	}

	// Process results
	matches := s.processSearchResults(result.Content)

	// Build summary and details
	summary := fmt.Sprintf("Found %d matches for '%s'", len(matches), pattern)
	details := s.buildSearchDetails(matches, pattern)

	return AgentResult{
		Success: true,
		Summary: summary,
		Details: details,
		Artifacts: map[string]interface{}{
			"pattern": pattern,
			"matches": matches,
			"scope":   scope,
		},
		Metadata: AgentMetadata{
			AgentName: s.Name(),
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime),
			ToolsUsed: []string{"grep"},
		},
	}, nil
}

// extractSearchPattern extracts the search pattern from the prompt
func (s *SearchAgent) extractSearchPattern(prompt string) string {
	// Look for quoted patterns
	if start := strings.Index(prompt, "\""); start != -1 {
		if end := strings.Index(prompt[start+1:], "\""); end != -1 {
			return prompt[start+1 : start+1+end]
		}
	}

	if start := strings.Index(prompt, "'"); start != -1 {
		if end := strings.Index(prompt[start+1:], "'"); end != -1 {
			return prompt[start+1 : start+1+end]
		}
	}

	// Look for pattern after keywords
	keywords := []string{" for ", " search ", " find ", " grep "}
	lowerPrompt := strings.ToLower(prompt)

	for _, keyword := range keywords {
		if idx := strings.Index(lowerPrompt, keyword); idx != -1 {
			pattern := prompt[idx+len(keyword):]
			// Take until next space or punctuation
			if endIdx := strings.IndexAny(pattern, " ,;."); endIdx != -1 {
				pattern = pattern[:endIdx]
			}
			return strings.TrimSpace(pattern)
		}
	}

	// Last resort - take the last word
	words := strings.Fields(prompt)
	if len(words) > 0 {
		return words[len(words)-1]
	}

	return ""
}

// determineScope determines the search scope from the request
func (s *SearchAgent) determineScope(request AgentRequest) SearchScope {
	scope := SearchScope{
		Path:            ".",
		Glob:            "",
		CaseInsensitive: false,
	}

	prompt := strings.ToLower(request.Prompt)

	// Check for file type specifications
	if strings.Contains(prompt, "go file") || strings.Contains(prompt, ".go") {
		scope.Glob = "*.go"
	} else if strings.Contains(prompt, "test") {
		scope.Glob = "*_test.go"
	} else if strings.Contains(prompt, "javascript") || strings.Contains(prompt, ".js") {
		scope.Glob = "*.js"
	}

	// Check for case sensitivity
	if strings.Contains(prompt, "case insensitive") || strings.Contains(prompt, "ignore case") {
		scope.CaseInsensitive = true
	}

	// Check for path specifications
	if execContext, ok := request.Context["execution_context"].(*ExecutionContext); ok {
		if targetPath, ok := execContext.SharedData["target_path"].(string); ok {
			scope.Path = targetPath
		}
	}

	return scope
}

// processSearchResults processes the raw search output
func (s *SearchAgent) processSearchResults(output string) []SearchMatch {
	matches := []SearchMatch{}
	lines := strings.Split(output, "\n")

	currentFile := ""
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse grep output format: filename:line_number:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			file := parts[0]
			lineNum := parts[1]
			content := parts[2]

			if file != currentFile {
				currentFile = file
			}

			matches = append(matches, SearchMatch{
				File:    file,
				Line:    lineNum,
				Content: strings.TrimSpace(content),
			})
		}
	}

	return matches
}

// buildSearchDetails builds detailed search results
func (s *SearchAgent) buildSearchDetails(matches []SearchMatch, pattern string) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for pattern: %s", pattern)
	}

	var details []string
	details = append(details, fmt.Sprintf("Search results for: '%s'", pattern))
	details = append(details, fmt.Sprintf("Found %d matches:", len(matches)))
	details = append(details, "")

	// Group by file
	fileGroups := make(map[string][]SearchMatch)
	for _, match := range matches {
		fileGroups[match.File] = append(fileGroups[match.File], match)
	}

	// Display results grouped by file
	for file, fileMatches := range fileGroups {
		details = append(details, fmt.Sprintf("File: %s (%d matches)", file, len(fileMatches)))
		for _, match := range fileMatches {
			details = append(details, fmt.Sprintf("  Line %s: %s", match.Line, match.Content))
		}
		details = append(details, "")
	}

	return strings.Join(details, "\n")
}

// Supporting types

type SearchScope struct {
	Path            string
	Glob            string
	CaseInsensitive bool
}

type SearchMatch struct {
	File    string
	Line    string
	Content string
}
