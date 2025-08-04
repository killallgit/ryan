# Streaming Implementation Guide

✅ **Status**: Production ready with full HTTP streaming and TUI integration

## Overview

Ryan implements real-time streaming functionality that enables users to see assistant responses as they are generated, providing an interactive experience similar to Claude Code. The streaming system integrates seamlessly with the tool calling system and maintains full UI responsiveness.

## Architecture

### Core Components

The streaming system consists of several key components working together:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TUI Layer     │    │  Controller     │    │   Chat Core     │    │ Streaming Client│
│   (events)      │◄───│    Layer        │◄───│   (business)    │◄───│  (HTTP/JSON)    │
│                 │    │                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │                       │
        ▼                       ▼                       ▼                       ▼
   UI Updates              Orchestration            Message Logic         Chunk Processing
   Event Handling          State Management         Stream Coordination   Error Handling
   Progress Display        Tool Integration         Result Accumulation   Connection Mgmt
```

### 1. Streaming Client (`pkg/chat/stream.go`)

**Purpose**: Handles HTTP streaming communication with Ollama API

**Key Features**:
- Buffers message chunks for optimal performance
- Manages connection lifecycle and error handling
- Generates unique stream identifiers
- Supports cancellation via context
- Automatic fallback to non-streaming mode

**Usage**:
```go
client := chat.NewStreamingClient("http://localhost:11434")
chunks, err := client.StreamMessage(ctx, chatRequest)

for chunk := range chunks {
    if chunk.Error != nil {
        // Handle streaming error
        break
    }
    if chunk.Done {
        // Stream complete
        break
    }
    // Process chunk.Content for UI updates
}
```

### 2. Message Accumulator (`pkg/chat/accumulator.go`)

**Purpose**: Assembles streaming chunks into complete messages

**Key Features**:
- Thread-safe accumulation of message content
- Tracks streaming statistics (chunk count, duration, etc.)
- Handles Unicode boundary issues between chunks
- Memory-efficient cleanup of completed streams

**Usage**:
```go
acc := chat.NewMessageAccumulator()

// Process incoming chunks
acc.AddChunk(chunk)

// Get current accumulated content
content := acc.GetCurrentContent(streamID)

// Check if stream is complete
if acc.IsComplete(streamID) {
    finalMessage, exists := acc.GetCompleteMessage(streamID)
    acc.CleanupStream(streamID) // Clean up resources
}
```

### 3. Streaming Events (`pkg/tui/events.go`)

**Purpose**: Defines TUI event types for streaming communication

**Event Types**:
- `MessageChunkEvent`: Individual content chunks
- `StreamStartEvent`: Stream initialization
- `StreamCompleteEvent`: Stream completion with final message
- `StreamErrorEvent`: Error handling
- `StreamProgressEvent`: Progress indicators

**Usage**:
```go
// Post streaming update to UI thread
app.screen.PostEvent(tcell.NewEventInterrupt(&MessageChunkEvent{
    StreamID: streamID,
    Content:  chunk.Content,
    Timestamp: time.Now(),
}))
```

### 4. Controller Integration (`pkg/controllers/chat.go`)

**Purpose**: Orchestrates streaming workflow with tool support

**Key Features**:
- Automatic fallback to non-streaming for incompatible clients
- Tool execution integration during streaming
- Conversation state management with rollback on errors
- Error recovery and graceful degradation

**Streaming Flow**:
```go
// Controller checks capabilities and starts appropriate flow
updates, err := controller.StartStreaming(ctx, "user message")
for update := range updates {
    switch update.Type {
    case controllers.StreamStarted:
        // Update UI with streaming indicator
    case controllers.ChunkReceived:
        // Update UI with partial content
    case controllers.MessageComplete:
        // Finalize message in UI
    case controllers.ToolCallDetected:
        // Handle tool execution during streaming
    case controllers.StreamError:
        // Handle errors gracefully
    }
}
```

### 5. TUI Integration (`pkg/tui/app.go`, `pkg/tui/chat_view.go`)

**Purpose**: Provides real-time UI updates during streaming

**Key Features**:
- Non-blocking UI updates via tcell events
- Progress indicators and status updates
- Graceful error display
- Responsive user input during streaming
- Streaming message accumulation display

## Message Flow

The complete streaming flow follows this sequence:

```
1. User submits message
   ↓
