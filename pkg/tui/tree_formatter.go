package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/agents"
)

// TreeFormatter formats activity trees for display in the TUI
type TreeFormatter struct {
	useColor   bool
	maxWidth   int
	showTiming bool
	compact    bool
}

// NewTreeFormatter creates a new tree formatter
func NewTreeFormatter(useColor bool, maxWidth int) *TreeFormatter {
	return &TreeFormatter{
		useColor:   useColor,
		maxWidth:   maxWidth,
		showTiming: false,
		compact:    true,
	}
}

// FormatActivityTree formats an activity tree for display
func (tf *TreeFormatter) FormatActivityTree(tree *agents.ActivityTree) string {
	if tree == nil || tree.IsEmpty() {
		return ""
	}

	var output strings.Builder
	children := tree.GetRootChildren()

	for i, child := range children {
		isLast := i == len(children)-1
		tf.formatNode(&output, child, "", isLast, 0)
	}

	return output.String()
}

// formatNode recursively formats a node and its children
func (tf *TreeFormatter) formatNode(output *strings.Builder, node *agents.ActivityNode, prefix string, isLast bool, depth int) {
	// Skip completed nodes in compact mode
	if tf.compact && node.Status == agents.ActivityStatusComplete {
		return
	}

	// Skip nodes that are too deep
	if depth > 5 {
		return
	}

	// Choose connector
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	// Build the line
	line := prefix + connector + tf.formatNodeContent(node)

	// Truncate if too wide
	if tf.maxWidth > 0 && len(line) > tf.maxWidth {
		line = line[:tf.maxWidth-3] + "..."
	}

	output.WriteString(line + "\n")

	// Format children
	node.Mu.RLock()
	children := node.Children
	node.Mu.RUnlock()

	for i, child := range children {
		childIsLast := i == len(children)-1
		tf.formatNode(output, child, childPrefix, childIsLast, depth+1)
	}
}

// formatNodeContent formats the content of a single node
func (tf *TreeFormatter) formatNodeContent(node *agents.ActivityNode) string {
	node.Mu.RLock()
	defer node.Mu.RUnlock()

	var parts []string

	// Agent name with color
	agentName := node.AgentName
	if tf.useColor {
		agentName = tf.colorizeAgent(agentName, node.Status)
	}
	parts = append(parts, agentName)

	// Operation
	if node.Operation != "" && node.Operation != "idle" {
		operation := node.Operation
		if tf.useColor {
			operation = tf.colorizeOperation(operation, node.OperationType)
		}
		parts = append(parts, "› "+operation)
	}

	// Status indicator
	statusIndicator := tf.getStatusIndicator(node.Status)
	if statusIndicator != "" {
		parts = append(parts, statusIndicator)
	}

	// Progress
	if node.Progress > 0 && node.Progress < 100 {
		progress := fmt.Sprintf("[%.0f%%]", node.Progress)
		if tf.useColor {
			progress = tf.colorizeProgress(progress, node.Progress)
		}
		parts = append(parts, progress)
	}

	// Timing
	if tf.showTiming && node.Status == agents.ActivityStatusActive {
		duration := time.Since(node.StartTime)
		timing := tf.formatDuration(duration)
		parts = append(parts, timing)
	}

	// Error indicator
	if node.Error != nil {
		errorMsg := "✗ " + node.Error.Error()
		if len(errorMsg) > 30 {
			errorMsg = errorMsg[:27] + "..."
		}
		if tf.useColor {
			errorMsg = fmt.Sprintf("[red]%s[-]", errorMsg)
		}
		parts = append(parts, errorMsg)
	}

	return strings.Join(parts, " ")
}

// colorizeAgent applies color to agent names based on status
func (tf *TreeFormatter) colorizeAgent(agent string, status agents.ActivityStatus) string {
	switch status {
	case agents.ActivityStatusActive:
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, agent)
	case agents.ActivityStatusPending:
		return fmt.Sprintf("[%s]%s[-]", ColorYellow, agent)
	case agents.ActivityStatusError:
		return fmt.Sprintf("[%s]%s[-]", ColorRed, agent)
	case agents.ActivityStatusComplete:
		return fmt.Sprintf("[%s]%s[-]", ColorBase03, agent)
	default:
		return agent
	}
}

// colorizeOperation applies color to operations based on type
func (tf *TreeFormatter) colorizeOperation(operation string, opType agents.OperationType) string {
	switch opType {
	case agents.OperationTypeTool:
		return fmt.Sprintf("[%s]%s[-]", ColorCyan, operation)
	case agents.OperationTypeAgent:
		return fmt.Sprintf("[%s]%s[-]", ColorMagenta, operation)
	case agents.OperationTypeAnalysis:
		return fmt.Sprintf("[%s]%s[-]", ColorBlue, operation)
	case agents.OperationTypePlanning:
		return fmt.Sprintf("[%s]%s[-]", ColorViolet, operation)
	default:
		return operation
	}
}

// colorizeProgress applies color to progress indicators
func (tf *TreeFormatter) colorizeProgress(progress string, percent float64) string {
	if percent < 33 {
		return fmt.Sprintf("[%s]%s[-]", ColorRed, progress)
	} else if percent < 66 {
		return fmt.Sprintf("[%s]%s[-]", ColorYellow, progress)
	} else {
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, progress)
	}
}

