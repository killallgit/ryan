# TUI Patterns & tcell Best Practices

## tcell Library Overview

tcell is a low-level terminal control library that provides direct access to terminal capabilities. Unlike higher-level TUI frameworks, tcell gives us complete control over the terminal interface, making it ideal for responsive streaming applications.

### Key tcell Advantages for Chat TUI
- **Event-Driven Architecture**: Clean separation of input and output
- **Non-Blocking Operations**: Essential for streaming interfaces
- **Cross-Platform**: Works consistently across terminals
- **Mouse Support**: Enhanced user interaction
- **Resize Handling**: Graceful terminal size changes

## Core tcell Patterns

### 1. Screen Initialization & Cleanup
```go
func initializeScreen() (tcell.Screen, error) {
    screen, err := tcell.NewScreen()
    if err != nil {
        return nil, err
    }
    
    if err := screen.Init(); err != nil {
        return nil, err
    }
    
    // Set default style
    defStyle := tcell.StyleDefault.
        Background(tcell.ColorReset).
        Foreground(tcell.ColorReset)
    screen.SetStyle(defStyle)
    
    // Enable features we need
    screen.EnableMouse()
    screen.EnablePaste()
    screen.Clear()
    
    return screen, nil
}

func cleanupScreen(screen tcell.Screen) {
    // Critical: Always clean up properly
    screen.Fini()
}

// Proper error handling with cleanup
func runApp() error {
    screen, err := initializeScreen()
    if err != nil {
        return err
    }
    
    // Ensure cleanup happens even on panic
    defer func() {
        if r := recover(); r != nil {
            screen.Fini()
            panic(r) // Re-raise panic after cleanup
        }
    }()
    defer screen.Fini()
    
    return eventLoop(screen)
}
```

### 2. Event Loop Structure
```go
type EventHandler interface {
    HandleKey(*tcell.EventKey) bool      // Returns true if handled
    HandleMouse(*tcell.EventMouse) bool
    HandleResize(*tcell.EventResize) bool
    HandleInterrupt(*tcell.EventInterrupt) bool
}

func eventLoop(screen tcell.Screen) error {
    handler := NewChatEventHandler()
    
    for {
        // Show updates to screen
        screen.Show()
        
        // Poll for next event (blocks until event arrives)
        event := screen.PollEvent()
        
        var handled bool
        switch ev := event.(type) {
        case *tcell.EventKey:
            handled = handler.HandleKey(ev)
            if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
                return nil // Exit application
            }
            
        case *tcell.EventMouse:
            handled = handler.HandleMouse(ev)
            
        case *tcell.EventResize:
            screen.Sync() // Important: sync after resize
            handled = handler.HandleResize(ev)
            
        case *tcell.EventInterrupt:
            // Custom events from other goroutines
            handled = handler.HandleInterrupt(ev)
        }
        
        if !handled {
            // Log unhandled events for debugging
            logUnhandledEvent(event)
        }
    }
}
```

### 3. Safe Content Rendering
```go
// Helper function to safely render text within bounds
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

// Multi-line text rendering with word wrapping
func renderTextBlock(screen tcell.Screen, rect Rect, text string, style tcell.Style) {
    words := strings.Fields(text)
    row := rect.Y
    col := rect.X
    
    for _, word := range words {
        wordLen := len([]rune(word))
        
        // Check if word fits on current line
        if col+wordLen > rect.X+rect.Width {
            row++
            col = rect.X
            
            // Check if we have room for another line
            if row >= rect.Y+rect.Height {
                break
            }
        }
        
        // Render the word
        for _, r := range []rune(word) {
            screen.SetContent(col, row, r, nil, style)
            col++
        }
        
        // Add space after word (if room)
        if col < rect.X+rect.Width {
            screen.SetContent(col, row, ' ', nil, style)
            col++
        }
    }
}
```

