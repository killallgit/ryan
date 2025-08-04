# TUI Development Guide

This guide covers terminal user interface development patterns, tcell best practices, and the specific architectural decisions that make Ryan's TUI responsive and maintainable.

## tcell Library Overview

tcell is Ryan's chosen TUI library because it provides:
- **Event-Driven Architecture**: Clean separation of input and output
- **Non-Blocking Operations**: Essential for streaming interfaces
- **Cross-Platform Support**: Consistent behavior across terminals
- **Low-Level Control**: Direct terminal manipulation when needed
- **Thread Safety**: Safe UI updates from background goroutines

## Core Architecture Patterns

### 1. Component-Based Design

Ryan's TUI uses stateless, composable components:

```go
type Component interface {
    Render(screen tcell.Screen, rect Rect)
    MinSize() (width, height int)
}

// Example: Message display component
type MessageList struct {
    Messages []Message
    Scroll   int
    Styles   MessageStyles
}

func (ml *MessageList) Render(screen tcell.Screen, rect Rect) {
    // Clear area and render messages
    ml.clearArea(screen, rect)
    
    visibleStart := ml.Scroll
    visibleEnd := min(len(ml.Messages), visibleStart+rect.Height)
    
    row := rect.Y
    for i := visibleStart; i < visibleEnd; i++ {
        msg := ml.Messages[i]
        style := ml.getStyleForRole(msg.Role)
        
        prefix := ml.getRolePrefix(msg.Role)
        col := renderText(screen, rect.X, row, rect.Width, prefix, style)
        renderText(screen, rect.X+col, row, rect.Width-col, msg.Content, style)
        
        row++
    }
}
```

### 2. Layout Management

Ryan uses a responsive 4-area layout:

```
┌─────────────────────────────────┐
│         Message Area            │ ← Chat history with scrolling
│    User: Hello there            │
│    Assistant: Hi! How can I..   │
├─────────────────────────────────┤
│         Alert Area              │ ← Spinner, errors, progress  
│    🔄 Sending... (5s)          │
├─────────────────────────────────┤
│         Input Area              │ ← Type your message here
│    > |                         │
├─────────────────────────────────┤
│        Status Area              │ ← Model info, connection status
│    Model: llama3.1:8b | Ready  │
└─────────────────────────────────┘
```

```go
type Layout struct {
    messageArea  Rect
    alertArea    Rect
    inputArea    Rect
    statusArea   Rect
}

func calculateLayout(w, h int) Layout {
    // Reserve bottom 3 lines for alert, input and status
    statusHeight := 1
    inputHeight := 1  
    alertHeight := 1
    messageHeight := h - statusHeight - inputHeight - alertHeight
    
    return Layout{
        messageArea: Rect{X: 0, Y: 0, Width: w, Height: messageHeight},
        alertArea:   Rect{X: 0, Y: messageHeight, Width: w, Height: alertHeight},
        inputArea:   Rect{X: 0, Y: messageHeight + alertHeight, Width: w, Height: inputHeight},
        statusArea:  Rect{X: 0, Y: h - statusHeight, Width: w, Height: statusHeight},
    }
}
```

### 3. Event-Driven Architecture

Ryan's event system supports custom events for async operations:

```go
type EventHandler interface {
    HandleKey(*tcell.EventKey) bool
    HandleMouse(*tcell.EventMouse) bool
    HandleResize(*tcell.EventResize) bool
    HandleInterrupt(*tcell.EventInterrupt) bool
}

// Custom event types
type MessageResponseEvent struct {
    tcell.EventTime
    Message chat.Message
}

type StreamChunkEvent struct {
    tcell.EventTime
    StreamID string
    Content  string
}

// Main event loop
func eventLoop(screen tcell.Screen, handler EventHandler) error {
    for {
        screen.Show()
        event := screen.PollEvent()
        
        switch ev := event.(type) {
        case *tcell.EventKey:
            if ev.Key() == tcell.KeyEscape {
                return nil
            }
            handler.HandleKey(ev)
            
        case *tcell.EventResize:
            screen.Sync()
            handler.HandleResize(ev)
            
        case *tcell.EventInterrupt:
            handler.HandleInterrupt(ev)
        }
    }
}
```

