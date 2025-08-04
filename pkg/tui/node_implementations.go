package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// BaseNode provides common functionality for all node types
type BaseNode struct {
	id       string
	message  chat.Message
	nodeType MessageNodeType
	state    NodeState
	bounds   NodeBounds
	cache    NodeRenderCache
}

// Common interface implementations for BaseNode

func (bn *BaseNode) ID() string                { return bn.id }
func (bn *BaseNode) Message() chat.Message     { return bn.message }
func (bn *BaseNode) NodeType() MessageNodeType { return bn.nodeType }
func (bn *BaseNode) State() NodeState          { return bn.state }
func (bn *BaseNode) Bounds() NodeBounds        { return bn.bounds }

// WithBounds will be implemented by each concrete type

// Helper method to invalidate cache when width changes
func (bn *BaseNode) invalidateCache(width int) {
	if bn.cache.LastWidth != width {
		bn.cache.Valid = false
		bn.cache.LastWidth = width
	}
}

// TextMessageNode handles regular text messages (user, assistant, system)
type TextMessageNode struct {
	BaseNode
}

func NewTextMessageNode(msg chat.Message, id string) *TextMessageNode {
	nodeType := NodeTypeText
	switch msg.Role {
	case chat.RoleSystem:
		nodeType = NodeTypeSystem
	case chat.RoleError:
		nodeType = NodeTypeError
	}

	return &TextMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: nodeType,
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
	}
}

func (tmn *TextMessageNode) WithState(state NodeState) MessageNode {
	updated := *tmn
	updated.state = state
	// Invalidate cache when state changes (especially expansion state)
	updated.cache.Valid = false
	return &updated
}

func (tmn *TextMessageNode) WithBounds(bounds NodeBounds) MessageNode {
	updated := *tmn
	updated.bounds = bounds
	return &updated
}

func (tmn *TextMessageNode) Render(area Rect, state NodeState) []RenderedLine {
	tmn.invalidateCache(area.Width)

	if !tmn.cache.Valid {
		// Determine style based on message role and state
		baseStyle := tmn.getBaseStyle()
		if state.Selected {
			baseStyle = baseStyle.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
		} else if state.Focused {
			baseStyle = baseStyle.Background(tcell.ColorGray)
		}

		var lines []string

		// Check if message is long and handle expansion/collapse
		allLines := WrapText(tmn.message.Content, area.Width)
		const maxCollapsedLines = 5 // Maximum lines to show when collapsed

		if len(allLines) > maxCollapsedLines && !state.Expanded {
			// Show truncated version with expand indicator
			lines = allLines[:maxCollapsedLines]
			lines = append(lines, "... (Tab to expand, "+fmt.Sprintf("%d more lines", len(allLines)-maxCollapsedLines)+")")
		} else {
			lines = allLines
		}

		tmn.cache.Lines = lines
		tmn.cache.Styles = make([]tcell.Style, len(lines))
		for i := range tmn.cache.Styles {
			tmn.cache.Styles[i] = baseStyle
		}
		tmn.cache.Valid = true
	}

	// Convert to RenderedLine format
	result := make([]RenderedLine, len(tmn.cache.Lines))
	for i, line := range tmn.cache.Lines {
		result[i] = RenderedLine{
			Text:   line,
			Style:  tmn.cache.Styles[i],
			Indent: 0,
		}
	}

	return result
}

func (tmn *TextMessageNode) getBaseStyle() tcell.Style {
	switch tmn.message.Role {
	case chat.RoleUser:
		return StyleUserText
	case chat.RoleAssistant:
		return StyleAssistantText
	case chat.RoleSystem:
		return StyleSystemText
	case chat.RoleError:
		return StyleBorderError
	default:
		return tcell.StyleDefault
	}
}

func (tmn *TextMessageNode) CalculateHeight(width int) int {
	tmn.invalidateCache(width)
	if !tmn.cache.Valid {
		lines := WrapText(tmn.message.Content, width)
		return len(lines)
	}
	return len(tmn.cache.Lines)
}

func (tmn *TextMessageNode) HandleClick(x, y int) (bool, NodeState) {
	// Click toggles selection
	return true, tmn.state.ToggleSelected()
}