### 4. Component-Based Rendering
```go
type Rect struct {
    X, Y, Width, Height int
}

type Component interface {
    Render(screen tcell.Screen, rect Rect)
    MinSize() (width, height int)
}

// Message display component
type MessageList struct {
    Messages []Message
    Scroll   int
    Styles   MessageStyles
}

func (ml *MessageList) Render(screen tcell.Screen, rect Rect) {
    // Clear the area first
    ml.clearArea(screen, rect)
    
    // Calculate visible message range
    visibleStart := ml.Scroll
    visibleEnd := min(len(ml.Messages), visibleStart+rect.Height)
    
    row := rect.Y
    for i := visibleStart; i < visibleEnd; i++ {
        msg := ml.Messages[i]
        style := ml.getStyleForRole(msg.Role)
        
        // Render message with prefix
        prefix := ml.getRolePrefix(msg.Role)
        col := renderText(screen, rect.X, row, rect.Width, prefix, style)
        renderText(screen, rect.X+col, row, rect.Width-col, msg.Content, style)
        
        row++
        if row >= rect.Y+rect.Height {
            break
        }
    }
}

func (ml *MessageList) clearArea(screen tcell.Screen, rect Rect) {
    clearStyle := tcell.StyleDefault.Background(tcell.ColorReset)
    for y := rect.Y; y < rect.Y+rect.Height; y++ {
        for x := rect.X; x < rect.X+rect.Width; x++ {
            screen.SetContent(x, y, ' ', nil, clearStyle)
        }
    }
}
```

## Thread Safety Patterns for Streaming

### 1. Safe UI Updates from Goroutines
```go
// CRITICAL PATTERN: Never update UI directly from goroutines
type StreamingApp struct {
    screen       tcell.Screen
    messageList  *MessageList
    updateQueue  chan UIUpdate
}

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
)

// Goroutine-safe method to queue UI updates
func (app *StreamingApp) QueueMessageChunk(streamID string, content string) {
    // This is safe to call from any goroutine
    app.screen.PostEvent(tcell.NewEventInterrupt(&UIUpdate{
        Type: MessageChunkUpdate,
        Data: MessageChunkData{
            StreamID: streamID,
            Content:  content,
        },
    }))
}

// Handle updates in main event loop only
func (app *StreamingApp) handleInterrupt(ev *tcell.EventInterrupt) bool {
    if update, ok := ev.Data().(*UIUpdate); ok {
        switch update.Type {
        case MessageChunkUpdate:
            data := update.Data.(MessageChunkData)
            app.updateStreamingMessage(data.StreamID, data.Content)
            return true
            
        case MessageCompleteUpdate:
            data := update.Data.(MessageCompleteData)
            app.finalizeMessage(data.Message)
            return true
        }
    }
    return false
}
```

### 2. Input Handling During Streaming
```go
type InputState struct {
    buffer     []rune
    cursor     int
    isEnabled  bool  // Disable during streaming if needed
}

func (app *StreamingApp) handleKeyEvent(ev *tcell.EventKey) bool {
    if !app.inputState.isEnabled {
        return true // Consume but don't process
    }
    
    switch ev.Key() {
    case tcell.KeyEnter:
        message := string(app.inputState.buffer)
        app.inputState.buffer = app.inputState.buffer[:0]
        app.inputState.cursor = 0
        
        // Start streaming (this will disable input until complete)
        app.startStreaming(message)
        return true
        
    case tcell.KeyBackspace, tcell.KeyBackspace2:
        if app.inputState.cursor > 0 {
            app.inputState.cursor--
            app.inputState.buffer = append(
                app.inputState.buffer[:app.inputState.cursor],
                app.inputState.buffer[app.inputState.cursor+1:]...,
            )
        }
        return true
        
    case tcell.KeyRune:
        // Insert character at cursor position
        char := ev.Rune()
        app.inputState.buffer = append(
            app.inputState.buffer[:app.inputState.cursor],
            append([]rune{char}, app.inputState.buffer[app.inputState.cursor:]...)...,
        )
        app.inputState.cursor++
        return true
    }
    
    return false
}
```