2. App optimistically adds user message to UI
   ↓
3. Controller checks if client supports streaming
   ↓
4. If streaming supported:
   - Prepare streaming request with tools
   - Start streaming goroutine
   - Send StreamStartEvent to UI
   ↓
5. For each chunk received:
   - Parse JSON response from HTTP stream
   - Add chunk to accumulator
   - Send MessageChunkEvent to UI
   - Update progress indicators
   ↓
6. On stream completion:
   - Finalize message in accumulator
   - Add complete message to conversation history
   - Send StreamCompleteEvent to UI
   - Handle any tool calls if present
   ↓
7. Update UI with final state and enable input
```

## Ollama Streaming Protocol

### HTTP Streaming Format

The Ollama API uses HTTP streaming with JSON objects sent incrementally:

```json
// Intermediate chunks (streaming)
{
  "model": "llama3.1:8b",
  "created_at": "2023-08-04T08:52:19.385406455-07:00",
  "message": {
    "role": "assistant",
    "content": "The sky",     // Partial content
    "tool_calls": null
  },
  "done": false              // Key indicator: more chunks coming
}

// Final chunk with statistics
{
  "model": "llama3.1:8b", 
  "created_at": "2023-08-04T19:22:45.499127Z",
  "message": {
    "role": "assistant",
    "content": "The sky appears blue due to Rayleigh scattering.",  // Complete content
    "tool_calls": [...]        // Tool calls if present
  },
  "done": true,               // Stream completion indicator
  "total_duration": 4883583458,
  "load_duration": 1334875,
  "prompt_eval_count": 26,
  "prompt_eval_duration": 342546000,
  "eval_count": 282,
  "eval_duration": 4535599000
}
```

### Key Protocol Observations

- **Incremental Content**: Each chunk contains partial `content` that accumulates
- **Completion Signal**: `"done": true` indicates stream completion
- **Error Handling**: HTTP errors or malformed JSON can occur at any point
- **Statistics**: Final chunk includes performance metrics
- **Tool Integration**: Tool calls included in final message

## Thread Safety & Concurrency

### Critical Threading Rules

1. **Single UI Thread**: All UI updates must happen in tcell's main event loop
2. **PostEvent for Communication**: Use `tcell.PostEvent()` to send data to UI thread
3. **No Direct UI Access**: Never call UI methods directly from streaming goroutines
4. **Channel-Based Communication**: Use channels for all goroutine communication

### Safe Streaming Pattern

```go
// CORRECT: Non-blocking streaming with safe UI updates
func (app *App) startStreaming(message string) {
    // 1. Update UI immediately
    app.setStatus("Streaming...")
    
    // 2. Start streaming in background
    go func() {
        updates, err := app.controller.StartStreaming(ctx, message)
        if err != nil {
            app.screen.PostEvent(NewStreamErrorEvent(err))
            return
        }
        
        // 3. Process updates and post to UI thread
        for update := range updates {
            app.screen.PostEvent(NewStreamUpdateEvent(update))
        }
    }()
    // 4. Function returns immediately - UI stays responsive
}

// Handle updates in main event loop only
func (app *App) handleStreamUpdate(update StreamUpdate) {
    switch update.Type {
    case ChunkReceived:
        app.accumulateChunk(update.StreamID, update.Content)
        app.refreshDisplay() // Safe: called from UI thread
    case MessageComplete:
        app.finalizeMessage(update.Message)
        app.setStatus("Ready")
    }
}
```

### State Management

```go
type StreamingState struct {
    isStreaming     bool
    currentStreamID string
    accumulator     *MessageAccumulator
    startTime       time.Time
}