func (tmn *TextMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	switch ev.Key() {
	case tcell.KeyEnter:
		// Enter toggles selection
		return true, tmn.state.ToggleSelected()
	}
	return false, tmn.state
}

func (tmn *TextMessageNode) IsCollapsible() bool {
	// Message is collapsible if it's more than 5 lines when wrapped
	const maxCollapsedLines = 5
	lines := WrapText(tmn.message.Content, 80) // Use a standard width for estimation
	return len(lines) > maxCollapsedLines
}

func (tmn *TextMessageNode) HasDetailView() bool {
	return tmn.IsCollapsible()
}

func (tmn *TextMessageNode) GetPreviewText() string {
	// Return first 100 characters for preview
	if len(tmn.message.Content) > 100 {
		return tmn.message.Content[:100] + "..."
	}
	return tmn.message.Content
}

// ThinkingMessageNode handles assistant messages (now simplified since thinking blocks are removed)
type ThinkingMessageNode struct {
	BaseNode
}

func NewThinkingMessageNode(msg chat.Message, id string) *ThinkingMessageNode {
	return &ThinkingMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: NodeTypeThinking,
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
	}
}

func (tmn *ThinkingMessageNode) WithState(state NodeState) MessageNode {
	updated := *tmn
	updated.state = state
	// Invalidate cache when state changes (especially expansion state)
	updated.cache.Valid = false
	return &updated
}

func (tmn *ThinkingMessageNode) WithBounds(bounds NodeBounds) MessageNode {
	updated := *tmn
	updated.bounds = bounds
	return &updated
}

func (tmn *ThinkingMessageNode) Render(area Rect, state NodeState) []RenderedLine {
	tmn.invalidateCache(area.Width)

	if !tmn.cache.Valid {
		var lines []RenderedLine

		// Render message content
		if tmn.message.Content != "" {
			responseLines := WrapText(tmn.message.Content, area.Width)
			for _, line := range responseLines {
				style := StyleAssistantText
				if state.Selected {
					style = style.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
				} else if state.Focused {
					style = style.Background(tcell.ColorGray)
				}
				lines = append(lines, RenderedLine{
					Text:   line,
					Style:  style,
					Indent: 0,
				})
			}
		}

		tmn.cache.Lines = make([]string, len(lines))
		tmn.cache.Styles = make([]tcell.Style, len(lines))
		for i, line := range lines {
			tmn.cache.Lines[i] = line.Text
			tmn.cache.Styles[i] = line.Style
		}
		tmn.cache.Valid = true

		return lines
	}

	// Convert cached data to RenderedLine format
	result := make([]RenderedLine, len(tmn.cache.Lines))
	for i, line := range tmn.cache.Lines {
		result[i] = RenderedLine{
			Text:   line,
			Style:  tmn.cache.Styles[i],
			Indent: 0,
		}
	}

	return result
}

func (tmn *ThinkingMessageNode) CalculateHeight(width int) int {
	// Calculate height for message content
	if tmn.message.Content != "" {
		responseLines := WrapText(tmn.message.Content, width)
		return len(responseLines)
	}
	return 0
}

func (tmn *ThinkingMessageNode) HandleClick(x, y int) (bool, NodeState) {
	// Click toggles selection, double-click could toggle expansion
	return true, tmn.state.ToggleSelected()
}

func (tmn *ThinkingMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	switch ev.Key() {
	case tcell.KeyEnter:
		return true, tmn.state.ToggleSelected()
	}
	return false, tmn.state
}

func (tmn *ThinkingMessageNode) IsCollapsible() bool {
	return false
}

func (tmn *ThinkingMessageNode) HasDetailView() bool {
	return false
}

func (tmn *ThinkingMessageNode) GetPreviewText() string {
	return tmn.message.Content
}

// ToolResultMessageNode handles tool execution results with special formatting
type ToolResultMessageNode struct {
	BaseNode
	truncatedOutput string
	fullOutput      string
	showFullOutput  bool
}

