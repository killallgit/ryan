# Chat TUI Architecture

## System Overview

This project implements an interactive command-line chat interface for Claude-like AI assistants, starting with Ollama integration. The system prioritizes responsiveness, clean architecture, and maintainability through functional programming principles.

### Core Goals
- **Responsive Streaming**: Real-time message streaming without UI blocking
- **Clean Separation**: TUI, business logic, and API layers remain independent  
- **Testable Design**: Every component can be tested in isolation
- **Functional Approach**: Immutable data, pure functions, channel-based communication
- **Incremental Complexity**: Build simple, add complexity gradually

## Architecture Principles

### 1. Functional Programming First
- Prefer pure functions over stateful objects
- Use immutable data structures
- Channel-based communication over shared memory
- Composition over inheritance (avoid OOP patterns)

### 2. Separation of Concerns
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TUI Layer     │    │  Controller     │    │   Chat Core     │
│   (tcell)       │────│    Layer        │────│   (business)    │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
   UI Events              Orchestration            API Calls
   Rendering              State Management         Message Logic
```

### 3. Threading Model
- **Main Thread**: tcell event loop + UI rendering
- **Coordinator Thread**: Message lifecycle management
- **Stream Readers**: HTTP streaming (created per request)
- **Input Handlers**: User input processing

## Component Architecture

### Core Components

#### 1. Chat Domain (`pkg/chat/`)
**Purpose**: Core business logic, API-agnostic message handling

```go
// messages.go - Pure data structures
type Message struct {
    Role      string    // "user" | "assistant" | "system"
    Content   string
    Timestamp time.Time
}

type Conversation struct {
    Messages []Message
    Model    string
}

// Pure functions - no side effects
func AddMessage(conv Conversation, msg Message) Conversation
func GetLastMessage(conv Conversation) (Message, bool)
```

#### 2. Ollama Integration (`pkg/ollama/`)
**Purpose**: HTTP client for Ollama API (existing + streaming extensions)

```go
// Extended from existing client
type StreamingClient struct {
    *Client  // Embed existing client
}

func (sc *StreamingClient) StreamChat(req ChatRequest) (<-chan MessageChunk, error)
```

#### 3. TUI Components (`pkg/tui/`)
**Purpose**: Terminal interface using tcell

```go
// Stateless rendering components
type MessageDisplay struct {
    Messages []chat.Message
    Width    int
    Height   int
}

type AlertDisplay struct {
    IsSpinnerVisible bool
    SpinnerFrame     int
    SpinnerText      string
    ErrorMessage     string
    Width            int
}

func RenderMessages(screen tcell.Screen, display MessageDisplay, area Rect)
func RenderInput(screen tcell.Screen, input string, area Rect)
func RenderAlert(screen tcell.Screen, alert AlertDisplay, area Rect)
```

#### 4. Controllers (`pkg/controllers/`)
**Purpose**: Orchestrate between TUI and chat domain

```go
type ChatController struct {
    client       ChatClient
    conversation chat.Conversation
}

// Synchronous version (Phase 1)
func (cc *ChatController) SendMessage(content string) (chat.Message, error)

// Streaming version (Phase 4)
func (cc *ChatController) StartStream(content string) (<-chan StreamingUpdate, error)
```

## Data Flow Patterns

### Phase 1: Synchronous Flow (Simple)
```
User Input → TUI → Controller → Ollama Client → Response → TUI Update
     ↑                                                         │
     └─────────── Single Thread (tcell event loop) ───────────┘
```

### Phase 4: Streaming Flow (Complex)
```
User Input ────┐
               ▼
         Coordinator ──→ Stream Reader ──→ Message Chunks
               │              │                    │
               ▼              ▼                    ▼
         TUI Updates ◄─── Channel Hub ◄─── Accumulator
               │
               ▼
         tcell.QueueUpdate() ──→ UI Rendering
