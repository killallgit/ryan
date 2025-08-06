package tui

import (
	"context"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/rivo/tview"
)

// MockController implements ControllerInterface for testing
type MockController struct {
	history []chat.Message
	model   string
}

func (mc *MockController) SendUserMessage(content string) (chat.Message, error) {
	msg := chat.Message{Role: chat.RoleUser, Content: content}
	mc.history = append(mc.history, msg)
	return msg, nil
}

func (mc *MockController) GetHistory() []chat.Message {
	return mc.history
}

func (mc *MockController) GetModel() string {
	return mc.model
}

func (mc *MockController) SetModel(model string) {
	mc.model = model
}

func (mc *MockController) AddUserMessage(content string) {
	mc.history = append(mc.history, chat.Message{Role: chat.RoleUser, Content: content})
}

func (mc *MockController) AddErrorMessage(errorMsg string) {
	mc.history = append(mc.history, chat.Message{Role: chat.RoleError, Content: errorMsg})
}

func (mc *MockController) Reset() {
	mc.history = nil
}

func (mc *MockController) StartStreaming(ctx context.Context, content string) (<-chan controllers.StreamingUpdate, error) {
	ch := make(chan controllers.StreamingUpdate, 1)
	close(ch)
	return ch, nil
}

func (mc *MockController) SetOllamaClient(client any) {
	// No-op for testing
}

func (mc *MockController) ValidateModel(model string) error {
	return nil
}

func (mc *MockController) GetToolRegistry() *tools.Registry {
	return tools.NewRegistry()
}

func (mc *MockController) GetTokenUsage() (promptTokens, responseTokens int) {
	return 0, 0
}

func (mc *MockController) CleanThinkingBlocks() {
	// No-op for testing
}

func TestChatViewInitialization(t *testing.T) {
	// Create a mock controller
	controller := &MockController{
		history: []chat.Message{
			{Role: chat.RoleUser, Content: "Hello"},
			{Role: chat.RoleAssistant, Content: "Hi there!"},
		},
		model: "test-model",
	}

	// Create a new tview application
	app := tview.NewApplication()

	// Test ChatView creation
	chatView := NewChatView(controller, app)
	if chatView == nil {
		t.Fatal("ChatView creation failed")
	}

	// Test that render manager is initialized (can be nil if creation failed)
	// This is not a failure condition since we have fallback rendering
	t.Logf("Render manager initialized: %v", chatView.renderManager != nil)

	// Test message updates don't panic
	chatView.UpdateMessages()

	// Test streaming functionality
	chatView.StartStreaming("test-stream-1")
	chatView.UpdateStreamingContent("test-stream-1", "Streaming content...")
	chatView.CompleteStreaming("test-stream-1", chat.Message{
		Role:    chat.RoleAssistant,
		Content: "Final content",
	})

	// Test sending state
	chatView.SetSending(true)
	chatView.SetSending(false)

	// Test activity tree
	chatView.UpdateActivityTree("test tree content")
	chatView.ClearActivityTree()

	// Test resize handler
	chatView.OnResize(120, 40)
}

func TestChatViewMessageFormatting(t *testing.T) {
	controller := &MockController{
		history: []chat.Message{
			{Role: chat.RoleUser, Content: "# Test Markdown\n\nThis is **bold** text."},
			{Role: chat.RoleAssistant, Content: "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"},
			{Role: chat.RoleAssistant, Content: "<thinking>Let me think...</thinking>\n\nHere's my answer."},
			{Role: chat.RoleError, Content: "An error occurred"},
		},
		model: "test-model",
	}

	app := tview.NewApplication()
	chatView := NewChatView(controller, app)

	// Test that UpdateMessages doesn't panic with various content types
	chatView.UpdateMessages()

	// Test streaming with different content types
	testStreamingContent := []string{
		"Plain text streaming",
		"# Markdown header streaming",
		"```python\nprint('code')\n```",
		"<thinking>Partial thinking block",
		"{\"json\": \"streaming\"}",
	}

	for i, content := range testStreamingContent {
		streamID := "test-stream-" + string(rune(i))
		chatView.StartStreaming(streamID)
		chatView.UpdateStreamingContent(streamID, content)
		chatView.CompleteStreaming(streamID, chat.Message{
			Role:    chat.RoleAssistant,
			Content: content + " [COMPLETE]",
		})
	}
}

func TestChatViewInputHandling(t *testing.T) {
	controller := &MockController{
		model: "test-model",
	}

	app := tview.NewApplication()
	chatView := NewChatView(controller, app)

	// Test message handler setting
	chatView.SetSendMessageHandler(func(content string) {
		// Message received in handler
	})

	// Simulate key events (this tests that input handler doesn't panic)
	inputHandler := chatView.InputHandler()
	if inputHandler == nil {
		t.Fatal("Input handler is nil")
	}

	// Test various key events
	testKeys := []tcell.Key{
		tcell.KeyEnter,
		tcell.KeyEscape,
		tcell.KeyPgUp,
		tcell.KeyPgDn,
		tcell.KeyUp,
		tcell.KeyDown,
	}

	for _, key := range testKeys {
		event := tcell.NewEventKey(key, 0, tcell.ModNone)
		// This shouldn't panic
		inputHandler(event, func(p tview.Primitive) {})
	}
}

func TestChatViewThemeConsistency(t *testing.T) {
	controller := &MockController{model: "test-model"}
	app := tview.NewApplication()
	chatView := NewChatView(controller, app)

	// Test that theme is applied consistently
	theme := DefaultTheme()
	ApplyTheme(theme)

	// Verify background color is set
	bgColor := chatView.GetBackgroundColor()
	expectedBg := tcell.GetColor(ColorBase00)
	if bgColor != expectedBg {
		t.Errorf("Expected background color %v, got %v", expectedBg, bgColor)
	}

	// Test components have consistent styling
	if chatView.messages == nil {
		t.Error("Messages component not initialized")
	}
	if chatView.input == nil {
		t.Error("Input component not initialized")
	}
	if chatView.footer == nil {
		t.Error("Footer component not initialized")
	}
	if chatView.statusContainer == nil {
		t.Error("Status container not initialized")
	}
	if chatView.modelView == nil {
		t.Error("Model view not initialized")
	}
}