func NewToolResultMessageNode(msg chat.Message, id string) *ToolResultMessageNode {
	// Parse the tool result to separate tool name and output
	fullOutput := msg.Content
	truncatedOutput := fullOutput

	// Truncate long outputs for better UX
	const maxPreviewLength = 300
	if len(fullOutput) > maxPreviewLength {
		truncatedOutput = fullOutput[:maxPreviewLength] + "..."
	}

	return &ToolResultMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: NodeTypeToolResult,
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
		truncatedOutput: truncatedOutput,
		fullOutput:      fullOutput,
		showFullOutput:  false, // Start collapsed for long outputs
	}
}

func (trn *ToolResultMessageNode) WithState(state NodeState) MessageNode {
	updated := *trn
	updated.state = state
	updated.cache.Valid = false
	return &updated
}

func (trn *ToolResultMessageNode) WithBounds(bounds NodeBounds) MessageNode {
	updated := *trn
	updated.bounds = bounds
	return &updated
}

func (trn *ToolResultMessageNode) Render(area Rect, state NodeState) []RenderedLine {
	trn.invalidateCache(area.Width)

	if !trn.cache.Valid {
		var lines []RenderedLine

		// Tool header with name from message ToolName field
		toolName := trn.message.ToolName
		if toolName == "" {
			toolName = "Tool" // Fallback
		}

		headerText := fmt.Sprintf("▶ %s", toolName)
		if state.Expanded && trn.IsCollapsible() {
			headerText = fmt.Sprintf("▼ %s", toolName)
		} else if trn.IsCollapsible() {
			headerText = fmt.Sprintf("▶ %s (click to expand)", toolName)
		}

		// Render tool header
		headerLines := WrapText(headerText, area.Width)
		for _, line := range headerLines {
			style := StyleToolText.Bold(true)
			if state.Selected {
				style = style.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
			} else if state.Focused {
				style = style.Background(tcell.ColorGray)
			}
			lines = append(lines, RenderedLine{
				Text:   line,
				Style:  style,
				Indent: 0,
			})
		}

		// Render tool output with proper formatting
		outputToShow := trn.truncatedOutput
		if state.Expanded || !trn.IsCollapsible() {
			outputToShow = trn.fullOutput
		}

		if outputToShow != "" {
			// Add indentation to show this is tool output
			outputLines := WrapText(outputToShow, area.Width-2) // Reserve 2 chars for indent
			for _, line := range outputLines {
				style := StyleToolText
				if state.Selected {
					style = style.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
				} else if state.Focused {
					style = style.Background(tcell.ColorGray)
				}
				lines = append(lines, RenderedLine{
					Text:   "  " + line, // 2-space indent
					Style:  style,
					Indent: 2,
				})
			}
		}

		trn.cache.Lines = make([]string, len(lines))
		trn.cache.Styles = make([]tcell.Style, len(lines))
		for i, line := range lines {
			trn.cache.Lines[i] = line.Text
			trn.cache.Styles[i] = line.Style
		}
		trn.cache.Valid = true

		return lines
	}

	// Convert cached data to RenderedLine format
	result := make([]RenderedLine, len(trn.cache.Lines))
	for i, line := range trn.cache.Lines {
		// Determine indent from the line content
		indent := 0
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "▶") && !strings.HasPrefix(line, "▼") {
			indent = 2
		}
		result[i] = RenderedLine{
			Text:   line,
			Style:  trn.cache.Styles[i],
			Indent: indent,
		}
	}

	return result
}

func (trn *ToolResultMessageNode) CalculateHeight(width int) int {
	height := 0

	// Header height
	toolName := trn.message.ToolName
	if toolName == "" {
		toolName = "Tool"
	}
	headerText := fmt.Sprintf("▶ %s", toolName)
	headerLines := WrapText(headerText, width)
	height += len(headerLines)

	// Output height
	outputToShow := trn.truncatedOutput
	if trn.state.Expanded || !trn.IsCollapsible() {
		outputToShow = trn.fullOutput
	}

	if outputToShow != "" {
		outputLines := WrapText(outputToShow, width-2)
		height += len(outputLines)
	}

	return height
}