// getStatusIndicator returns a status indicator character
func (tf *TreeFormatter) getStatusIndicator(status agents.ActivityStatus) string {
	switch status {
	case agents.ActivityStatusActive:
		if tf.useColor {
			return fmt.Sprintf("[%s]●[-]", ColorGreen)
		}
		return "●"
	case agents.ActivityStatusPending:
		if tf.useColor {
			return fmt.Sprintf("[%s]○[-]", ColorYellow)
		}
		return "○"
	case agents.ActivityStatusError:
		if tf.useColor {
			return fmt.Sprintf("[%s]✗[-]", ColorRed)
		}
		return "✗"
	case agents.ActivityStatusComplete:
		if tf.useColor {
			return fmt.Sprintf("[%s]✓[-]", ColorGreen)
		}
		return "✓"
	default:
		return ""
	}
}

// formatDuration formats a duration for display
func (tf *TreeFormatter) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// SetShowTiming enables or disables timing display
func (tf *TreeFormatter) SetShowTiming(show bool) {
	tf.showTiming = show
}

// SetCompact enables or disables compact mode
func (tf *TreeFormatter) SetCompact(compact bool) {
	tf.compact = compact
}

// AnimatedSpinner provides animated spinner characters
type AnimatedSpinner struct {
	frames  []string
	current int
}

// NewAnimatedSpinner creates a new animated spinner
func NewAnimatedSpinner() *AnimatedSpinner {
	return &AnimatedSpinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Next returns the next frame in the animation
func (as *AnimatedSpinner) Next() string {
	frame := as.frames[as.current]
	as.current = (as.current + 1) % len(as.frames)
	return frame
}

// Reset resets the spinner to the first frame
func (as *AnimatedSpinner) Reset() {
	as.current = 0
}

// ProgressBar creates a visual progress bar
func ProgressBar(progress float64, width int) string {
	if width <= 0 {
		return ""
	}

	filled := int(float64(width) * progress / 100)
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled)
	if filled < width {
		bar += strings.Repeat("░", width-filled)
	}

	return bar
}

// ActivityIndicator combines spinner and text for activity display
type ActivityIndicator struct {
	spinner   *AnimatedSpinner
	formatter *TreeFormatter
	lastFrame time.Time
	frameRate time.Duration
}

// NewActivityIndicator creates a new activity indicator
func NewActivityIndicator(useColor bool, maxWidth int) *ActivityIndicator {
	return &ActivityIndicator{
		spinner:   NewAnimatedSpinner(),
		formatter: NewTreeFormatter(useColor, maxWidth),
		frameRate: 100 * time.Millisecond,
	}
}

// Format formats the activity tree with animation
func (ai *ActivityIndicator) Format(tree *agents.ActivityTree) string {
	if tree == nil || tree.IsEmpty() {
		return ""
	}

	// Update spinner frame if enough time has passed
	now := time.Now()
	if now.Sub(ai.lastFrame) >= ai.frameRate {
		ai.spinner.Next()
		ai.lastFrame = now
	}

	// Get the tree format
	treeStr := ai.formatter.FormatActivityTree(tree)
	if treeStr == "" {
		return ""
	}

	// Add spinner to the first line if there are active nodes
	if len(tree.GetActiveActivities()) > 0 {
		lines := strings.Split(treeStr, "\n")
		if len(lines) > 0 && lines[0] != "" {
			// Find where to insert the spinner
			if idx := strings.Index(lines[0], "├── "); idx >= 0 {
				lines[0] = lines[0][:idx] + "├── " + ai.spinner.frames[ai.spinner.current] + " " + lines[0][idx+4:]
			} else if idx := strings.Index(lines[0], "└── "); idx >= 0 {
				lines[0] = lines[0][:idx] + "└── " + ai.spinner.frames[ai.spinner.current] + " " + lines[0][idx+4:]
			}
		}
		treeStr = strings.Join(lines, "\n")
	}

	return treeStr
}

// GetFrameRate returns the current frame rate
func (ai *ActivityIndicator) GetFrameRate() time.Duration {
	return ai.frameRate
}

// SetFrameRate sets the animation frame rate
func (ai *ActivityIndicator) SetFrameRate(rate time.Duration) {
	ai.frameRate = rate
}

// FormatCompactTree formats a tree in ultra-compact single-line mode
func FormatCompactTree(tree *agents.ActivityTree) string {
	if tree == nil || tree.IsEmpty() {
		return ""
	}

	active := tree.GetActiveActivities()
	if len(active) == 0 {
		return ""
	}

	// Take the first few active nodes
	parts := make([]string, 0, 3)
	for i, node := range active {
		if i >= 3 {
			parts = append(parts, "...")
			break
		}

		part := node.AgentName
		if node.Operation != "" && node.Operation != "idle" {
			part += "(" + node.Operation + ")"
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, " › ")
}

// ColorForStatus returns the appropriate color for a status
func ColorForStatus(status agents.ActivityStatus) tcell.Color {
	switch status {
	case agents.ActivityStatusActive:
		return tcell.GetColor(ColorGreen)
	case agents.ActivityStatusPending:
		return tcell.GetColor(ColorYellow)
	case agents.ActivityStatusError:
		return tcell.GetColor(ColorRed)
	case agents.ActivityStatusComplete:
		return tcell.GetColor(ColorBase03)
	default:
		return tcell.GetColor(ColorBase05)
	}
}
