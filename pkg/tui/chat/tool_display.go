package chat

import (
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/stream/core"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

// ToolDisplay formats tool events for display in the chat
type ToolDisplay struct{}

// FormatToolEvent formats a tool event for display
func (td *ToolDisplay) FormatToolEvent(event core.ToolEvent) string {
	switch event.Type {
	case core.ToolEventStart:
		return td.formatToolStart(event)
	case core.ToolEventOutput:
		return td.formatToolOutput(event)
	case core.ToolEventComplete:
		return td.formatToolComplete(event)
	case core.ToolEventError:
		return td.formatToolError(event)
	default:
		return ""
	}
}

// formatToolStart formats a tool start event
func (td *ToolDisplay) formatToolStart(event core.ToolEvent) string {
	// Format: ● ToolName(args...)
	args := td.formatArguments(event.Arguments)
	return fmt.Sprintf("%s %s(%s)",
		theme.Styles.ToolIndicator.Render("●"),
		theme.Styles.ToolName.Render(event.Name),
		args)
}

// formatToolOutput formats a tool output event
func (td *ToolDisplay) formatToolOutput(event core.ToolEvent) string {
	// Format: ⎿ <truncated output>
	output := td.truncateOutput(event.Output, 200)
	lines := strings.Split(output, "\n")

	var formatted []string
	for i, line := range lines {
		if i == 0 {
			formatted = append(formatted, fmt.Sprintf("  %s %s",
				theme.Styles.ToolOutputPrefix.Render("⎿"),
				line))
		} else {
			formatted = append(formatted, fmt.Sprintf("    %s", line))
		}
	}

	return strings.Join(formatted, "\n")
}

// formatToolComplete formats a tool complete event
func (td *ToolDisplay) formatToolComplete(event core.ToolEvent) string {
	if event.Output != "" {
		return td.formatToolOutput(event)
	}
	return fmt.Sprintf("  %s %s",
		theme.Styles.ToolOutputPrefix.Render("⎿"),
		theme.Styles.ToolSuccess.Render("✓ Complete"))
}

// formatToolError formats a tool error event
func (td *ToolDisplay) formatToolError(event core.ToolEvent) string {
	return fmt.Sprintf("  %s %s",
		theme.Styles.ToolOutputPrefix.Render("⎿"),
		theme.Styles.ToolError.Render("✗ "+event.Error))
}

// formatArguments formats tool arguments for display
func (td *ToolDisplay) formatArguments(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}

	var parts []string
	for key, value := range args {
		// Format value based on type
		var valueStr string
		switch v := value.(type) {
		case string:
			// Truncate long strings
			if len(v) > 50 {
				valueStr = fmt.Sprintf("\"%s...\"", v[:47])
			} else {
				valueStr = fmt.Sprintf("\"%s\"", v)
			}
		default:
			valueStr = fmt.Sprintf("%v", v)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, valueStr))
	}

	return strings.Join(parts, ", ")
}

// truncateOutput truncates output for display
func (td *ToolDisplay) truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}

	// Try to truncate at a newline if possible
	truncated := output[:maxLen]
	if idx := strings.LastIndex(truncated, "\n"); idx > maxLen/2 {
		truncated = truncated[:idx]
	}

	lines := strings.Split(truncated, "\n")
	if len(lines) > 3 {
		// Keep first 2 lines and last line
		result := append(lines[:2], "...", lines[len(lines)-1])
		return strings.Join(result, "\n")
	}

	return truncated + "..."
}
