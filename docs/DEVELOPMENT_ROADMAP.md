# Chat TUI Development Roadmap

## Development Philosophy

### Incremental Complexity
- Start with the simplest possible working solution
- Add one complex piece at a time
- Test thoroughly at each increment
- Never add complexity until the current layer is solid

### Test-Driven Development
- Write tests before implementation
- Every function must be testable in isolation
- Integration tests validate component interactions
- Concurrency tests prevent race conditions and deadlocks

### Clean Architecture Progression
```
Phase 1: Simple → Phase 2: UI → Phase 3: Streaming → Phase 4: Integration → Phase 5: Polish
   ↓                ↓              ↓                  ↓                    ↓
Pure Functions   Basic TUI    Isolated Streams   Concurrent Integration  Production Ready
```

## Phase 1: Foundation - Non-Streaming Chat Core ✓ COMPLETED
**Duration**: 1 week  
**Goal**: Basic request/response chat without streaming complexity

### 1.1: Message Types & Core Domain (`pkg/chat/`)
**Files**: `messages.go`, `conversation.go`

```go
type Message struct {
    Role      string    // "user" | "assistant" | "system"
    Content   string
    Timestamp time.Time
}

type Conversation struct {
    Messages []Message
    Model    string
}

// Pure functions - no side effects, fully testable
func AddMessage(conv Conversation, msg Message) Conversation
func GetMessages(conv Conversation) []Message
func GetLastAssistantMessage(conv Conversation) (Message, bool)
func GetMessageCount(conv Conversation) int
```

**Tests Required**:
- Message creation and validation
- Conversation manipulation
- Message retrieval functions
- Edge cases (empty conversations, invalid messages)

**Exit Criteria**: All pure functions tested, conversation logic works perfectly

### 1.2: Simple HTTP Client (`pkg/chat/client.go`)
**Purpose**: Synchronous HTTP client - NO streaming yet

```go
type Client struct {
    baseURL string
    client  *http.Client
}

type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    Stream   bool      `json:"stream"` // Always false in Phase 1
}

// Blocks until complete response received
func (c *Client) SendMessage(req ChatRequest) (Message, error)
```

**Tests Required**:
- Mock HTTP responses
- Error handling (network failures, API errors)
- Request/response serialization
- Timeout behavior

**Exit Criteria**: Can successfully communicate with Ollama API (non-streaming mode)

### 1.3: Basic Controller (`pkg/controllers/chat.go`)
**Purpose**: Simple request-response orchestration

```go
type ChatController struct {
    client       ChatClient      // Interface for testing
    conversation Conversation
}

func NewChatController(client ChatClient, model string) *ChatController

// Simple synchronous operation
func (cc *ChatController) SendUserMessage(content string) (Message, error) {
    // 1. Add user message to conversation
    // 2. Send request to client
    // 3. Add assistant response to conversation
    // 4. Return assistant message
}

func (cc *ChatController) GetHistory() []Message
```

**Tests Required**:
- Mock chat client
- Conversation state management
- Error propagation
- Message ordering

**Exit Criteria**: Controller manages conversation state correctly, handles errors gracefully

### 1.4: Command Integration (`cmd/chat.go`)
**Purpose**: Basic command to test the foundation

```go
var chatCmd = &cobra.Command{
    Use:   "chat",
    Short: "Start a simple chat (non-streaming)",
    Run: func(cmd *cobra.Command, args []string) {
        client := chat.NewClient(viper.GetString("ollama.url"))
        controller := controllers.NewChatController(client, "llama3.1:8b")
        
        // Simple CLI loop for testing
        for {
            fmt.Print("> ")
            input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
            response, err := controller.SendUserMessage(strings.TrimSpace(input))
            if err != nil {
                fmt.Printf("Error: %v\n", err)
                continue
            }
            fmt.Printf("Assistant: %s\n", response.Content)
        }
    },
}
```

**Tests Required**:
- Command registration
- Configuration handling
- Basic integration test

**Exit Criteria**: Can run `ryan chat` and have a basic conversation ✓ COMPLETED