func (trn *ToolResultMessageNode) HandleClick(x, y int) (bool, NodeState) {
	// Click toggles expansion if collapsible, otherwise toggles selection
	if trn.IsCollapsible() {
		return true, trn.state.ToggleExpanded()
	}
	return true, trn.state.ToggleSelected()
}

func (trn *ToolResultMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	switch ev.Key() {
	case tcell.KeyEnter:
		// Enter toggles selection
		return true, trn.state.ToggleSelected()
	case tcell.KeyTab:
		// Tab toggles expansion if collapsible
		if trn.IsCollapsible() {
			return true, trn.state.ToggleExpanded()
		}
	}
	return false, trn.state
}

func (trn *ToolResultMessageNode) IsCollapsible() bool {
	// Collapsible if output is longer than truncated version
	return len(trn.fullOutput) > len(trn.truncatedOutput)
}

func (trn *ToolResultMessageNode) HasDetailView() bool {
	return trn.IsCollapsible()
}

func (trn *ToolResultMessageNode) GetPreviewText() string {
	toolName := trn.message.ToolName
	if toolName == "" {
		toolName = "Tool"
	}
	return fmt.Sprintf("%s: %s", toolName, trn.truncatedOutput)
}

// ToolProgressMessageNode handles tool execution progress messages
type ToolProgressMessageNode struct {
	BaseNode
}

func NewToolProgressMessageNode(msg chat.Message, id string) *ToolProgressMessageNode {
	return &ToolProgressMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: NodeTypeToolProgress,
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
	}
}

func (tpn *ToolProgressMessageNode) WithState(state NodeState) MessageNode {
	updated := *tpn
	updated.state = state
	return &updated
}

func (tpn *ToolProgressMessageNode) WithBounds(bounds NodeBounds) MessageNode {
	updated := *tpn
	updated.bounds = bounds
	return &updated
}

func (tpn *ToolProgressMessageNode) Render(area Rect, state NodeState) []RenderedLine {
	tpn.invalidateCache(area.Width)

	if tpn.cache.Valid {
		// Convert cached data to RenderedLine format
		result := make([]RenderedLine, len(tpn.cache.Lines))
		for i, line := range tpn.cache.Lines {
			result[i] = RenderedLine{
				Text:   line,
				Style:  tpn.cache.Styles[i],
				Indent: 0,
			}
		}
		return result
	}

	// Create a subtle progress indicator style (dim gray/yellow)
	progressStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Dim(true)

	// Format: "🔧 Shell(docker ps -a)"
	content := fmt.Sprintf("🔧 %s", tpn.message.Content)

	lines := []RenderedLine{
		{
			Text:   content,
			Style:  progressStyle,
			Indent: 0,
		},
	}

	// Cache the data
	tpn.cache.Lines = make([]string, len(lines))
	tpn.cache.Styles = make([]tcell.Style, len(lines))
	for i, line := range lines {
		tpn.cache.Lines[i] = line.Text
		tpn.cache.Styles[i] = line.Style
	}
	tpn.cache.Valid = true

	return lines
}

func (tpn *ToolProgressMessageNode) CalculateHeight(width int) int {
	return 1 // Tool progress messages are always single line
}

func (tpn *ToolProgressMessageNode) HandleClick(x, y int) (bool, NodeState) {
	return false, tpn.state // Tool progress messages are not interactive
}

func (tpn *ToolProgressMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	return false, tpn.state // No key handling for progress messages
}

func (tpn *ToolProgressMessageNode) IsCollapsible() bool {
	return false // Progress messages can't be collapsed
}

func (tpn *ToolProgressMessageNode) HasDetailView() bool {
	return false // No detail view for progress messages
}

func (tpn *ToolProgressMessageNode) GetPreviewText() string {
	return tpn.message.Content
}

// ToolCallMessageNode handles messages with tool calls
type ToolCallMessageNode struct {
	BaseNode
}

func NewToolCallMessageNode(msg chat.Message, id string) *ToolCallMessageNode {
	return &ToolCallMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: NodeTypeToolCall,
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
	}
}

