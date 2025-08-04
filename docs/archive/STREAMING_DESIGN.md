# Streaming Design & Concurrency Specifications

## Ollama Streaming API Analysis

### HTTP Streaming Behavior
The Ollama API uses HTTP streaming with JSON objects sent incrementally:

```json
// Intermediate chunks (streaming)
{
  "model": "llama3.2",
  "created_at": "2023-08-04T08:52:19.385406455-07:00",
  "message": {
    "role": "assistant",
    "content": "The",     // Partial content
    "images": null
  },
  "done": false           // Key indicator
}

// Final chunk
{
  "model": "llama3.2", 
  "created_at": "2023-08-04T19:22:45.499127Z",
  "message": {
    "role": "assistant",
    "content": "The sky appears blue due to a phenomenon called Rayleigh scattering."  // Complete content
  },
  "done": true,           // Completion indicator
  "total_duration": 4883583458,
  "load_duration": 1334875,
  "prompt_eval_count": 26,
  "prompt_eval_duration": 342546000,
  "eval_count": 282,
  "eval_duration": 4535599000
}
```

### Key Observations
- **Incremental Content**: Each chunk contains partial `content` that must be accumulated
- **Completion Signal**: `"done": true` indicates stream completion
- **Error Handling**: HTTP errors or malformed JSON can occur at any point
- **Statistics**: Final chunk includes performance metrics
- **Single Request**: One HTTP request per message, response body streams

## Go Concurrency Patterns Research

### Channel-Based Communication
Following Go's principle: *"Don't communicate by sharing memory; share memory by communicating"*

#### Pipeline Pattern
```go
// Stage 1: HTTP Reader
httpChunks := readHTTPStream(response.Body)

// Stage 2: JSON Parser  
jsonChunks := parseJSONChunks(httpChunks)

// Stage 3: Message Accumulator
messages := accumulateMessage(jsonChunks)

// Stage 4: UI Updates
uiUpdates := formatForUI(messages)
```

#### Worker Pool Pattern
```go
type WorkerPool struct {
    httpReaders    chan StreamReader
    jsonParsers    chan JSONParser  
    accumulators   chan Accumulator
    uiUpdaters     chan UIUpdater
}
```

#### Fan-Out/Fan-In Pattern
```go
// Fan-out: Distribute chunks to multiple processors
for _, processor := range processors {
    go processor.ProcessChunk(chunkChannel)
}

// Fan-in: Collect results from multiple processors
results := make(chan ProcessedChunk)
for _, processor := range processors {
    go func(p Processor) {
        results <- p.GetResult()
    }(processor)
}
```

### Select Statement for Multiple Channels
```go
for {
    select {
    case chunk := <-streamChannel:
        handleStreamChunk(chunk)
    case input := <-userInputChannel:
        handleUserInput(input)
    case err := <-errorChannel:
        handleError(err)
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## tcell TUI Concurrency Constraints

### Critical Rules from Research
1. **Single UI Thread**: All UI updates must happen in tcell's main event loop
2. **QueueUpdate Pattern**: Use `QueueUpdateDraw()` to safely update UI from goroutines  
3. **No Direct UI Access**: Never call tcell/UI methods directly from streaming goroutines
4. **PostEvent for Communication**: Use `PostEvent()` to send data to UI thread

### Safe UI Update Pattern
```go
// WRONG - Direct UI access from goroutine
go func() {
    for chunk := range streamChannel {
        textView.SetText(chunk.Content)  // RACE CONDITION!
    }
}()

