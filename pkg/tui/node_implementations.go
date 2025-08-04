package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
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
	if msg.Role == chat.RoleSystem {
		nodeType = NodeTypeSystem
	} else if msg.Role == chat.RoleError {
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

// ThinkingMessageNode handles assistant messages with thinking blocks
type ThinkingMessageNode struct {
	BaseNode
	parsed       ParsedContent
	showThinking bool
}

func NewThinkingMessageNode(msg chat.Message, id string) *ThinkingMessageNode {
	// Use new separated thinking structure if available, fallback to parsing
	var parsed ParsedContent
	var showThinking bool

	if msg.HasThinking() {
		// Use separated thinking data
		parsed = ParsedContent{
			ThinkingBlock:   msg.Thinking.Content,
			ResponseContent: msg.Content,
			HasThinking:     true,
		}
		showThinking = msg.Thinking.Visible
	} else {
		// Fallback to old parsing method for backwards compatibility
		parsed = ParseThinkingBlock(msg.Content)

		// Get showThinking from config
		showThinking = true
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Config not initialized, use default
				}
			}()
			if cfg := config.Get(); cfg != nil {
				showThinking = cfg.ShowThinking
			}
		}()
	}

	return &ThinkingMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: NodeTypeThinking,
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
		parsed:       parsed,
		showThinking: showThinking,
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

		// Render thinking block if present and expanded
		if tmn.parsed.HasThinking && tmn.showThinking && state.Expanded {
			var thinkingText string

			// Show full thinking content when expanded
			thinkingText = "Thinking: " + tmn.parsed.ThinkingBlock
			if tmn.parsed.ResponseContent != "" {
				// Still truncate if there's response content to save space
				thinkingText = "Thinking: " + TruncateThinkingBlock(tmn.parsed.ThinkingBlock, 3, area.Width-10)
			}

			thinkingLines := WrapText(thinkingText, area.Width)
			for _, line := range thinkingLines {
				style := StyleThinkingText
				if state.Selected {
					// Keep the dimmed color but add blue background for selection
					style = style.Background(tcell.ColorBlue)
				} else if state.Focused {
					style = style.Background(tcell.ColorGray)
				}
				lines = append(lines, RenderedLine{
					Text:   line,
					Style:  style,
					Indent: 0,
				})
			}

			// Add separator if response content follows
			if tmn.parsed.ResponseContent != "" {
				lines = append(lines, RenderedLine{Text: "", Style: tcell.StyleDefault, Indent: 0})
			}
		}

		// Render response content
		var contentToRender string
		if !tmn.parsed.HasThinking {
			// No thinking block, always show content
			contentToRender = tmn.message.Content
		} else {
			// Has thinking block, always show response content (both when collapsed and expanded)
			contentToRender = tmn.parsed.ResponseContent
		}

		if contentToRender != "" {
			responseLines := WrapText(contentToRender, area.Width)
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
	// This is a simplified calculation - for performance we might cache this
	totalHeight := 0

	// Calculate height for thinking block (only when expanded)
	if tmn.parsed.HasThinking && tmn.showThinking && tmn.state.Expanded {
		thinkingText := "Thinking: " + tmn.parsed.ThinkingBlock
		if tmn.parsed.ResponseContent != "" {
			thinkingText = "Thinking: " + TruncateThinkingBlock(tmn.parsed.ThinkingBlock, 3, width-10)
		}
		thinkingLines := WrapText(thinkingText, width)
		totalHeight += len(thinkingLines)

		if tmn.parsed.ResponseContent != "" {
			totalHeight += 1 // separator line
		}
	}

	// Calculate height for response content (always shown)
	var contentToRender string
	if !tmn.parsed.HasThinking {
		contentToRender = tmn.message.Content
	} else {
		contentToRender = tmn.parsed.ResponseContent
	}

	if contentToRender != "" {
		responseLines := WrapText(contentToRender, width)
		totalHeight += len(responseLines)
	}

	return totalHeight
}

func (tmn *ThinkingMessageNode) HandleClick(x, y int) (bool, NodeState) {
	// Click toggles selection, double-click could toggle expansion
	return true, tmn.state.ToggleSelected()
}

func (tmn *ThinkingMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	switch ev.Key() {
	case tcell.KeyEnter:
		return true, tmn.state.ToggleSelected()
	case tcell.KeyTab:
		// Tab toggles thinking block visibility
		if tmn.IsCollapsible() {
			return true, tmn.state.ToggleExpanded()
		}
	}
	return false, tmn.state
}