### Phase 1 Completion Notes:
- Basic chat functionality implemented and working with Ollama API
- TUI mode available as the sole interface (run `ryan` to start)
- Integration tests created for real API testing
- Task commands added: `task test:integration`, `task test:all`
- App is now TUI-focused with no CLI chat interface

## Phase 2: Simple TUI Foundation ✓ COMPLETED
**Duration**: 1 week  
**Goal**: Basic TUI that works with synchronous chat - Extended to non-blocking

### 2.1: Basic TUI Components (`pkg/tui/`)
**Files**: `components.go`, `layout.go`, `render.go`

```go
// Stateless rendering components
type MessageDisplay struct {
    Messages []chat.Message
    Width    int
    Height   int
    Scroll   int  // Current scroll position
}

type InputField struct {
    Content string
    Cursor  int
    Width   int
}

type StatusBar struct {
    Model     string
    Status    string // "Ready", "Connecting", "Error"
    Width     int
}

// Pure rendering functions
func RenderMessages(screen tcell.Screen, display MessageDisplay, area Rect)
func RenderInput(screen tcell.Screen, input InputField, area Rect)
func RenderStatus(screen tcell.Screen, status StatusBar, area Rect)
```

**Tests Required**:
- Rendering component tests (screen capture/comparison)
- Layout calculations
- Text wrapping and scrolling
- Unicode handling

**Exit Criteria**: Components render correctly in isolation

### 2.2: Basic Event Loop (`pkg/tui/app.go`)
**Purpose**: Simple, single-threaded TUI application

```go
type App struct {
    screen       tcell.Screen
    controller   *controllers.ChatController
    input        string
    messages     []chat.Message
    status       string
    quit         bool
}

func NewApp(controller *controllers.ChatController) (*App, error)

// Single-threaded event loop - no goroutines yet
func (a *App) Run() error {
    for !a.quit {
        a.render()
        event := a.screen.PollEvent()
        a.handleEvent(event)
    }
}

func (a *App) handleKeyEvent(ev *tcell.EventKey) {
    switch ev.Key() {
    case tcell.KeyEnter:
        a.sendMessage()
    case tcell.KeyEscape:
        a.quit = true
    default:
        a.handleTextInput(ev.Rune())
    }
}

func (a *App) sendMessage() {
    // This will block the UI - that's OK for Phase 2
    response, err := a.controller.SendUserMessage(a.input)
    if err != nil {
        a.status = fmt.Sprintf("Error: %v", err)
        return
    }
    a.messages = a.controller.GetHistory()
    a.input = ""
}
```

**Tests Required**:
- Event handling simulation
- Keyboard input processing
- Screen state management
- Error display

**Exit Criteria**: Functional chat TUI with basic keyboard interaction

### 2.3: Layout Management (`pkg/tui/layout.go`)
**Purpose**: Responsive layout calculations

```go
type Rect struct {
    X, Y, Width, Height int
}

type Layout struct {
    Screen Rect
    Messages Rect
    Input Rect
    Status Rect
}

func CalculateLayout(screenWidth, screenHeight int) Layout {
    // Messages: Top 80% of screen
    // Input: Next to bottom line
    // Status: Bottom line
}

func (l Layout) HandleResize(newWidth, newHeight int) Layout
```

**Tests Required**:
- Layout calculations for various screen sizes
- Resize handling
- Minimum size constraints

**Exit Criteria**: TUI adapts correctly to terminal resizing ✓ COMPLETED

### 2.4: Non-Blocking Implementation (`pkg/tui/events.go`, `pkg/tui/app.go`)
**Purpose**: Prevent UI blocking during API calls - IMPLEMENTED BEYOND SPEC

```go
// Custom events for non-blocking communication
type MessageResponseEvent struct {
    tcell.EventTime
    Message chat.Message
}

type MessageErrorEvent struct {
    tcell.EventTime
    Error error
}

// Non-blocking sendMessage implementation
func (app *App) sendMessage() {
    // Clear input immediately and set sending state
    app.input = app.input.Clear()
    app.sending = true
    app.status = app.status.WithStatus("Sending...")
    
    // Send in goroutine to avoid blocking UI
    go func() {
        response, err := app.controller.SendUserMessage(content)
        if err != nil {
            app.screen.PostEvent(NewMessageErrorEvent(err))
        } else {
            app.screen.PostEvent(NewMessageResponseEvent(response))
        }
    }()
}
```