// CORRECT - Queue updates for UI thread
go func() {
    for chunk := range streamChannel {
        app.QueueUpdateDraw(func() {
            textView.SetText(chunk.Content)  // Safe in UI thread
        })
    }
}()
```

### Event Loop Integration
```go
// Main event loop handles both tcell events and custom events
for {
    event := screen.PollEvent()
    switch ev := event.(type) {
    case *tcell.EventKey:
        handleKeyboard(ev)
    case *tcell.EventResize:
        handleResize(ev)
    case *tcell.EventInterrupt:
        if streamUpdate, ok := ev.Data().(*StreamUpdate); ok {
            handleStreamUpdate(streamUpdate)
        }
    }
}
```

## Proposed Threading Architecture

### Goroutine Roles

#### 1. Main Goroutine (UI Thread)
**Purpose**: tcell event loop and all UI rendering  
**Responsibilities**:
- Handle keyboard/mouse/resize events
- Render all UI components  
- Process queued UI updates from other goroutines
- Coordinate application shutdown

**Lifetime**: Entire application lifetime

#### 2. Coordinator Goroutine  
**Purpose**: Central message lifecycle management  
**Responsibilities**:
- Receive user input from UI thread
- Orchestrate streaming requests
- Manage conversation state
- Distribute updates to UI thread
- Handle errors and cleanup

**Lifetime**: Entire application lifetime  

#### 3. Stream Reader Goroutines (per request)
**Purpose**: HTTP streaming and JSON parsing  
**Responsibilities**:
- Read HTTP response body
- Parse JSON chunks
- Send chunks to coordinator
- Handle stream-specific errors
- Clean up on completion/error

**Lifetime**: Per user message (created and destroyed)

#### 4. Input Handler Goroutine
**Purpose**: Pre-process user input  
**Responsibilities**:
- Validate user input
- Handle special commands
- Rate limiting/debouncing
- Send processed input to coordinator

**Lifetime**: Entire application lifetime

### Channel Design

```go
type ChatChannels struct {
    // UI Thread → Coordinator
    UserInput     chan string
    UIEvents      chan UIEvent
    
    // Stream Readers → Coordinator  
    StreamChunks  chan MessageChunk
    StreamErrors  chan StreamError
    StreamDone    chan StreamID
    
    // Coordinator → UI Thread
    UIUpdates     chan UIUpdate
    StatusUpdates chan StatusUpdate
    
    // Coordinator → Stream Readers
    StreamCancel  chan StreamID
    
    // Global
    Shutdown      chan bool
    Errors        chan error
}
```

### Message Flow Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   UI Thread     │    │  Coordinator    │    │ Stream Reader   │
│  (tcell loop)   │    │   Goroutine     │    │   Goroutine     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ UserInput              │                       │
         ├──────────────────────→│                       │
         │                       │ StreamRequest         │
         │                       ├──────────────────────→│
         │                       │                       │
         │                       │            StreamChunk│
         │                       │◄──────────────────────┤
         │                       │                       │
         │            UIUpdate   │                       │
         │◄──────────────────────┤                       │
         │                       │                       │
         │                       │               StreamDone
         │                       │◄──────────────────────┤
         │                       │                       │
         │        StatusUpdate   │                       │
         │◄──────────────────────┤                       │
         │                       │                    [Dies]
```

## Detailed Component Specifications

### MessageChunk Structure
```go
type MessageChunk struct {
    ID        string          // Unique chunk identifier
    Content   string          // Incremental text content
    Done      bool            // Stream completion indicator
    Timestamp time.Time       // When chunk was received
    StreamID  string          // Which stream this belongs to
}
```

### StreamingCoordinator
```go
type StreamingCoordinator struct {
    channels      ChatChannels
    conversation  Conversation
    activeStreams map[string]*StreamReader
    accumulator   MessageAccumulator
}

func (sc *StreamingCoordinator) Run(ctx context.Context) error {
    for {
        select {
        case userInput := <-sc.channels.UserInput:
            sc.handleUserInput(userInput)
            
        case chunk := <-sc.channels.StreamChunks:
            sc.handleStreamChunk(chunk)
            
        case streamID := <-sc.channels.StreamDone:
            sc.handleStreamComplete(streamID)
            
        case err := <-sc.channels.StreamErrors:
            sc.handleStreamError(err)
            
        case <-ctx.Done():
            sc.cleanup()
            return ctx.Err()
        }
    }
}
```

