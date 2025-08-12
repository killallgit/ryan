package react

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/stream"
)

// Interceptor formats ReAct reasoning steps for display
type Interceptor struct {
	base   stream.Handler
	buffer bytes.Buffer
	state  parseState
}

type parseState int

const (
	stateNormal parseState = iota
	stateThought
	stateAction
	stateActionInput
	stateObservation
	stateFinalAnswer
)

// NewInterceptor creates a new ReAct interceptor
func NewInterceptor(base stream.Handler) *Interceptor {
	return &Interceptor{
		base:  base,
		state: stateNormal,
	}
}

// OnChunk processes incoming chunks and formats them
func (i *Interceptor) OnChunk(chunk []byte) error {
	// Add to buffer for line-based processing
	i.buffer.Write(chunk)

	// Process complete lines
	for {
		line, err := i.buffer.ReadString('\n')
		if err != nil {
			// No complete line yet, put back what we read
			if len(line) > 0 {
				i.buffer.WriteString(line)
			}
			break
		}

		// Process the line
		formatted := i.formatLine(line)
		if formatted != "" {
			if err := i.base.OnChunk([]byte(formatted)); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatLine formats a single line based on ReAct patterns
func (i *Interceptor) formatLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Detect state changes
	switch {
	case strings.HasPrefix(trimmed, "Thought:"):
		i.state = stateThought
		content := strings.TrimPrefix(trimmed, "Thought:")
		return fmt.Sprintf("🤔 **Thinking:** %s\n", strings.TrimSpace(content))

	case strings.HasPrefix(trimmed, "Action:"):
		i.state = stateAction
		content := strings.TrimPrefix(trimmed, "Action:")
		return fmt.Sprintf("⚡ **Action:** %s\n", strings.TrimSpace(content))

	case strings.HasPrefix(trimmed, "Action Input:"):
		i.state = stateActionInput
		content := strings.TrimPrefix(trimmed, "Action Input:")
		return fmt.Sprintf("📝 **Input:** %s\n", strings.TrimSpace(content))

	case strings.HasPrefix(trimmed, "Observation:"):
		i.state = stateObservation
		content := strings.TrimPrefix(trimmed, "Observation:")
		return fmt.Sprintf("👁️ **Observation:** %s\n", strings.TrimSpace(content))

	case strings.HasPrefix(trimmed, "Final Answer:"):
		i.state = stateFinalAnswer
		content := strings.TrimPrefix(trimmed, "Final Answer:")
		return fmt.Sprintf("✅ **Answer:** %s\n", strings.TrimSpace(content))

	case strings.HasPrefix(trimmed, "Plan:"):
		i.state = stateNormal
		return fmt.Sprintf("📋 **Plan:**\n")

	case strings.HasPrefix(trimmed, "Summary:"):
		i.state = stateNormal
		content := strings.TrimPrefix(trimmed, "Summary:")
		return fmt.Sprintf("📊 **Summary:** %s\n", strings.TrimSpace(content))

	default:
		// Continue previous state formatting
		switch i.state {
		case stateThought:
			if trimmed != "" {
				return fmt.Sprintf("   %s\n", trimmed)
			}
		case stateObservation:
			if trimmed != "" {
				return fmt.Sprintf("   %s\n", trimmed)
			}
		case stateFinalAnswer:
			if trimmed != "" {
				return fmt.Sprintf("   %s\n", trimmed)
			}
		default:
			// Pass through unchanged
			return line
		}
	}

	return ""
}

// OnComplete handles completion
func (i *Interceptor) OnComplete(finalContent string) error {
	// Process any remaining buffer content
	if i.buffer.Len() > 0 {
		remaining := i.buffer.String()
		formatted := i.formatLine(remaining)
		if formatted != "" {
			if err := i.base.OnChunk([]byte(formatted)); err != nil {
				return err
			}
		}
	}

	// Pass completion to base handler
	return i.base.OnComplete(finalContent)
}

// OnError handles errors
func (i *Interceptor) OnError(err error) {
	i.base.OnError(err)
}

// Ensure Interceptor implements Handler
var _ stream.Handler = (*Interceptor)(nil)
