package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// Custom event types for non-blocking API communication

// MessageResponseEvent is sent when an API call succeeds
type MessageResponseEvent struct {
	tcell.EventTime
	Message chat.Message
}

// MessageErrorEvent is sent when an API call fails
type MessageErrorEvent struct {
	tcell.EventTime
	Error error
}

// ViewChangeEvent is sent when a view change is requested
type ViewChangeEvent struct {
	tcell.EventTime
	ViewName string
}

// MenuToggleEvent is sent when the menu should be toggled
type MenuToggleEvent struct {
	tcell.EventTime
	Show bool
}

// ModelListUpdateEvent is sent when model list data is updated
type ModelListUpdateEvent struct {
	tcell.EventTime
	Models []ModelInfo
}

// ModelStatsUpdateEvent is sent when model statistics are updated
type ModelStatsUpdateEvent struct {
	tcell.EventTime
	Stats ModelStats
}

// NewMessageResponseEvent creates a new message response event
func NewMessageResponseEvent(message chat.Message) *MessageResponseEvent {
	return &MessageResponseEvent{
		EventTime: tcell.EventTime{},
		Message:   message,
	}
}

// NewMessageErrorEvent creates a new message error event
func NewMessageErrorEvent(err error) *MessageErrorEvent {
	return &MessageErrorEvent{
		EventTime: tcell.EventTime{},
		Error:     err,
	}
}

// NewViewChangeEvent creates a new view change event
func NewViewChangeEvent(viewName string) *ViewChangeEvent {
	return &ViewChangeEvent{
		EventTime: tcell.EventTime{},
		ViewName:  viewName,
	}
}

// NewMenuToggleEvent creates a new menu toggle event
func NewMenuToggleEvent(show bool) *MenuToggleEvent {
	return &MenuToggleEvent{
		EventTime: tcell.EventTime{},
		Show:      show,
	}
}

// NewModelListUpdateEvent creates a new model list update event
func NewModelListUpdateEvent(models []ModelInfo) *ModelListUpdateEvent {
	return &ModelListUpdateEvent{
		EventTime: tcell.EventTime{},
		Models:    models,
	}
}

// ModelErrorEvent is sent when model operations fail
type ModelErrorEvent struct {
	tcell.EventTime
	Error error
}

// NewModelStatsUpdateEvent creates a new model stats update event
func NewModelStatsUpdateEvent(stats ModelStats) *ModelStatsUpdateEvent {
	return &ModelStatsUpdateEvent{
		EventTime: tcell.EventTime{},
		Stats:     stats,
	}
}

// NewModelErrorEvent creates a new model error event
func NewModelErrorEvent(err error) *ModelErrorEvent {
	return &ModelErrorEvent{
		EventTime: tcell.EventTime{},
		Error:     err,
	}
}

// ModelDeletedEvent is sent when a model is successfully deleted
type ModelDeletedEvent struct {
	tcell.EventTime
	ModelName string
}

// NewModelDeletedEvent creates a new model deleted event
func NewModelDeletedEvent(modelName string) *ModelDeletedEvent {
	return &ModelDeletedEvent{
		EventTime: tcell.EventTime{},
		ModelName: modelName,
	}
}

// ChatMessageSendEvent is sent when a chat message should be sent
type ChatMessageSendEvent struct {
	tcell.EventTime
	Content string
}

// NewChatMessageSendEvent creates a new chat message send event
func NewChatMessageSendEvent(content string) *ChatMessageSendEvent {
	return &ChatMessageSendEvent{
		EventTime: tcell.EventTime{},
		Content:   content,
	}
}

// SpinnerAnimationEvent is sent to update spinner animation frames
type SpinnerAnimationEvent struct {
	tcell.EventTime
}

// NewSpinnerAnimationEvent creates a new spinner animation event
func NewSpinnerAnimationEvent() *SpinnerAnimationEvent {
	return &SpinnerAnimationEvent{
		EventTime: tcell.EventTime{},
	}
}

// ToolExecutionStartEvent is sent when a tool starts executing
type ToolExecutionStartEvent struct {
	tcell.EventTime
	ToolName string
}

// ToolExecutionCompleteEvent is sent when a tool completes successfully
type ToolExecutionCompleteEvent struct {
	tcell.EventTime
	ToolName string
	Result   string
}

// ToolExecutionErrorEvent is sent when a tool execution fails
type ToolExecutionErrorEvent struct {
	tcell.EventTime
	ToolName string
	Error    error
}

// NewToolExecutionStartEvent creates a new tool execution start event
func NewToolExecutionStartEvent(toolName string) *ToolExecutionStartEvent {
	return &ToolExecutionStartEvent{
		EventTime: tcell.EventTime{},
		ToolName:  toolName,
	}
}

// NewToolExecutionCompleteEvent creates a new tool execution complete event
func NewToolExecutionCompleteEvent(toolName, result string) *ToolExecutionCompleteEvent {
	return &ToolExecutionCompleteEvent{
		EventTime: tcell.EventTime{},
		ToolName:  toolName,
		Result:    result,
	}
}