**Exit Criteria**: TUI remains responsive during API calls ✓ COMPLETED

### Phase 2 Completion Notes:
- Basic TUI components fully implemented and working
- Layout management and responsive resizing ✓
- Event handling and keyboard navigation ✓
- **NON-BLOCKING UI**: Extended beyond original spec to solve UI blocking
- Custom event system for thread-safe API communication
- State management prevents multiple concurrent requests
- **ENHANCED UX**: Significant improvements beyond specification:
  - AlertDisplay component with dedicated alert area
  - Immediate spinner visibility with state synchronization
  - Progress feedback with elapsed time tracking
  - Base16 red error colors in alert area
  - Ollama connectivity validation with specific error guidance
  - Escape key cancellation for long-running operations
  - Enhanced error messages with actionable suggestions
- Ready for streaming integration in Phase 3

## Phase 3: Streaming Infrastructure
**Duration**: 1 week  
**Goal**: Add streaming without TUI complexity

### 3.1: Streaming Client (`pkg/chat/stream.go`)
**Purpose**: HTTP streaming implementation, testable in isolation

```go
type StreamingClient struct {
    baseURL string
    client  *http.Client
}

type MessageChunk struct {
    Content   string
    Done      bool
    Timestamp time.Time
    Error     error
}

// Returns channels - testable with mock responses
func (sc *StreamingClient) StreamMessage(req ChatRequest) (<-chan MessageChunk, error) {
    // HTTP streaming implementation
    // Parse JSON chunks from response body
    // Handle connection errors gracefully
}

// For testing - inject mock HTTP response
func (sc *StreamingClient) WithHTTPClient(client *http.Client) *StreamingClient
```

**Tests Required**:
- Mock HTTP streaming responses
- JSON chunk parsing
- Error handling (connection drops, malformed JSON)
- Proper channel closure

**Exit Criteria**: Streaming client works perfectly in isolation

### 3.2: Message Accumulator (`pkg/chat/accumulator.go`)
**Purpose**: Assemble streaming chunks into complete messages

```go
type Accumulator struct {
    chunks    []MessageChunk
    content   strings.Builder
    startTime time.Time
}

// Pure functions - no side effects
func NewAccumulator() Accumulator
func (a Accumulator) AddChunk(chunk MessageChunk) Accumulator
func (a Accumulator) GetCurrentContent() string
func (a Accumulator) IsComplete() bool
func (a Accumulator) GetCompleteMessage() (Message, bool)
```

**Tests Required**:
- Chunk accumulation correctness
- Unicode handling across chunk boundaries
- Performance with large messages
- Edge cases (empty chunks, out-of-order chunks)

**Exit Criteria**: Message assembly is correct and efficient

### 3.3: Streaming Controller (`pkg/controllers/stream.go`)
**Purpose**: Manage streaming operations

```go
type StreamingController struct {
    client       StreamingClient
    conversation Conversation
}

type StreamingUpdate struct {
    Type    UpdateType // ChunkReceived, MessageComplete, Error
    Chunk   MessageChunk
    Message Message
    Error   error
}

// Returns channels for streaming updates
func (sc *StreamingController) StartStreaming(userMessage string) (
    <-chan StreamingUpdate,
    error,
) {
    // 1. Add user message to conversation
    // 2. Start streaming from client
    // 3. Accumulate chunks
    // 4. Send updates via channel
    // 5. Add complete message to conversation
}
```

**Tests Required**:
- Mock streaming client
- Update channel behavior
- Error propagation
- Conversation state consistency

**Exit Criteria**: Streaming controller manages message lifecycle correctly

### 3.4: CLI Testing Tool (`cmd/stream_test.go`)
**Purpose**: Test streaming without TUI complexity