### StreamReader  
```go
type StreamReader struct {
    id       string  
    client   *http.Client
    request  ChatRequest
    chunks   chan<- MessageChunk  
    errors   chan<- StreamError
    done     chan<- string
    cancel   <-chan bool
}

func (sr *StreamReader) Stream(ctx context.Context) error {
    resp, err := sr.client.Do(sr.buildHTTPRequest())
    if err != nil {
        return sr.sendError(err)
    }
    defer resp.Body.Close()
    
    decoder := json.NewDocoder(resp.Body)
    for {
        select {
        case <-sr.cancel:
            return ErrStreamCancelled
        case <-ctx.Done():
            return ctx.Err()
        default:
            var chunk OllamaChunk
            if err := decoder.Decode(&chunk); err != nil {
                if err == io.EOF {
                    sr.done <- sr.id
                    return nil
                }
                return sr.sendError(err)
            }
            
            sr.chunks <- MessageChunk{
                ID:        sr.id,
                Content:   chunk.Message.Content,
                Done:      chunk.Done,
                Timestamp: time.Now(),
                StreamID:  sr.id,
            }
            
            if chunk.Done {
                sr.done <- sr.id
                return nil
            }
        }
    }
}
```

### MessageAccumulator
```go
type MessageAccumulator struct {
    activeMessages map[string]*AccumulatingMessage
}

type AccumulatingMessage struct {
    StreamID     string
    Content      strings.Builder
    ChunkCount   int
    StartTime    time.Time
    LastUpdate   time.Time
}

func (ma *MessageAccumulator) AddChunk(chunk MessageChunk) {
    msg := ma.getOrCreateMessage(chunk.StreamID)
    msg.Content.WriteString(chunk.Content)
    msg.ChunkCount++
    msg.LastUpdate = chunk.Timestamp
    
    if chunk.Done {
        ma.finalizeMessage(chunk.StreamID)
    }
}
```

### UIUpdateManager
```go
type UIUpdate struct {
    Type UpdateType
    Data interface{}
}

type UpdateType int
const (
    MessageChunkUpdate UpdateType = iota
    MessageCompleteUpdate  
    StatusUpdate
    ErrorUpdate
    ConversationUpdate
)

func (uim *UIUpdateManager) SendStreamingUpdate(streamID string, content string) {
    uim.updates <- UIUpdate{
        Type: MessageChunkUpdate,
        Data: StreamingTextUpdate{
            StreamID: streamID,
            Content:  content,
        },
    }
}
```

## Error Handling Strategy

### Error Categories
1. **Network Errors**: Connection failures, timeouts
2. **Protocol Errors**: Invalid JSON, malformed responses  
3. **Application Errors**: Invalid state, resource exhaustion
4. **UI Errors**: Rendering failures, event handling issues

### Error Recovery Patterns
```go
type ErrorRecovery struct {
    MaxRetries    int
    BackoffTime   time.Duration
    RecoveryFunc  func(error) bool
}

func (sc *StreamingCoordinator) handleStreamError(err StreamError) {
    switch err.Type {
    case NetworkError:
        if sc.shouldRetry(err) {
            sc.retryStream(err.StreamID)
        } else {
            sc.notifyUserError(err)
        }
    case ProtocolError:
        sc.logError(err)
        sc.restartStream(err.StreamID)
    case ApplicationError:
        sc.notifyUserError(err)
        sc.shutdown()
    }
}
```

### Graceful Degradation
- **Stream Failure**: Fall back to non-streaming mode
- **Network Issues**: Queue messages for retry
- **Resource Limits**: Limit concurrent streams
- **UI Problems**: Display error state, continue operation

## Performance Considerations