func (tcn *ToolCallMessageNode) WithState(state NodeState) MessageNode {
	updated := *tcn
	updated.state = state
	// Invalidate cache when state changes (especially expansion state)
	updated.cache.Valid = false
	return &updated
}

func (tcn *ToolCallMessageNode) WithBounds(bounds NodeBounds) MessageNode {
	updated := *tcn
	updated.bounds = bounds
	return &updated
}

func (tcn *ToolCallMessageNode) Render(area Rect, state NodeState) []RenderedLine {
	tcn.invalidateCache(area.Width)

	if !tcn.cache.Valid {
		var lines []RenderedLine

		// Render tool calls in the format requested: Shell(docker ps -a) or Search("https://...")
		for i, toolCall := range tcn.message.ToolCalls {
			var toolText string

			// Format tool call based on tool type
			toolName := tcn.formatToolName(toolCall.Function.Name)

			if state.Expanded {
				// Show full arguments when expanded
				if len(toolCall.Function.Arguments) > 0 {
					// Format arguments nicely
					var argStr string
					if cmd, ok := toolCall.Function.Arguments["command"].(string); ok {
						argStr = fmt.Sprintf("(%s)", tcn.truncateCommand(cmd, 50))
					} else {
						argStr = fmt.Sprintf("(%v)", toolCall.Function.Arguments)
					}
					toolText = fmt.Sprintf("%s%s", toolName, argStr)
				} else {
					toolText = fmt.Sprintf("%s()", toolName)
				}
			} else {
				// Show truncated version when collapsed
				if cmd, ok := toolCall.Function.Arguments["command"].(string); ok {
					toolText = fmt.Sprintf("%s(%s)", toolName, tcn.truncateCommand(cmd, 30))
				} else if len(toolCall.Function.Arguments) > 0 {
					toolText = fmt.Sprintf("%s(...)", toolName)
				} else {
					toolText = fmt.Sprintf("%s()", toolName)
				}
			}

			toolLines := WrapText(toolText, area.Width)
			for _, line := range toolLines {
				style := StyleToolText.Bold(true)
				if state.Selected {
					style = style.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
				} else if state.Focused {
					style = style.Background(tcell.ColorGray)
				}
				lines = append(lines, RenderedLine{
					Text:   line,
					Style:  style,
					Indent: 0,
				})
			}

			// Add spacing between tool calls
			if i < len(tcn.message.ToolCalls)-1 {
				lines = append(lines, RenderedLine{Text: "", Style: tcell.StyleDefault, Indent: 0})
			}
		}

		tcn.cache.Lines = make([]string, len(lines))
		tcn.cache.Styles = make([]tcell.Style, len(lines))
		for i, line := range lines {
			tcn.cache.Lines[i] = line.Text
			tcn.cache.Styles[i] = line.Style
		}
		tcn.cache.Valid = true

		return lines
	}

	// Convert cached data to RenderedLine format
	result := make([]RenderedLine, len(tcn.cache.Lines))
	for i, line := range tcn.cache.Lines {
		result[i] = RenderedLine{
			Text:   line,
			Style:  tcn.cache.Styles[i],
			Indent: 0,
		}
	}

	return result
}

// formatToolName converts tool names to user-friendly display names
func (tcn *ToolCallMessageNode) formatToolName(toolName string) string {
	switch toolName {
	case "execute_bash":
		return "Shell"
	case "read_file":
		return "ReadFile"
	case "write_file":
		return "WriteFile"
	case "search_web":
		return "Search"
	default:
		// Capitalize first letter and keep the rest
		if len(toolName) > 0 {
			return strings.ToUpper(toolName[:1]) + toolName[1:]
		}
		return toolName
	}
}

// truncateCommand truncates a command string for display
func (tcn *ToolCallMessageNode) truncateCommand(cmd string, maxLen int) string {
	if len(cmd) <= maxLen {
		return cmd
	}
	return cmd[:maxLen-3] + "..."
}