```go
// Hidden command for development testing
var streamTestCmd = &cobra.Command{
    Use:    "stream-test",
    Hidden: true,
    Run: func(cmd *cobra.Command, args []string) {
        // Simple CLI that shows streaming in action
        // Used to validate streaming before TUI integration
    },
}
```

**Exit Criteria**: Can see streaming work in simple CLI environment

### Phase 3 Completion Notes: ✓ COMPLETED
- **Streaming Client**: Full HTTP streaming implementation with chunk processing ✓
- **Message Accumulator**: Thread-safe message assembly with Unicode handling ✓  
- **Controller Integration**: Streaming support with tool execution and fallback ✓
- **TUI Event System**: Complete streaming event types and handlers ✓
- **Comprehensive Testing**: Unit tests for all streaming components ✓
- **Architecture Enhancements**: Exceeded original spec with:
  - Advanced stream statistics and progress tracking
  - Automatic fallback for non-streaming clients
  - Memory-efficient chunk accumulation
  - Robust error handling and recovery
  - Tool system integration during streaming
  - Thread-safe UI updates via tcell events
- **Performance Optimizations**: Buffered channels, efficient string building, proper cleanup
- **Documentation**: Complete implementation guide and API reference
- Ready for production use with real-time streaming experience

## Phase 4: TUI + Streaming Integration
**Duration**: 1 week  
**Goal**: Carefully integrate streaming into TUI

### 4.1: Update Channels (`pkg/tui/updates.go`)
**Purpose**: Clean abstraction for TUI updates

```go
type UIUpdate struct {
    Type    UIUpdateType
    Payload interface{}
}

type UIUpdateType int
const (
    MessageChunkUpdate UIUpdateType = iota
    MessageCompleteUpdate
    ErrorUpdate
    StatusUpdate
)

// Convert streaming events to UI updates
func StreamToUIUpdates(
    streamUpdates <-chan controllers.StreamingUpdate,
    uiUpdates chan<- UIUpdate,
    done <-chan bool,
) {
    // Goroutine that transforms streaming events to UI events
    // Handles cleanup and shutdown properly
}
```

**Tests Required**:
- Channel transformation correctness
- Proper goroutine cleanup
- Backpressure handling

**Exit Criteria**: Clean streaming → UI update transformation

### 4.2: Enhanced TUI App (`pkg/tui/streaming_app.go`)
**Purpose**: TUI app with streaming support

```go
type StreamingApp struct {
    screen             tcell.Screen
    streamController   *controllers.StreamingController
    messages           []chat.Message
    currentInput       string
    currentStream      string  // Accumulating stream content
    isStreaming        bool
    uiUpdates         chan UIUpdate
    quit              chan bool
}

func (sa *StreamingApp) Run() error {
    go sa.handleUIUpdates()  // Separate goroutine for updates
    
    for {
        sa.render()
        select {
        case event := <-sa.eventChannel():
            sa.handleEvent(event)
        case <-sa.quit:
            return nil
        }
    }
}

// Critical: Use tcell's QueueUpdate for thread safety
func (sa *StreamingApp) handleUIUpdates() {
    for update := range sa.uiUpdates {
        sa.screen.PostEvent(tcell.NewEventInterrupt(update))
    }
}
```

**Tests Required**:
- Concurrent UI update handling
- Event processing during streaming
- Proper goroutine lifecycle
- Race condition testing

**Exit Criteria**: User can type while messages stream

### 4.3: Safe UI Updates (`pkg/tui/safety.go`)
**Purpose**: Ensure thread-safe UI operations

```go
// Wrapper functions that ensure UI updates happen safely
func (sa *StreamingApp) safeUpdateMessages(messages []chat.Message) {
    sa.screen.PostEvent(tcell.NewEventInterrupt(&UIUpdate{
        Type: MessageCompleteUpdate,
        Payload: messages,
    }))
}

func (sa *StreamingApp) safeUpdateStreamingText(content string) {
    sa.screen.PostEvent(tcell.NewEventInterrupt(&UIUpdate{
        Type: MessageChunkUpdate,
        Payload: content,
    }))
}
```

**Tests Required**:
- Thread safety verification
- UI consistency during updates
- No race conditions in screen updates

