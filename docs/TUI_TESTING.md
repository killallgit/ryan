# TUI Testing Pattern with SimulationScreen

This document describes the testing pattern for TUI components using tcell's SimulationScreen, specifically focused on testing chat streaming functionality.

## Overview

tcell provides a `SimulationScreen` that allows us to test TUI components without requiring an actual terminal. This is particularly useful for testing streaming behavior, user interactions, and UI updates.

## Core Components

### 1. Test Helpers (`test_helpers.go`)

We've created a set of test helpers that extend SimulationScreen's functionality:

#### TestScreen
```go
type TestScreen struct {
    tcell.SimulationScreen
    mu            sync.Mutex
    eventHistory  []tcell.Event
    screenHistory []string
}
```

Key features:
- `CaptureContent()` - Captures the current screen content as a string
- `PostEventAndWait()` - Posts an event and waits for processing
- `FindInContent()` - Searches for text in the current screen
- `GetRegion()` - Extracts content from a specific screen region

#### StreamingTestHelper
```go
type StreamingTestHelper struct {
    screen      *TestScreen
    Chunks      chan string
    Errors      chan error
    done        chan struct{}
    wg          sync.WaitGroup
}
```

Provides utilities for simulating streaming scenarios:
- `SimulateStreaming()` - Simulates chunks arriving over time
- `GetNextChunk()` - Gets the next chunk with timeout support
- `Stop()` - Cleanly stops the streaming simulation

### 2. Mock Controllers

For testing streaming behavior, we create mock controllers that simulate the chat backend:

```go
type MockStreamingController struct {
    model           string
    messages        []chat.Message
    streamChannel   chan string
    errorChannel    chan error
    stopChannel     chan struct{}
    streamingActive bool
    mu              sync.Mutex
}
```

Key methods:
- `SendUserMessage()` - Simulates sending a message and starting streaming
- `simulateStreaming()` - Runs in a goroutine to simulate chunk delivery
- `StopStreaming()` - Safely stops streaming with proper channel management
- `Reset()` - Resets the controller for reuse in tests

### 3. Testing Patterns

#### Basic Screen Interaction
```go
// Write content directly to screen
for i, ch := range "hello" {
    screen.SetContent(i, 22, ch, nil, tcell.StyleDefault)
}
screen.Show()

// Capture and verify content
content := captureScreenContent(screen)
Expect(content).To(ContainSubstring("hello"))
```

#### Streaming Simulation
```go
// Simulate streaming chunks
chunks := []string{"<think>", "Processing...", "</think>", "Response"}
helper.SimulateStreaming(chunks, delays)

// Verify chunks appear progressively
for i := 0; i < len(chunks); i++ {
    chunk, ok := helper.GetNextChunk(100 * time.Millisecond)
    Expect(ok).To(BeTrue())
    receivedChunks = append(receivedChunks, chunk)
}
```

#### Stream Parser Testing
```go
parser := tui.NewStreamParser()

// Test partial tag handling
chunks := []string{"<thi", "nk>", "Content", "</think>"}
for _, chunk := range chunks {
    segments := parser.ParseChunk(chunk)
    // Verify formatting and content
}
```

## Test Organization

### 1. Unit Tests
- Test individual components in isolation
- Focus on specific behaviors (e.g., parser handling partial tags)
- Use table-driven tests for multiple scenarios

### 2. Integration Tests
- Test component interactions
- Simulate realistic streaming scenarios
- Verify UI updates during streaming

### 3. Performance Tests
- Test resource management (goroutines, memory)
- Verify no leaks during long streaming sessions
- Test concurrent streaming scenarios

## Best Practices

### 1. Screen Setup
```go
BeforeEach(func() {
    screen = tcell.NewSimulationScreen("UTF-8")
    err := screen.Init()
    Expect(err).ToNot(HaveOccurred())
    screen.SetSize(80, 24)
    screen.Clear()
})

AfterEach(func() {
    screen.Fini()
})
```

### 2. Resource Management
- Always use proper cleanup in AfterEach
- Use contexts for cancellation
- Prevent channel panics with safe closing patterns

### 3. Timing and Synchronization
- Use Eventually() for async assertions
- Add appropriate delays for streaming simulation
- Consider race conditions in concurrent tests

### 4. Content Verification
- Use screen regions for targeted verification
- Capture screen history for debugging
- Test both visible content and styling

## Common Patterns

### Testing UI Responsiveness During Streaming
```go
It("should continue accepting user input during streaming", func() {
    // Start streaming
    go controller.simulateStreaming()
    
    // Inject keys during streaming
    screen.InjectKey(tcell.KeyUp, 0, tcell.ModNone)
    
    // Verify UI remains responsive
    Eventually(func() bool {
        // Check UI state
        return true
    }).Should(BeTrue())
})
```

### Testing Error Handling
```go
It("should gracefully handle streaming errors", func() {
    // Simulate error
    controller.errorChannel <- fmt.Errorf("network error")
    
    // Verify error display
    Eventually(func() string {
        return captureScreenContent(screen)
    }).Should(ContainSubstring("error"))
})
```

### Testing Terminal Resize
```go
It("should handle resize during streaming", func() {
    // Start streaming
    go controller.simulateStreaming()
    
    // Resize terminal
    screen.SetSize(120, 40)
    screen.PostEvent(tcell.NewEventResize(120, 40))
    
    // Verify content reflows correctly
})
```

## Debugging Tips

1. **Capture Screen History**: Use TestScreen's history feature to debug test failures
2. **Event Recording**: Track all events to understand interaction flow
3. **Verbose Output**: Use Ginkgo's -v flag for detailed test output
4. **Focused Tests**: Use FIt() to run specific tests during development

## Future Enhancements

1. **Visual Regression Testing**: Capture and compare screen snapshots
2. **Performance Benchmarks**: Add benchmarks for streaming performance
3. **Accessibility Testing**: Verify screen reader compatibility
4. **Cross-Platform Testing**: Test terminal-specific behaviors

## Example Test Suite

See the following files for complete examples:
- `pkg/tui/chat_stream_test.go` - Basic streaming tests
- `pkg/tui/streaming_integration_test.go` - Advanced integration tests
- `pkg/tui/chat_stream_example_test.go` - Example usage patterns
- `pkg/tui/test_helpers.go` - Reusable test utilities