// Prevent concurrent streams
func (app *App) canStartStreaming() bool {
    return !app.streamingState.isStreaming
}
```

## Tool Integration During Streaming

### Tool Call Detection

When tools are called during streaming, the system handles them seamlessly:

```go
// Tool calls detected in streaming response
if message.ToolCalls != nil {
    // Execute tools while maintaining streaming context
    for _, toolCall := range message.ToolCalls {
        result, err := app.toolRegistry.Execute(ctx, toolCall)
        
        // Stream tool results back to user
        app.streamToolResult(toolCall.Name, result)
        
        // Continue conversation with tool results
        app.continueStreamingWithToolResults(toolResults)
    }
}
```

### Streaming Tool Results

Tool execution results are streamed back to provide immediate feedback:

```go
// Show tool execution progress
app.screen.PostEvent(NewToolStartEvent(toolCall.Name))

// Stream tool output as it becomes available
for chunk := range toolOutput {
    app.screen.PostEvent(NewToolChunkEvent(toolCall.Name, chunk))
}

// Show completion
app.screen.PostEvent(NewToolCompleteEvent(toolCall.Name, result))
```

## Error Handling & Recovery

### Network Error Recovery

```go
type StreamingError struct {
    Type    ErrorType
    Message string
    Cause   error
    Retry   bool
}

func (sc *StreamingClient) handleStreamError(err error) StreamingError {
    switch {
    case isNetworkTimeout(err):
        return StreamingError{
            Type:    NetworkTimeout,
            Message: "Connection timed out",
            Cause:   err,
            Retry:   true,
        }
    case isConnectionRefused(err):
        return StreamingError{
            Type:    ConnectionRefused,
            Message: "Cannot connect to Ollama server",
            Cause:   err,
            Retry:   false,
        }
    default:
        return StreamingError{
            Type:    UnknownError,
            Message: err.Error(),
            Cause:   err,
            Retry:   false,
        }
    }
}
```

### Graceful Degradation

If streaming fails, the system automatically falls back to non-streaming mode:

```go
func (controller *ChatController) sendMessage(content string) (Message, error) {
    // Try streaming first
    if streamingClient, ok := controller.client.(StreamingChatClient); ok {
        return controller.tryStreaming(streamingClient, content)
    }
    
    // Fallback to regular request/response
    return controller.regularSend(content)
}

func (controller *ChatController) tryStreaming(client StreamingChatClient, content string) (Message, error) {
    updates, err := controller.StartStreaming(ctx, content)
    if err != nil {
        // Streaming failed, fall back to regular mode
        log.Warn("Streaming failed, falling back to regular API", "error", err)
        return controller.regularSend(content)
    }
    
    // Process streaming updates...
    return controller.processStreamingUpdates(updates)
}
```

## Performance Considerations

### Memory Management

```go
// Efficient chunk accumulation
type MessageAccumulator struct {
    streams map[string]*StreamState
    mu      sync.RWMutex
}

type StreamState struct {
    builder    strings.Builder  // Efficient string building
    chunkCount int
    startTime  time.Time
    lastUpdate time.Time
}

// Cleanup completed streams to prevent memory leaks
func (ma *MessageAccumulator) CleanupStream(streamID string) {
    ma.mu.Lock()
    defer ma.mu.Unlock()
    delete(ma.streams, streamID)
}
```

### Network Efficiency

```go
// Buffered channels prevent blocking
const (
    ChunkBufferSize = 100  // Buffer chunks for smooth processing
    UpdateBufferSize = 50  // Buffer UI updates
)

// Connection reuse
type StreamingClient struct {
    httpClient *http.Client  // Reuses connections
    baseURL    string
}
```

### UI Performance

```go
// Minimize UI updates for better performance
type UIUpdateThrottler struct {
    lastUpdate time.Time
    minInterval time.Duration
}