func (tmn *ThinkingMessageNode) IsCollapsible() bool {
	return tmn.parsed.HasThinking
}

func (tmn *ThinkingMessageNode) HasDetailView() bool {
	return tmn.parsed.HasThinking
}

func (tmn *ThinkingMessageNode) GetPreviewText() string {
	if tmn.parsed.ResponseContent != "" {
		return tmn.parsed.ResponseContent
	}
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
	// Handle user, system, error messages, and assistant messages without thinking/tools
	if msg.Role == chat.RoleUser || msg.Role == chat.RoleSystem || msg.Role == chat.RoleError {
		return true
	}

	if msg.Role == chat.RoleAssistant {
		// Check if it has separated thinking first
		if msg.HasThinking() {
			return false
		}

		// Check if it has tool calls
		hasTools := len(msg.ToolCalls) > 0
		if hasTools {
			return false
		}

		// Fallback: check if content has thinking blocks for backwards compatibility
		parsed := ParseThinkingBlock(msg.Content)
		return !parsed.HasThinking
	}

	return false
}

type ThinkingNodeFactory struct{}

func (tnf *ThinkingNodeFactory) CreateNode(msg chat.Message, id string) MessageNode {
	return NewThinkingMessageNode(msg, id)
}

func (tnf *ThinkingNodeFactory) CanHandle(msg chat.Message) bool {
	if msg.Role != chat.RoleAssistant {
		return false
	}

	// Check if message has separated thinking first
	if msg.HasThinking() {
		return true
	}

	// Fallback to parsing content for backwards compatibility
	parsed := ParseThinkingBlock(msg.Content)
	return parsed.HasThinking
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

// ToolExecutionMessageNode handles real-time tool execution feedback
type ToolExecutionMessageNode struct {
	BaseNode
	toolDisplayName string
	executionState  ToolExecutionState
	progress        string
	result          string
}

// ToolExecutionState represents the current state of tool execution
type ToolExecutionState int

const (
	ToolStateStarted ToolExecutionState = iota
	ToolStateProgress
	ToolStateCompleted
	ToolStateError
)

func NewToolExecutionMessageNode(toolName, displayName string, id string) *ToolExecutionMessageNode {
	// Create a synthetic message for the tool execution
	msg := chat.Message{
		Role:      "tool_execution", // Special role for tool execution nodes
		Content:   fmt.Sprintf("Executing %s...", displayName),
		Timestamp: time.Now(),
	}

	return &ToolExecutionMessageNode{
		BaseNode: BaseNode{
			id:       id,
			message:  msg,
			nodeType: NodeTypeText, // We'll create a new type later if needed
			state:    NewNodeState(),
			bounds:   NodeBounds{},
			cache:    NodeRenderCache{},
		},
		toolDisplayName: displayName,
		executionState:  ToolStateStarted,
		progress:        "",
		result:          "",
	}
}

func (temn *ToolExecutionMessageNode) WithState(state NodeState) MessageNode {
	updated := *temn
	updated.state = state
	updated.cache.Valid = false
	return &updated
}

func (temn *ToolExecutionMessageNode) WithBounds(bounds NodeBounds) MessageNode {
	updated := *temn
	updated.bounds = bounds
	return &updated
}

// UpdateExecutionState updates the tool execution state
func (temn *ToolExecutionMessageNode) UpdateExecutionState(state ToolExecutionState, progress, result string) *ToolExecutionMessageNode {
	updated := *temn
	updated.executionState = state
	updated.progress = progress
	updated.result = result
	updated.cache.Valid = false // Invalidate cache when state changes

	// Update the message content based on state
	switch state {
	case ToolStateStarted:
		updated.message.Content = fmt.Sprintf("Executing %s...", temn.toolDisplayName)
	case ToolStateProgress:
		updated.message.Content = fmt.Sprintf("Executing %s... %s", temn.toolDisplayName, progress)
	case ToolStateCompleted:
		updated.message.Content = fmt.Sprintf("✓ %s", temn.toolDisplayName)
	case ToolStateError:
		updated.message.Content = fmt.Sprintf("✗ %s (failed)", temn.toolDisplayName)
	}

	return &updated
}

func (temn *ToolExecutionMessageNode) Render(area Rect, state NodeState) []RenderedLine {
	temn.invalidateCache(area.Width)

	if !temn.cache.Valid {
		var lines []RenderedLine

		// Main tool execution line with appropriate styling and spinner/status
		var mainText string
		var mainStyle tcell.Style

		switch temn.executionState {
		case ToolStateStarted:
			mainText = fmt.Sprintf("⏳ %s", temn.toolDisplayName)
			mainStyle = StyleToolText.Foreground(tcell.ColorYellow)
		case ToolStateProgress:
			mainText = fmt.Sprintf("⏳ %s", temn.toolDisplayName)
			if temn.progress != "" {
				mainText += fmt.Sprintf(" (%s)", temn.progress)
			}
			mainStyle = StyleToolText.Foreground(tcell.ColorYellow)
		case ToolStateCompleted:
			mainText = fmt.Sprintf("✓ %s", temn.toolDisplayName)
			mainStyle = StyleToolText.Foreground(tcell.ColorGreen)
		case ToolStateError:
			mainText = fmt.Sprintf("✗ %s", temn.toolDisplayName)
			mainStyle = StyleToolText.Foreground(tcell.ColorRed)
		}

		if state.Selected {
			mainStyle = mainStyle.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
		} else if state.Focused {
			mainStyle = mainStyle.Background(tcell.ColorGray)
		}

		mainLines := WrapText(mainText, area.Width)
		for _, line := range mainLines {
			lines = append(lines, RenderedLine{
				Text:   line,
				Style:  mainStyle,
				Indent: 0,
			})
		}

		// Show result if completed and expanded
		if temn.executionState == ToolStateCompleted && temn.result != "" && state.Expanded {
			// Add separator
			lines = append(lines, RenderedLine{Text: "", Style: tcell.StyleDefault, Indent: 0})

			// Add result content with indentation
			resultLines := WrapText(temn.result, area.Width-2)
			for _, line := range resultLines {
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

		temn.cache.Lines = make([]string, len(lines))
		temn.cache.Styles = make([]tcell.Style, len(lines))
		for i, line := range lines {
			temn.cache.Lines[i] = line.Text
			temn.cache.Styles[i] = line.Style
		}
		temn.cache.Valid = true

		return lines
	}

	// Convert cached data to RenderedLine format
	result := make([]RenderedLine, len(temn.cache.Lines))
	for i, line := range temn.cache.Lines {
		// Determine indent from line content
		indent := 0
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "⏳") && !strings.HasPrefix(line, "✓") && !strings.HasPrefix(line, "✗") {
			indent = 2
		}
		result[i] = RenderedLine{
			Text:   line,
			Style:  temn.cache.Styles[i],
			Indent: indent,
		}
	}

	return result
}

func (temn *ToolExecutionMessageNode) CalculateHeight(width int) int {
	height := 0

	// Main tool execution line
	mainText := fmt.Sprintf("⏳ %s", temn.toolDisplayName)
	if temn.progress != "" {
		mainText += fmt.Sprintf(" (%s)", temn.progress)
	}
	mainLines := WrapText(mainText, width)
	height += len(mainLines)

	// Result lines if expanded and completed
	if temn.executionState == ToolStateCompleted && temn.result != "" && temn.state.Expanded {
		height += 1 // separator
		resultLines := WrapText(temn.result, width-2)
		height += len(resultLines)
	}

	return height
}

func (temn *ToolExecutionMessageNode) HandleClick(x, y int) (bool, NodeState) {
	// Click toggles expansion if there's a result to show
	if temn.executionState == ToolStateCompleted && temn.result != "" {
		return true, temn.state.ToggleExpanded()
	}
	return true, temn.state.ToggleSelected()
}

func (temn *ToolExecutionMessageNode) HandleKeyEvent(ev *tcell.EventKey) (bool, NodeState) {
	switch ev.Key() {
	case tcell.KeyEnter:
		return true, temn.state.ToggleSelected()
	case tcell.KeyTab:
		if temn.executionState == ToolStateCompleted && temn.result != "" {
			return true, temn.state.ToggleExpanded()
		}
	}
	return false, temn.state
}

func (temn *ToolExecutionMessageNode) IsCollapsible() bool {
	return temn.executionState == ToolStateCompleted && temn.result != ""
}

func (temn *ToolExecutionMessageNode) HasDetailView() bool {
	return temn.IsCollapsible()
}

func (temn *ToolExecutionMessageNode) GetPreviewText() string {
	return temn.toolDisplayName
}
