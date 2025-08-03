package tui

import (
	"fmt"

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
	parsed := ParseThinkingBlock(msg.Content)

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
		showThinking: true, // Default to showing thinking
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

		// Render tool calls
		for i, toolCall := range tcn.message.ToolCalls {
			toolText := fmt.Sprintf("🔧 Tool: %s", toolCall.Function.Name)
			if state.Expanded {
				// Show arguments if expanded
				if len(toolCall.Function.Arguments) > 0 {
					toolText += fmt.Sprintf(" (args: %v)", toolCall.Function.Arguments)
				}
			}

			toolLines := WrapText(toolText, area.Width)
			for _, line := range toolLines {
				style := StyleAssistantText.Foreground(tcell.ColorGreen)
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

func (tcn *ToolCallMessageNode) CalculateHeight(width int) int {
	height := 0
	for i, toolCall := range tcn.message.ToolCalls {
		toolText := fmt.Sprintf("🔧 Tool: %s", toolCall.Function.Name)
		if tcn.state.Expanded {
			if len(toolCall.Function.Arguments) > 0 {
				toolText += fmt.Sprintf(" (args: %v)", toolCall.Function.Arguments)
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
		// Check if it has thinking blocks or tool calls
		parsed := ParseThinkingBlock(msg.Content)
		hasTools := len(msg.ToolCalls) > 0
		return !parsed.HasThinking && !hasTools
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
	// For now, tool results are handled as text nodes
	// Could be extended later for specialized rendering
	return NewTextMessageNode(msg, id)
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