// NewToolExecutionErrorEvent creates a new tool execution error event
func NewToolExecutionErrorEvent(toolName string, err error) *ToolExecutionErrorEvent {
	return &ToolExecutionErrorEvent{
		EventTime: tcell.EventTime{},
		ToolName:  toolName,
		Error:     err,
	}
}

// ModelDownloadProgressEvent is sent during model download progress
type ModelDownloadProgressEvent struct {
	tcell.EventTime
	ModelName string
	Status    string
	Progress  float64
}

// ModelDownloadCompleteEvent is sent when model download completes
type ModelDownloadCompleteEvent struct {
	tcell.EventTime
	ModelName string
}

// ModelDownloadErrorEvent is sent when model download fails
type ModelDownloadErrorEvent struct {
	tcell.EventTime
	ModelName string
	Error     error
}

// ModelNotFoundEvent is sent when a selected model is not available locally
type ModelNotFoundEvent struct {
	tcell.EventTime
	ModelName string
}

// NewModelDownloadProgressEvent creates a new model download progress event
func NewModelDownloadProgressEvent(modelName, status string, progress float64) *ModelDownloadProgressEvent {
	return &ModelDownloadProgressEvent{
		EventTime: tcell.EventTime{},
		ModelName: modelName,
		Status:    status,
		Progress:  progress,
	}
}

// NewModelDownloadCompleteEvent creates a new model download complete event
func NewModelDownloadCompleteEvent(modelName string) *ModelDownloadCompleteEvent {
	return &ModelDownloadCompleteEvent{
		EventTime: tcell.EventTime{},
		ModelName: modelName,
	}
}

// NewModelDownloadErrorEvent creates a new model download error event
func NewModelDownloadErrorEvent(modelName string, err error) *ModelDownloadErrorEvent {
	return &ModelDownloadErrorEvent{
		EventTime: tcell.EventTime{},
		ModelName: modelName,
		Error:     err,
	}
}

// NewModelNotFoundEvent creates a new model not found event
func NewModelNotFoundEvent(modelName string) *ModelNotFoundEvent {
	return &ModelNotFoundEvent{
		EventTime: tcell.EventTime{},
		ModelName: modelName,
	}
}

// Streaming Events

// MessageChunkEvent is sent when a streaming message chunk is received
type MessageChunkEvent struct {
	tcell.EventTime
	StreamID   string
	Content    string
	IsComplete bool
	ChunkIndex int
	Timestamp  time.Time
}

// StreamStartEvent is sent when a streaming message starts
type StreamStartEvent struct {
	tcell.EventTime
	StreamID string
	Model    string
}

// StreamCompleteEvent is sent when a streaming message completes
type StreamCompleteEvent struct {
	tcell.EventTime
	StreamID     string
	FinalMessage chat.Message
	TotalChunks  int
	Duration     time.Duration
}

// StreamErrorEvent is sent when a streaming message encounters an error
type StreamErrorEvent struct {
	tcell.EventTime
	StreamID string
	Error    error
}

// StreamProgressEvent is sent to update streaming progress indicators
type StreamProgressEvent struct {
	tcell.EventTime
	StreamID      string
	ContentLength int
	ChunkCount    int
	Duration      time.Duration
}

// NewMessageChunkEvent creates a new message chunk event
func NewMessageChunkEvent(streamID, content string, isComplete bool, chunkIndex int) *MessageChunkEvent {
	return &MessageChunkEvent{
		EventTime:  tcell.EventTime{},
		StreamID:   streamID,
		Content:    content,
		IsComplete: isComplete,
		ChunkIndex: chunkIndex,
		Timestamp:  time.Now(),
	}
}

// NewStreamStartEvent creates a new stream start event
func NewStreamStartEvent(streamID, model string) *StreamStartEvent {
	return &StreamStartEvent{
		EventTime: tcell.EventTime{},
		StreamID:  streamID,
		Model:     model,
	}
}

// NewStreamCompleteEvent creates a new stream complete event
func NewStreamCompleteEvent(streamID string, finalMessage chat.Message, totalChunks int, duration time.Duration) *StreamCompleteEvent {
	return &StreamCompleteEvent{
		EventTime:    tcell.EventTime{},
		StreamID:     streamID,
		FinalMessage: finalMessage,
		TotalChunks:  totalChunks,
		Duration:     duration,
	}
}

// NewStreamErrorEvent creates a new stream error event
func NewStreamErrorEvent(streamID string, err error) *StreamErrorEvent {
	return &StreamErrorEvent{
		EventTime: tcell.EventTime{},
		StreamID:  streamID,
		Error:     err,
	}
}

// NewStreamProgressEvent creates a new stream progress event
func NewStreamProgressEvent(streamID string, contentLength, chunkCount int, duration time.Duration) *StreamProgressEvent {
	return &StreamProgressEvent{
		EventTime:     tcell.EventTime{},
		StreamID:      streamID,
		ContentLength: contentLength,
		ChunkCount:    chunkCount,
		Duration:      duration,
	}
}