### Memory Management
```go
type ResourceLimits struct {
    MaxMessageHistory    int           // Prevent unbounded growth
    MaxConcurrentStreams int           // Limit active streams  
    StreamTimeout        time.Duration // Prevent hanging streams
    ChunkBufferSize      int           // Channel buffer tuning
}
```

### Channel Buffer Sizing
```go
// High-frequency, low-latency channels
streamChunks := make(chan MessageChunk, 100)

// Low-frequency, user-interaction channels  
userInput := make(chan string, 1)

// Error channels should be unbuffered for immediate handling
errors := make(chan error)
```

### Goroutine Lifecycle Management
```go
type GoroutineManager struct {
    wg       sync.WaitGroup
    cancel   context.CancelFunc  
    shutdown chan bool
}

func (gm *GoroutineManager) StartStreamReader(reader *StreamReader) {
    gm.wg.Add(1)
    go func() {
        defer gm.wg.Done()
        reader.Stream(context.WithCancel(gm.ctx))
    }()
}

func (gm *GoroutineManager) Shutdown() error {
    gm.cancel()                    // Signal cancellation
    gm.shutdown <- true           // Stop coordinator
    
    done := make(chan bool)
    go func() {
        gm.wg.Wait()              // Wait for cleanup
        done <- true
    }()
    
    select {
    case <-done:
        return nil
    case <-time.After(5 * time.Second):
        return ErrShutdownTimeout
    }
}
```

## Testing Strategy for Concurrent Code

### Unit Testing Channels
```go
func TestStreamingCoordinator(t *testing.T) {
    channels := ChatChannels{
        UserInput:    make(chan string, 1),
        StreamChunks: make(chan MessageChunk, 10),
        UIUpdates:    make(chan UIUpdate, 10),
    }
    
    coordinator := NewStreamingCoordinator(channels)
    
    // Test input handling
    channels.UserInput <- "Hello"
    
    // Verify output  
    select {
    case update := <-channels.UIUpdates:
        assert.Equal(t, MessageChunkUpdate, update.Type)
    case <-time.After(100 * time.Millisecond):
        t.Error("Expected UI update not received")
    }
}
```

### Race Condition Testing
```go
func TestConcurrentMessageAccumulation(t *testing.T) {
    accumulator := NewMessageAccumulator()
    
    // Simulate concurrent chunk arrivals
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(chunkNum int) {
            defer wg.Done()
            chunk := MessageChunk{
                StreamID: "test-stream",
                Content:  fmt.Sprintf("chunk-%d ", chunkNum),
            }
            accumulator.AddChunk(chunk)
        }(i)
    }
    
    wg.Wait()
    
    // Verify all chunks accumulated correctly
    message := accumulator.GetMessage("test-stream")
    assert.Contains(t, message.Content, "chunk-")
}
```

### Deadlock Prevention Testing
```go
func TestNoDeadlocks(t *testing.T) {
    done := make(chan bool)
    
    go func() {
        // Run the system
        coordinator.Run(context.Background())
        done <- true
    }()
    
    // Send shutdown signal
    coordinator.Shutdown()
    
    // Verify shutdown completes
    select {
    case <-done:
        // Success
    case <-time.After(2 * time.Second):
        t.Error("Deadlock detected - shutdown did not complete")
    }
}
```

## Implementation Checkpoints

### Phase 3 Validation (Streaming Infrastructure)
- [ ] HTTP streaming client works with mock responses
- [ ] JSON chunk parsing handles all edge cases
- [ ] Message accumulation is correct and race-free  
- [ ] Error handling covers all failure modes
- [ ] All goroutines clean up properly
- [ ] No memory leaks in long-running tests

### Phase 4 Validation (TUI Integration)
- [ ] UI updates arrive in correct order
- [ ] No race conditions in UI state
- [ ] User input works during streaming
- [ ] Error states display correctly
- [ ] Terminal resizing doesn't break streaming
- [ ] All channel communication is deadlock-free

---

*This design document captures our research findings and serves as the implementation specification for streaming functionality*