```

## Technology Stack

### Core Dependencies
- **tcell/v2**: Terminal control and event handling
- **cobra**: Command-line interface
- **viper**: Configuration management
- **ginkgo/gomega**: Testing framework
- **testify**: Mocking and assertions

### Standard Library Usage
- **net/http**: HTTP client for Ollama API
- **encoding/json**: JSON parsing for API responses
- **context**: Request cancellation and timeouts
- **sync**: Channel coordination (minimal use)

## Concurrency Model

### Channel Design
```go
type StreamingChannels struct {
    UserInput    chan string           // UI → Coordinator
    StreamChunks chan MessageChunk     // Stream Reader → Coordinator  
    UIUpdates    chan UIUpdateEvent    // Coordinator → UI
    Errors       chan error            // Any → Coordinator
    Done         chan bool             // Coordinator → All
}
```

### Goroutine Lifecycle
1. **Main Goroutine**: tcell event loop (never blocks)
2. **Coordinator**: Created at app start, lives for app lifetime
3. **Stream Readers**: Created per user message, die when stream completes
4. **Input Handler**: Created at app start, lives for app lifetime

### Synchronization Rules
- **No Mutexes**: Use channels for all communication
- **No Shared State**: Immutable data structures only  
- **UI Thread Safety**: Only update UI via tcell.QueueUpdate()
- **Clean Shutdown**: All goroutines must be cancellable via context

## Testing Strategy

### Unit Testing
- **Pure Functions**: Test all business logic in isolation
- **Mock Dependencies**: HTTP clients, TUI components
- **Property Testing**: Message handling correctness

### Integration Testing  
- **Component Interaction**: Controller + Client integration
- **Real API Testing**: Optional integration with running Ollama
- **TUI Simulation**: Event injection and screen capture

### Concurrency Testing
- **Race Detection**: Run tests with `-race` flag
- **Deadlock Prevention**: Timeout-based tests
- **Channel Behavior**: Test all channel communication patterns

## Decision Records

### Why tcell over other TUI libraries?
- **Low-level control**: Direct terminal manipulation
- **Event model**: Clean separation of input/output
- **Streaming friendly**: Non-blocking event loop
- **Mature**: Battle-tested, good documentation

### Why channels over mutexes?
- **Go idiom**: "Don't communicate by sharing memory"
- **Testability**: Easier to test channel protocols
- **Deadlock prevention**: Channels compose better
- **Clarity**: Data flow is explicit and visible

### Why functional approach?
- **Predictability**: Pure functions are easier to reason about
- **Testability**: No hidden state to mock
- **Concurrency safety**: Immutable data eliminates races
- **Maintainability**: Clear input/output contracts

## Performance Considerations

### Memory Management
- **Message History Limits**: Prevent unlimited growth
- **Goroutine Cleanup**: Ensure no leaks on stream completion
- **Buffer Sizes**: Tune channel buffers for throughput

### Response Time Requirements
- **UI Responsiveness**: < 16ms for 60fps feel
- **Stream Latency**: Display chunks within 100ms of receipt
- **Input Lag**: User input processed within 50ms

### Scalability Limits
- **Single User**: Designed for one user, one conversation
- **Message Volume**: Handle conversations up to 1000 messages
- **Concurrent Streams**: Only one active stream at a time

## Current Implementation Status

### Phase 1: Foundation (COMPLETED)
- ✓ Core message and conversation types implemented
- ✓ Synchronous HTTP client for Ollama API
- ✓ Basic chat controller with conversation management
- ✓ TUI foundation with basic rendering
- ✓ Integration tests with real Ollama API
- ✓ Configuration management with Viper

### Phase 2: Non-Blocking TUI (COMPLETED)
- ✓ Non-blocking message sending with goroutines
- ✓ Custom event system for API responses
- ✓ Responsive UI during API calls
- ✓ State management to prevent multiple concurrent requests
- ✓ Thread-safe communication via tcell's PostEvent
- ✓ AlertDisplay component for spinner and error feedback
- ✓ Progress tracking with elapsed time display
- ✓ Enhanced error handling with base16 red colors
- ✓ Ollama connectivity checking
- ✓ Escape key cancellation support

#### Non-Blocking Implementation Details
The Phase 2 implementation solved a critical UX issue where the TUI would freeze during API calls:

**Problem**: Synchronous `controller.SendUserMessage()` blocked the main event loop
**Solution**: Event-driven architecture with goroutines

```go
// Custom event types for thread-safe communication
type MessageResponseEvent struct {
    tcell.EventTime
    Message chat.Message
}

// Non-blocking flow
User Input → sendMessage() → goroutine → API call → PostEvent → UI update
     ↑                                                              │  
     └───────────── UI stays responsive ──────────────────────────┘
```

**Key Benefits**:
- Users can scroll and navigate during API calls
- No UI blocking or freezing
- Clean error handling with visual feedback
- Immediate spinner visibility with progress tracking
- Enhanced error display with base16 red colors in dedicated alert area
- Ollama connectivity validation with actionable error messages
- Escape key cancellation for long-running operations
- Foundation ready for streaming integration

### Ready for Phase 3: Streaming
The foundation is now solid with:
- Working chat functionality verified against real Ollama deployment
- Non-blocking TUI that stays responsive during API calls
- Clean architecture with proper separation of concerns  
- Comprehensive test coverage (unit and integration)
- Thread-safe event-driven architecture ready for streaming

---

*This document is living - updated as we learn and evolve the architecture*