func (throttler *UIUpdateThrottler) ShouldUpdate() bool {
    now := time.Now()
    if now.Sub(throttler.lastUpdate) >= throttler.minInterval {
        throttler.lastUpdate = now
        return true
    }
    return false
}
```

## Configuration

### Streaming Settings

No additional configuration is required for streaming. It's automatically enabled when:

1. The chat client implements `StreamingChatClient` interface
2. The Ollama server supports streaming (default)
3. The model supports streaming responses

### Fallback Configuration

```yaml
streaming:
  enabled: true
  timeout: "60s"
  chunk_buffer_size: 100
  ui_update_interval: "100ms"
  fallback_on_error: true
  retry_attempts: 3
  retry_delay: "1s"
```

## Testing

### Unit Tests

```go
func TestMessageAccumulator(t *testing.T) {
    acc := chat.NewMessageAccumulator()
    streamID := "test-stream"
    
    // Add chunks
    acc.AddChunk(MessageChunk{
        StreamID: streamID,
        Content:  "Hello ",
        Done:     false,
    })
    
    acc.AddChunk(MessageChunk{
        StreamID: streamID,
        Content:  "world!",
        Done:     true,
    })
    
    // Verify accumulation
    content := acc.GetCurrentContent(streamID)
    assert.Equal(t, "Hello world!", content)
    
    // Verify completion
    assert.True(t, acc.IsComplete(streamID))
    
    message, exists := acc.GetCompleteMessage(streamID)
    assert.True(t, exists)
    assert.Equal(t, "Hello world!", message.Content)
}
```

### Integration Tests

```go
func TestStreamingIntegration(t *testing.T) {
    // Setup mock streaming server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Send streaming JSON chunks
        chunks := []string{
            `{"message":{"content":"Hello"},"done":false}`,
            `{"message":{"content":" world"},"done":false}`,
            `{"message":{"content":"!"},"done":true}`,
        }
        
        for _, chunk := range chunks {
            w.Write([]byte(chunk + "\n"))
            w.(http.Flusher).Flush()
        }
    }))
    defer server.Close()
    
    // Test streaming client
    client := chat.NewStreamingClient(server.URL)
    chunks, err := client.StreamMessage(context.Background(), chatRequest)
    
    assert.NoError(t, err)
    
    var content strings.Builder
    for chunk := range chunks {
        content.WriteString(chunk.Content)
    }
    
    assert.Equal(t, "Hello world!", content.String())
}
```

## Debugging

### Logging

Enable detailed streaming logs:

```yaml
logging:
  level: "debug"
  components:
    - "streaming_client"
    - "message_accumulator"
    - "tui_events"
```

### Common Issues

1. **UI Not Updating During Streaming**
   - Check that events are posted to UI thread via `PostEvent()`
   - Verify event handlers are registered correctly
   - Ensure UI updates happen only in main event loop

2. **Memory Leaks**
   - Verify `CleanupStream()` is called after completion
   - Check for unclosed channels or goroutine leaks
   - Monitor memory usage with `go tool pprof`

3. **Choppy Streaming**
   - Adjust chunk buffer sizes
   - Check network latency and connection quality
   - Verify UI update throttling is appropriate

4. **Tool Calls Not Working During Streaming**
   - Ensure tool registry is available to streaming controller
   - Check tool execution permissions and timeouts
   - Verify tool results are properly integrated into stream

## Future Enhancements

### Planned Features

1. **Enhanced Streaming UX**:
   - Token-by-token display for smoother experience
   - Advanced typing indicators with animation
   - Stream resumption after network interruptions
   - Configurable streaming buffer strategies

2. **Performance Optimizations**:
   - Batch UI updates for efficiency
   - Adaptive chunk processing based on content
   - Connection pooling and keep-alive optimization
   - Memory usage optimization for long streams

3. **Advanced Features**:
   - Stream branching for multiple response options
   - Streaming tool execution with real-time progress
   - Custom streaming renderers and filters
   - Stream analytics and performance monitoring

---

This streaming implementation provides a robust, performant foundation for real-time AI interactions while maintaining the responsiveness and reliability users expect from modern chat interfaces.