### 3. Responsive Layout Management
```go
type Layout struct {
    screenWidth  int
    screenHeight int
    messageArea  Rect
    inputArea    Rect
    statusArea   Rect
}

func (app *StreamingApp) calculateLayout() Layout {
    w, h := app.screen.Size()
    
    // Reserve bottom 2 lines for input and status
    messageHeight := h - 2
    if messageHeight < 1 {
        messageHeight = 1
    }
    
    return Layout{
        screenWidth:  w,
        screenHeight: h,
        messageArea: Rect{
            X: 0, Y: 0,
            Width: w, Height: messageHeight,
        },
        inputArea: Rect{
            X: 0, Y: messageHeight,
            Width: w, Height: 1,
        },
        statusArea: Rect{
            X: 0, Y: h - 1,
            Width: w, Height: 1,
        },
    }
}

func (app *StreamingApp) handleResize(ev *tcell.EventResize) bool {
    app.layout = app.calculateLayout()
    
    // Adjust scroll position if needed
    maxScroll := len(app.messageList.Messages) - app.layout.messageArea.Height
    if app.messageList.Scroll > maxScroll {
        app.messageList.Scroll = max(0, maxScroll)
    }
    
    return true
}
```

## Streaming-Specific UI Patterns

### 1. Progressive Message Display
```go
type StreamingMessage struct {
    ID          string
    Role        string
    Content     strings.Builder // Efficient string accumulation
    IsComplete  bool
    StartTime   time.Time
    LastUpdate  time.Time
}

func (app *StreamingApp) updateStreamingMessage(streamID, chunk string) {
    // Find or create streaming message
    msg := app.findOrCreateStreamingMessage(streamID)
    
    // Append chunk to content
    msg.Content.WriteString(chunk)
    msg.LastUpdate = time.Now()
    
    // Auto-scroll to bottom if user is at bottom
    if app.isScrolledToBottom() {
        app.scrollToBottom()
    }
    
    // Mark screen for redraw
    app.screen.Show()
}

func (app *StreamingApp) isScrolledToBottom() bool {
    maxScroll := len(app.messageList.Messages) - app.layout.messageArea.Height
    return app.messageList.Scroll >= maxScroll
}
```

### 2. Typing Indicators
```go
type TypingIndicator struct {
    IsVisible bool
    StartTime time.Time
    Animation int // Animation frame
}

func (app *StreamingApp) renderTypingIndicator(screen tcell.Screen, rect Rect) {
    if !app.typingIndicator.IsVisible {
        return
    }
    
    // Animate typing dots
    frames := []string{"   ", ".  ", ".. ", "..."}
    frame := frames[app.typingIndicator.Animation%len(frames)]
    
    style := tcell.StyleDefault.Foreground(tcell.ColorGray)
    prefix := "Assistant is typing"
    
    renderText(screen, rect.X, rect.Y, rect.Width, prefix+frame, style)
}

func (app *StreamingApp) updateTypingAnimation() {
    if app.typingIndicator.IsVisible {
        app.typingIndicator.Animation++
        // Update every 500ms
        if time.Since(app.typingIndicator.StartTime) > 500*time.Millisecond {
            app.typingIndicator.StartTime = time.Now()
        }
    }
}
```

### 3. Status and Error Display
```go
type StatusBar struct {
    Message   string
    Type      StatusType
    Timestamp time.Time
}

type StatusType int
const (
    StatusInfo StatusType = iota
    StatusSuccess
    StatusWarning
    StatusError
)

func (sb *StatusBar) Render(screen tcell.Screen, rect Rect) {
    style := sb.getStatusStyle()
    
    // Clear status area
    for x := rect.X; x < rect.X+rect.Width; x++ {
        screen.SetContent(x, rect.Y, ' ', nil, style)
    }
    
    // Render status message
    renderText(screen, rect.X, rect.Y, rect.Width, sb.Message, style)
}

func (sb *StatusBar) getStatusStyle() tcell.Style {
    base := tcell.StyleDefault
    switch sb.Type {
    case StatusError:
        return base.Foreground(tcell.ColorWhite).Background(tcell.ColorRed)
    case StatusWarning:
        return base.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow)
    case StatusSuccess:
        return base.Foreground(tcell.ColorWhite).Background(tcell.ColorGreen)
    default:
        return base.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue)
    }
}
```

## Advanced tcell Features

### 1. Mouse Support
```go
func (app *StreamingApp) handleMouse(ev *tcell.EventMouse) bool {
    x, y := ev.Position()
    buttons := ev.Buttons()
    
    // Check which area was clicked
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

func (r Rect) Contains(x, y int) bool {
    return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}
```

### 2. Clipboard Support
```go
func (app *StreamingApp) handlePaste(ev *tcell.EventPaste) bool {
    if ev.Start() {
        app.pasteMode = true
        app.pasteBuffer.Reset()
        return true
    } else {
        app.pasteMode = false
        pastedText := app.pasteBuffer.String()
        app.insertTextAtCursor(pastedText)
        return true
    }
}
```