**Exit Criteria**: All UI updates are thread-safe

### 4.4: Integration Testing
**Purpose**: Validate the complete streaming TUI system

**Tests Required**:
- End-to-end streaming scenarios
- Concurrent user interaction during streaming
- Error recovery during streaming
- Network failure handling

**Exit Criteria**: Streaming TUI works reliably under all conditions

## Phase 5: Polish & Production Readiness
**Duration**: 1 week  
**Goal**: Production-ready features and robustness

### 5.1: Advanced UI Features
- **Typing Indicators**: Show when assistant is "thinking"
- **Message Timestamps**: Display message times
- **Scroll Handling**: Proper message history scrolling
- **Text Wrapping**: Smart word wrapping for long messages
- **Syntax Highlighting**: Basic markdown rendering

### 5.2: Error Handling & Recovery
- **Network Failures**: Graceful degradation and retry logic
- **Stream Interruption**: Recovery from broken connections
- **Invalid Responses**: Handle malformed API responses
- **Resource Exhaustion**: Memory and goroutine limits

### 5.3: Performance Optimization
- **Message History Limits**: Prevent memory growth
- **Efficient Rendering**: Minimal screen updates
- **Goroutine Cleanup**: No resource leaks
- **Response Time**: Sub-100ms UI responsiveness

### 5.4: Configuration & Customization
- **Model Selection**: Easy model switching
- **Theme Support**: Color schemes and styling
- **Keyboard Shortcuts**: Customizable key bindings
- **Export Options**: Save conversation history

## Testing Strategy by Phase

### Phase 1-2: Foundation Testing
- **Unit Tests**: 100% coverage of pure functions
- **Integration Tests**: Component interaction validation
- **Mock Testing**: Isolate external dependencies

### Phase 3-4: Concurrency Testing
- **Race Detection**: All tests run with `-race` flag
- **Deadlock Prevention**: Timeout-based validation
- **Channel Testing**: Verify all communication patterns
- **Load Testing**: High-frequency message scenarios

### Phase 5: Production Testing
- **Stress Testing**: Long conversations, network issues
- **Memory Leak Detection**: Extended runtime validation
- **Cross-Platform Testing**: Multiple terminal types
- **User Acceptance Testing**: Real-world usage scenarios

## Success Metrics

### Phase 1: Foundation ✅ ACHIEVED
- [x] All unit tests pass
- [x] Can have basic conversation (TUI-only)
- [x] Clean, testable architecture

### Phase 2: Basic TUI ✅ ACHIEVED
- [x] Functional chat interface
- [x] Responsive to terminal resizing
- [x] Clean event handling
- [x] Non-blocking UI during API calls
- [x] Thread-safe event communication
- [x] AlertDisplay component with spinner and error feedback
- [x] Progress tracking with elapsed time display
- [x] Enhanced error messages with base16 red colors
- [x] Ollama connectivity validation
- [x] Escape key cancellation support

### Phase 3: Streaming
- [ ] Streaming works in isolation
- [ ] Proper error handling
- [ ] No resource leaks

### Phase 4: Integration
- [ ] Real-time streaming in TUI
- [ ] Responsive UI during streaming
- [ ] Thread-safe operations

### Phase 5: Production
- [ ] Sub-100ms UI response time
- [ ] Handles 1000+ message conversations
- [ ] Graceful error recovery
- [ ] Memory usage stays bounded

## Risk Mitigation

### Technical Risks
- **Concurrency Bugs**: Comprehensive testing with race detection
- **Memory Leaks**: Regular profiling and leak detection
- **UI Blocking**: Strict separation of UI and business logic

### Architecture Risks
- **Complexity Creep**: Strict phase boundaries and simple-first approach
- **Over-Engineering**: Build only what's needed for current phase
- **Testing Debt**: Test-first methodology mandatory

### Timeline Risks
- **Scope Creep**: Clear phase exit criteria
- **Integration Issues**: Early and frequent integration testing
- **Performance Problems**: Regular benchmarking and profiling

---

*This roadmap is living - updated as we progress and learn*