## Thread Safety for Streaming

### Critical Rule: UI Thread Only

**NEVER update UI directly from goroutines.** Always use tcell's PostEvent mechanism:

```go
// WRONG: Direct UI access from goroutine
go func() {
    for chunk := range streamChannel {
        messageDisplay.AddChunk(chunk.Content)  // RACE CONDITION!
    }
}()

// CORRECT: Queue updates for UI thread
go func() {
    for chunk := range streamChannel {
        screen.PostEvent(tcell.NewEventInterrupt(&StreamChunkEvent{
            StreamID: chunk.StreamID,
            Content:  chunk.Content,
        }))
    }
}()

// Handle in main event loop
func (app *App) handleInterrupt(ev *tcell.EventInterrupt) bool {
    if chunkEvent, ok := ev.Data().(*StreamChunkEvent); ok {
        app.accumulateChunk(chunkEvent.StreamID, chunkEvent.Content)
        return true
    }
    return false
}
```

### Non-Blocking Message Flow

This pattern keeps the UI responsive during API calls:

```go
func (app *App) sendMessage() {
    // 1. Update UI immediately
    app.input = app.input.Clear()
    app.sending = true
    app.showSpinner("Sending...")
    
    // 2. API call in background
    go func() {
        response, err := app.controller.SendUserMessage(content)
        
        // 3. Post result back to main thread
        if err != nil {
            app.screen.PostEvent(NewMessageErrorEvent(err))
        } else {
            app.screen.PostEvent(NewMessageResponseEvent(response))
        }
    }()
    // 4. Returns immediately - UI stays responsive
}
```

### State Management

Prevent concurrent operations with simple state tracking:

```go
type App struct {
    sending    bool       // Prevents multiple concurrent requests
    streaming  bool       // Tracks streaming state
    inputState InputState // Manages input buffer and cursor
}

// In key handler
case tcell.KeyEnter:
    if !app.sending && !app.streaming {
        app.sendMessage()
    }
```

## Safe Rendering Techniques

### Text Rendering with Bounds Checking

```go
func renderText(screen tcell.Screen, x, y, maxWidth int, text string, style tcell.Style) int {
    col := x
    for _, r := range []rune(text) {
        if col >= x+maxWidth {
            break // Don't exceed bounds
        }
        screen.SetContent(col, y, r, nil, style)
        col++
    }
    return col - x // Return characters rendered
}

func renderTextBlock(screen tcell.Screen, rect Rect, text string, style tcell.Style) {
    words := strings.Fields(text)
    row := rect.Y
    col := rect.X
    
    for _, word := range words {
        wordLen := len([]rune(word))
        
        // Word wrapping
        if col+wordLen > rect.X+rect.Width {
            row++
            col = rect.X
            if row >= rect.Y+rect.Height {
                break
            }
        }
        
        // Render word
        for _, r := range []rune(word) {
            screen.SetContent(col, row, r, nil, style)
            col++
        }
        
        // Add space
        if col < rect.X+rect.Width {
            screen.SetContent(col, row, ' ', nil, style)
            col++
        }
    }
}
```

### Area Clearing

Always clear areas before rendering new content:

```go
func clearArea(screen tcell.Screen, rect Rect) {
    clearStyle := tcell.StyleDefault.Background(tcell.ColorReset)
    for y := rect.Y; y < rect.Y+rect.Height; y++ {
        for x := rect.X; x < rect.X+rect.Width; x++ {
            screen.SetContent(x, y, ' ', nil, clearStyle)
        }
    }
}
```

## Enhanced UI Components

### Alert Display with Spinner

Ryan's alert area provides immediate feedback:

```go
type AlertDisplay struct {
    IsSpinnerVisible bool
    SpinnerFrame     int
    SpinnerText      string
    ErrorMessage     string
    ElapsedTime      time.Duration
}

func (ad *AlertDisplay) Render(screen tcell.Screen, rect Rect) {
    clearArea(screen, rect)
    
    var content string
    var style tcell.Style
    
    if ad.ErrorMessage != "" {
        // Base16 red for errors
        style = tcell.StyleDefault.Foreground(tcell.NewRGBColor(220, 50, 47))
        content = ad.ErrorMessage
    } else if ad.IsSpinnerVisible {
        style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
        
        // Animated spinner frames
        frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
        spinner := frames[ad.SpinnerFrame%len(frames)]
        
        // Include elapsed time
        timeStr := ""
        if ad.ElapsedTime > 0 {
            timeStr = fmt.Sprintf(" (%s)", ad.ElapsedTime.Round(time.Second))
        }
        
        content = spinner + " " + ad.SpinnerText + timeStr
    }
    
    if content != "" {
        renderText(screen, rect.X, rect.Y, rect.Width, content, style)
    }
}

func (ad *AlertDisplay) UpdateSpinner() {
    if ad.IsSpinnerVisible {
        ad.SpinnerFrame++
    }
}
```

### Input Field with Cursor

```go
type InputField struct {
    buffer   []rune
    cursor   int
    maxWidth int
    prompt   string
}

func (input *InputField) Render(screen tcell.Screen, rect Rect) {
    clearArea(screen, rect)
    
    // Render prompt
    col := renderText(screen, rect.X, rect.Y, rect.Width, input.prompt, 
        tcell.StyleDefault.Foreground(tcell.ColorBlue))
    
    // Render input content
    availableWidth := rect.Width - col
    content := string(input.buffer)
    renderText(screen, rect.X+col, rect.Y, availableWidth, content, 
        tcell.StyleDefault)
    
    // Position cursor
    cursorX := rect.X + col + input.cursor
    if cursorX < rect.X+rect.Width {
        screen.ShowCursor(cursorX, rect.Y)
    }
}

func (input *InputField) HandleKey(ev *tcell.EventKey) bool {
    switch ev.Key() {
    case tcell.KeyBackspace, tcell.KeyBackspace2:
        if input.cursor > 0 {
            input.cursor--
            input.buffer = append(input.buffer[:input.cursor], 
                input.buffer[input.cursor+1:]...)
        }
        return true
        
    case tcell.KeyLeft:
        if input.cursor > 0 {
            input.cursor--
        }
        return true
        
    case tcell.KeyRight:
        if input.cursor < len(input.buffer) {
            input.cursor++
        }
        return true
        
    case tcell.KeyRune:
        char := ev.Rune()
        input.buffer = append(input.buffer[:input.cursor],
            append([]rune{char}, input.buffer[input.cursor:]...)...)
        input.cursor++
        return true
    }
    return false
}
```

## Initialization and Cleanup

### Proper Screen Management

```go
func initializeScreen() (tcell.Screen, error) {
    screen, err := tcell.NewScreen()
    if err != nil {
        return nil, err
    }
    
    if err := screen.Init(); err != nil {
        return nil, err
    }
    
    // Configure screen
    defStyle := tcell.StyleDefault.
        Background(tcell.ColorReset).
        Foreground(tcell.ColorReset)
    screen.SetStyle(defStyle)
    
    screen.EnableMouse()
    screen.EnablePaste()
    screen.Clear()
    
    return screen, nil
}

func runApp() error {
    screen, err := initializeScreen()
    if err != nil {
        return err
    }
    
    // Ensure cleanup happens even on panic
    defer func() {
        if r := recover(); r != nil {
            screen.Fini()
            panic(r)
        }
    }()
    defer screen.Fini()
    
    return runEventLoop(screen)
}
```

## Advanced Features

### Mouse Support

```go
func (app *App) handleMouse(ev *tcell.EventMouse) bool {
    x, y := ev.Position()
    buttons := ev.Buttons()
    
    // Scroll in message area
    if app.layout.messageArea.Contains(x, y) {
        if buttons&tcell.WheelUp != 0 {
            app.scrollUp(3)
            return true
        }
        if buttons&tcell.WheelDown != 0 {
            app.scrollDown(3)
            return true
        }
    }
    
    return false
}
```