func (tcn *ToolCallMessageNode) CalculateHeight(width int) int {
	height := 0
	for i, toolCall := range tcn.message.ToolCalls {
		var toolText string
		toolName := tcn.formatToolName(toolCall.Function.Name)

		if tcn.state.Expanded {
			if len(toolCall.Function.Arguments) > 0 {
				var argStr string
				if cmd, ok := toolCall.Function.Arguments["command"].(string); ok {
					argStr = fmt.Sprintf("(%s)", tcn.truncateCommand(cmd, 50))
				} else {
					argStr = fmt.Sprintf("(%v)", toolCall.Function.Arguments)
				}
				toolText = fmt.Sprintf("%s%s", toolName, argStr)
			} else {
				toolText = fmt.Sprintf("%s()", toolName)
			}
		} else {
			if cmd, ok := toolCall.Function.Arguments["command"].(string); ok {
				toolText = fmt.Sprintf("%s(%s)", toolName, tcn.truncateCommand(cmd, 30))
			} else if len(toolCall.Function.Arguments) > 0 {
				toolText = fmt.Sprintf("%s(...)", toolName)
			} else {
				toolText = fmt.Sprintf("%s()", toolName)
			}
		}

		lines := WrapText(toolText, width)
		height += len(lines)

		if i < len(tcn.message.ToolCalls)-1 {
			height += 1 // spacing
		}
	}
	return height
}

func (tcn *ToolCallMessageNode) HandleClick(x, y int) (bool, NodeState) {
	return true, tcn.state.ToggleSelected()
}

func (tcn *ToolCallMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	switch ev.Key() {
	case tcell.KeyEnter:
		return true, tcn.state.ToggleSelected()
	case tcell.KeyTab:
		return true, tcn.state.ToggleExpanded()
	}
	return false, tcn.state
}

func (tcn *ToolCallMessageNode) IsCollapsible() bool { return true }
func (tcn *ToolCallMessageNode) HasDetailView() bool { return true }
func (tcn *ToolCallMessageNode) GetPreviewText() string {
	if len(tcn.message.ToolCalls) > 0 {
		return fmt.Sprintf("Tool: %s", tcn.message.ToolCalls[0].Function.Name)
	}
	return "Tool call"
}

// Node Factory Implementations

type TextNodeFactory struct{}

func (tnf *TextNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewTextMessageNode(msg, id)
}

func (tnf *TextNodeFactory) CanHandle(msg chat.Message) bool {
	// Handle user, system, error messages
	if msg.Role == chat.RoleUser || msg.Role == chat.RoleSystem || msg.Role == chat.RoleError {
		return true
	}

	// Handle assistant messages without tool calls
	if msg.Role == chat.RoleAssistant {
		hasTools := len(msg.ToolCalls) > 0
		return !hasTools
	}

	return false
}

type ThinkingNodeFactory struct{}

func (tnf *ThinkingNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewThinkingMessageNode(msg, id)
}

func (tnf *ThinkingNodeFactory) CanHandle(msg chat.Message) bool {
	// Since thinking blocks are removed, this factory is no longer used
	return false
}

type ToolCallNodeFactory struct{}

func (tcf *ToolCallNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewToolCallMessageNode(msg, id)
}

func (tcf *ToolCallNodeFactory) CanHandle(msg chat.Message) bool {
	return len(msg.ToolCalls) > 0
}

type ToolResultNodeFactory struct{}

func (trf *ToolResultNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewToolResultMessageNode(msg, id)
}

func (trf *ToolResultNodeFactory) CanHandle(msg chat.Message) bool {
	return msg.Role == chat.RoleTool
}

type ToolProgressNodeFactory struct{}

func (tpf *ToolProgressNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewToolProgressMessageNode(msg, id)
}

func (tpf *ToolProgressNodeFactory) CanHandle(msg chat.Message) bool {
	return msg.Role == chat.RoleToolProgress
}

type SystemNodeFactory struct{}

func (snf *SystemNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewTextMessageNode(msg, id)
}

func (snf *SystemNodeFactory) CanHandle(msg chat.Message) bool {
	return msg.Role == chat.RoleSystem
}

type ErrorNodeFactory struct{}

func (enf *ErrorNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewTextMessageNode(msg, id)
}

func (enf *ErrorNodeFactory) CanHandle(msg chat.Message) bool {
	return msg.Role == chat.RoleError
}
