package tui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
)

// ChatViewMode represents the current interaction mode
type ChatViewMode int

const (
	ModeInput ChatViewMode = iota // Text input mode (default)
	ModeNodes                     // Node selection/navigation mode
)

func (m ChatViewMode) String() string {
	switch m {
	case ModeInput:
		return "Input"
	case ModeNodes:
		return "Select"
	default:
		return "Unknown"
	}
}

type ChatView struct {
	controller       controllers.ChatControllerInterface
	input            InputField
	messages         MessageDisplay
	status           StatusBar
	layout           Layout
	screen           tcell.Screen
	alert            AlertDisplay     // Legacy field for compatibility
	statusRow        StatusRowDisplay // New enhanced status row
	downloadModal    DownloadPromptModal
	progressModal    ProgressModal
	helpModal        HelpModal
	downloadCtx      context.Context
	downloadCancel   context.CancelFunc
	pendingMessage   string
	modelsController *controllers.ModelsController

	// Interaction mode
	mode ChatViewMode // Current interaction mode

	// Streaming state
	isStreaming         bool
	streamingContent    string
	currentStreamID     string
	isStreamingThinking bool   // Track if currently streaming thinking content
	thinkingContent     string // Accumulate thinking content separately
	responseContent     string // Accumulate response content separately

	// Early detection buffering
	contentBuffer       string // Buffer for early content type detection
	contentTypeDetected bool   // Whether we've determined the content type
	bufferSize          int    // Current buffer size
}

func NewChatView(controller controllers.ChatControllerInterface, modelsController *controllers.ModelsController, screen tcell.Screen) *ChatView {
	width, height := screen.Size()

	view := &ChatView{
		controller:       controller,
		input:            NewInputField(width),
		messages:         NewMessageDisplay(width, height-5), // -5 for status, input, and alert areas
		status:           NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready").WithModelAvailability(true),
		layout:           NewLayout(width, height),
		screen:           screen,
		alert:            NewAlertDisplay(width),
		statusRow:        NewStatusRowDisplay(width),
		downloadModal:    NewDownloadPromptModal(),
		progressModal:    NewProgressModal(),
		helpModal:        NewHelpModal(),
		downloadCtx:      nil,
		downloadCancel:   nil,
		pendingMessage:   "",
		modelsController: modelsController,

		// Initialize interaction mode
		mode: ModeInput, // Start in input mode

		// Initialize streaming state
		isStreaming:         false,
		streamingContent:    "",
		currentStreamID:     "",
		isStreamingThinking: false,
		thinkingContent:     "",
		responseContent:     "",

		// Initialize buffering state
		contentBuffer:       "",
		contentTypeDetected: false,
		bufferSize:          0,
	}

	view.updateMessages()
	return view
}

func (cv *ChatView) Name() string {
	return "chat"
}

func (cv *ChatView) Description() string {
	return "Chat with AI"
}