### Resize Handling

```go
func (app *App) handleResize(ev *tcell.EventResize) bool {
    app.layout = calculateLayout(ev.Size())
    
    // Adjust scroll position if needed
    maxScroll := len(app.messages) - app.layout.messageArea.Height
    if app.scroll > maxScroll {
        app.scroll = max(0, maxScroll)
    }
    
    return true
}
```

## Testing TUI Components

### Component Testing with Simulation Screen

```go
func TestMessageListRender(t *testing.T) {
    screen := tcell.NewSimulationScreen("UTF-8")
    screen.Init()
    screen.SetSize(80, 24)
    
    messages := []Message{
        {Role: "user", Content: "Hello"},
        {Role: "assistant", Content: "Hi there!"},
    }
    
    messageList := &MessageList{Messages: messages}
    rect := Rect{X: 0, Y: 0, Width: 80, Height: 20}
    
    messageList.Render(screen, rect)
    
    // Verify content
    contents, _, _ := screen.GetContents()
    screenText := contentsToString(contents)
    
    assert.Contains(t, screenText, "Hello")
    assert.Contains(t, screenText, "Hi there!")
}
```

### Event Simulation

```go
func TestKeyboardInput(t *testing.T) {
    screen := tcell.NewSimulationScreen("UTF-8")
    screen.Init()
    
    app := NewApp(screen)
    
    // Simulate typing
    screen.InjectKey(tcell.KeyRune, 'H', tcell.ModNone)
    screen.InjectKey(tcell.KeyRune, 'i', tcell.ModNone)
    screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
    
    // Process events
    for i := 0; i < 3; i++ {
        event := screen.PollEvent()
        app.handleEvent(event)
    }
    
    assert.Equal(t, "Hi", app.getLastInput())
}
```

## Performance Optimization

### Minimal Redraws

Only update areas that have changed:

```go
type DirtyRegions struct {
    messageArea bool
    alertArea   bool
    inputArea   bool
    statusArea  bool
}

func (app *App) render() {
    if app.dirty.messageArea {
        app.messageList.Render(app.screen, app.layout.messageArea)
        app.dirty.messageArea = false
    }
    
    if app.dirty.alertArea {
        app.alertDisplay.Render(app.screen, app.layout.alertArea)
        app.dirty.alertArea = false
    }
    
    // Only call Show() once after all updates
    app.screen.Show()
}
```

### Efficient String Building

Use strings.Builder for accumulating text:

```go
type StreamingMessage struct {
    content   strings.Builder  // Efficient accumulation
    lines     []int           // Track line positions for scrolling
    complete  bool
}

func (sm *StreamingMessage) AddChunk(chunk string) {
    start := sm.content.Len()
    sm.content.WriteString(chunk)
    
    // Track line positions
    for i, r := range chunk {
        if r == '\n' {
            sm.lines = append(sm.lines, start+i+1)
        }
    }
}
```

## Best Practices Summary

### Do's
- ✅ Use PostEvent() for all UI updates from goroutines
- ✅ Clear areas before rendering new content
- ✅ Handle resize events to maintain layout
- ✅ Use bounds checking in all text rendering
- ✅ Implement proper cleanup with defer statements
- ✅ Test components with simulation screens
- ✅ Use component-based architecture for maintainability

### Don'ts
- ❌ Never update UI directly from background goroutines
- ❌ Don't forget to call screen.Show() after updates
- ❌ Don't render outside component bounds
- ❌ Don't block the main event loop with long operations
- ❌ Don't forget screen.Fini() cleanup
- ❌ Don't use global state for UI components

### Threading Rules
- **Main Thread**: Handle all tcell events and UI rendering
- **Background Threads**: API calls, file operations, streaming
- **Communication**: Only via PostEvent() and channels
- **State**: Manage UI state only in main thread

This architecture provides the foundation for Ryan's responsive, streaming-capable TUI that maintains excellent performance even during intensive operations.