### 3. Color and Styling
```go
type Theme struct {
    UserMessage      tcell.Style
    AssistantMessage tcell.Style
    SystemMessage    tcell.Style
    InputField       tcell.Style
    StatusBar        tcell.Style
    ErrorMessage     tcell.Style
}

func DefaultTheme() Theme {
    return Theme{
        UserMessage: tcell.StyleDefault.
            Foreground(tcell.ColorBlue).
            Bold(true),
        AssistantMessage: tcell.StyleDefault.
            Foreground(tcell.ColorGreen),
        SystemMessage: tcell.StyleDefault.
            Foreground(tcell.ColorGray).
            Italic(true),
        InputField: tcell.StyleDefault.
            Background(tcell.ColorDarkGray),
        StatusBar: tcell.StyleDefault.
            Background(tcell.ColorBlue).
            Foreground(tcell.ColorWhite),
        ErrorMessage: tcell.StyleDefault.
            Foreground(tcell.ColorRed).
            Bold(true),
    }
}
```

## Testing TUI Components

### 1. Component Unit Testing
```go
func TestMessageListRender(t *testing.T) {
    // Create a simulation screen for testing
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
    
    // Verify content was rendered correctly
    contents, width, height := screen.GetContents()
    
    // Check that messages appear in output
    screenText := string(contents)
    assert.Contains(t, screenText, "Hello")
    assert.Contains(t, screenText, "Hi there!")
}
```

### 2. Event Simulation Testing
```go
func TestKeyboardInput(t *testing.T) {
    screen := tcell.NewSimulationScreen("UTF-8")
    screen.Init()
    
    app := NewStreamingApp(screen)
    
    // Simulate typing
    screen.InjectKey(tcell.KeyRune, 'H', tcell.ModNone)
    screen.InjectKey(tcell.KeyRune, 'i', tcell.ModNone)
    screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
    
    // Process events
    for i := 0; i < 3; i++ {
        event := screen.PollEvent()
        app.handleEvent(event)
    }
    
    // Verify input was processed
    assert.Equal(t, "Hi", app.getLastUserInput())
}
```

### 3. Layout Testing
```go
func TestResponsiveLayout(t *testing.T) {
    app := &StreamingApp{}
    
    // Test various screen sizes
    sizes := []struct{ w, h int }{
        {80, 24}, {120, 40}, {40, 10},
    }
    
    for _, size := range sizes {
        app.screen = createMockScreen(size.w, size.h)
        layout := app.calculateLayout()
        
        // Verify layout constraints
        assert.True(t, layout.messageArea.Height >= 1)
        assert.Equal(t, layout.inputArea.Height, 1)
        assert.Equal(t, layout.statusArea.Height, 1)
        assert.Equal(t, layout.messageArea.Height+2, size.h)
    }
}
```

## Performance Optimizations

### 1. Minimal Redraws
```go
type DirtyRegions struct {
    messageArea bool
    inputArea   bool
    statusArea  bool
}

func (app *StreamingApp) render() {
    if app.dirty.messageArea {
        app.messageList.Render(app.screen, app.layout.messageArea)
        app.dirty.messageArea = false
    }
    
    if app.dirty.inputArea {
        app.inputField.Render(app.screen, app.layout.inputArea)
        app.dirty.inputArea = false
    }
    
    if app.dirty.statusArea {
        app.statusBar.Render(app.screen, app.layout.statusArea)
        app.dirty.statusArea = false
    }
}
```

### 2. Efficient Text Handling
```go
// Use strings.Builder for efficient text accumulation
type EfficientMessageBuffer struct {
    builder strings.Builder
    lines   []int // Track line start positions for fast scrolling
}

func (emb *EfficientMessageBuffer) AppendChunk(chunk string) {
    emb.builder.WriteString(chunk)
    
    // Update line positions for new content
    start := emb.builder.Len() - len(chunk)
    for i, r := range chunk {
        if r == '\n' {
            emb.lines = append(emb.lines, start+i+1)
        }
    }
}
```

---

*This document provides the foundational patterns for implementing a responsive, streaming-capable TUI using tcell*