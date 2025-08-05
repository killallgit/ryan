package tui

import (
	"github.com/killallgit/ryan/pkg/chat"
)

// TestChatView wraps ChatView with additional methods for testing
type TestChatView struct {
	*ChatView
}

// NewTestChatView creates a new TestChatView
func NewTestChatView(cv *ChatView) *TestChatView {
	return &TestChatView{ChatView: cv}
}

// WithMessages sets the messages for testing
func (tcv *TestChatView) WithMessages(messages []chat.Message) *TestChatView {
	tcv.messages = MessageDisplay{
		Messages: messages,
		Width:    tcv.messages.Width,
		Height:   tcv.messages.Height,
		Scroll:   tcv.messages.Scroll,
	}
	tcv.updateMessages()
	return tcv
}

// WithStreamingContent sets streaming content for testing
func (tcv *TestChatView) WithStreamingContent(content string, id string, isThinking bool) *TestChatView {
	tcv.isStreaming = true
	tcv.streamingContent = content
	tcv.currentStreamID = id
	tcv.isStreamingThinking = isThinking
	if isThinking {
		tcv.thinkingContent = content
	} else {
		tcv.responseContent = content
	}
	return tcv
}

// GetMessages returns the current messages
func (tcv *TestChatView) GetMessages() []chat.Message {
	return tcv.messages.Messages
}

// IsStreaming returns whether the view is currently streaming
func (tcv *TestChatView) IsStreaming() bool {
	return tcv.isStreaming
}

// GetInputContent returns the current input field content
func (tcv *TestChatView) GetInputContent() string {
	return tcv.input.Content
}