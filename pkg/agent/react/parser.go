package react

import (
	"regexp"
	"strings"
)

// ParsedResponse represents a parsed ReAct response
type ParsedResponse struct {
	Thought     string
	Action      string
	ActionInput string
	Observation string
	FinalAnswer string
	Raw         string
}

// ResponseParser parses LLM responses to extract ReAct components
type ResponseParser struct {
	thoughtRegex     *regexp.Regexp
	actionRegex      *regexp.Regexp
	actionInputRegex *regexp.Regexp
	observationRegex *regexp.Regexp
	finalAnswerRegex *regexp.Regexp
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{
		thoughtRegex:     regexp.MustCompile(`(?i)thought:\s*(.+?)(?:\n|$)`),
		actionRegex:      regexp.MustCompile(`(?i)action:\s*(.+?)(?:\n|$)`),
		actionInputRegex: regexp.MustCompile(`(?i)action\s+input:\s*(.+?)(?:\n|$)`),
		observationRegex: regexp.MustCompile(`(?i)observation:\s*(.+?)(?:\n|$)`),
		finalAnswerRegex: regexp.MustCompile(`(?i)final\s+answer:\s*(.+?)(?:\n|$)`),
	}
}

// Parse extracts ReAct components from the response
func (p *ResponseParser) Parse(response string) (*ParsedResponse, error) {
	parsed := &ParsedResponse{
		Raw: response,
	}

	// Extract thought
	if matches := p.thoughtRegex.FindStringSubmatch(response); len(matches) > 1 {
		parsed.Thought = strings.TrimSpace(matches[1])
	}

	// Extract action
	if matches := p.actionRegex.FindStringSubmatch(response); len(matches) > 1 {
		parsed.Action = strings.TrimSpace(matches[1])
	}

	// Extract action input
	if matches := p.actionInputRegex.FindStringSubmatch(response); len(matches) > 1 {
		parsed.ActionInput = strings.TrimSpace(matches[1])
	}

	// Extract observation (if present in response)
	if matches := p.observationRegex.FindStringSubmatch(response); len(matches) > 1 {
		parsed.Observation = strings.TrimSpace(matches[1])
	}

	// Extract final answer
	if matches := p.finalAnswerRegex.FindStringSubmatch(response); len(matches) > 1 {
		parsed.FinalAnswer = strings.TrimSpace(matches[1])
	}

	// If no ReAct format found, treat entire response as thought or answer
	if parsed.Thought == "" && parsed.Action == "" && parsed.FinalAnswer == "" {
		// Check if it looks like a direct answer
		if !strings.Contains(strings.ToLower(response), "thought:") &&
			!strings.Contains(strings.ToLower(response), "action:") {
			parsed.FinalAnswer = strings.TrimSpace(response)
		} else {
			parsed.Thought = strings.TrimSpace(response)
		}
	}

	return parsed, nil
}

// ParseJSON parses JSON-formatted action input
func (p *ResponseParser) ParseJSON(input string) (map[string]interface{}, error) {
	// For now, we'll handle simple key:value pairs
	// This can be enhanced to handle actual JSON parsing
	result := make(map[string]interface{})

	// Try to parse as simple key:value format
	if strings.Contains(input, ":") {
		parts := strings.SplitN(input, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	} else {
		// Single value, use default key
		result["input"] = input
	}

	return result, nil
}
