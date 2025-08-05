# TUI Patterns & tview Best Practices

## tview Library Overview

tview is a rich terminal user interface library for Go that provides high-level widgets and components. Built on top of tcell, tview abstracts away the low-level terminal control, allowing us to focus on application logic and user experience.

### Key tview Advantages for Chat TUI
- **Rich Widget Library**: Pre-built components like TextView, InputField, Table, TreeView
- **Built-in Layout Management**: Flexible box layouts with automatic resizing
- **Thread-Safe Updates**: QueueUpdateDraw for safe concurrent UI updates
- **Event Handling**: Simplified keyboard and mouse event handling
- **Focus Management**: Automatic focus traversal between components

## Core tview Patterns

### 1. Application Structure
```go
// Main application structure
type App struct {
    app        *tview.Application
    pages      *tview.Pages       // For view switching
    controller ControllerInterface
    
    // Views
    chatView        *ChatView
    modelView       *ModelView
    toolsView       *ToolsView
    vectorStoreView *VectorStoreView
    contextTreeView *ContextTreeView
}

// Initialize tview application
func NewApp(controller ControllerInterface) (*App, error) {
    tviewApp := tview.NewApplication()
    
    app := &App{
        app:         tviewApp,
        pages:       tview.NewPages(),
        controller:  controller,
        currentView: "chat",
    }
    
    // Initialize views
    if err := app.initializeViews(); err != nil {
        return nil, fmt.Errorf("failed to initialize views: %w", err)
    }
    
    // Set root and run
    tviewApp.SetRoot(app.pages, true).SetFocus(app.pages)
    return app, nil
}
```

### 2. View Management with Pages
```go
// Using tview.Pages for view switching
func (a *App) initializeViews() error {
    // Create and add views to pages
    a.chatView = NewChatView(a.controller, a.app)
    a.pages.AddPage("chat", a.chatView, true, true)
    
    a.modelView = NewModelView(modelsController, a.controller, a.app)
    a.pages.AddPage("models", a.modelView, true, false)
    
    // Add more views...
    return nil
}

// Switch between views
func (a *App) switchToView(viewName string) {
    a.currentView = viewName
    a.pages.SwitchToPage(viewName)
}
```

### 3. Component-Based Views
```go
// Chat view using tview widgets
type ChatView struct {
    *tview.Flex                     // Embed Flex for layout
    messages     *tview.TextView     // For displaying messages
    input        *tview.InputField   // For user input
    status       *tview.TextView     // For status line
    spinner      *Spinner            // Custom spinner component
    
    controller   ControllerInterface
    app          *tview.Application
}

func NewChatView(controller ControllerInterface, app *tview.Application) *ChatView {
    cv := &ChatView{
        Flex:       tview.NewFlex().SetDirection(tview.FlexRow),
        messages:   tview.NewTextView(),
        input:      tview.NewInputField(),
        status:     tview.NewTextView(),
        spinner:    NewSpinner(app),
        controller: controller,
        app:        app,
    }
    
    // Configure widgets
    cv.messages.
        SetDynamicColors(true).
        SetRegions(true).
        SetWordWrap(true).
        SetScrollable(true)
    
    cv.input.
        SetLabel("> ").
        SetFieldBackgroundColor(tcell.ColorDefault)
    
    // Build layout
    cv.AddItem(cv.messages, 0, 1, false).
        AddItem(cv.spinner, 1, 0, false).
        AddItem(cv.input, 1, 0, true).
        AddItem(cv.status, 1, 0, false)
    
    return cv
}
```

## Thread Safety Patterns for Streaming

### 1. Safe UI Updates from Goroutines
```go
// CRITICAL PATTERN: Always use QueueUpdateDraw for concurrent updates
func (a *App) processStreamingUpdates(updates <-chan controllers.StreamingUpdate) {
    streamingContent := ""
    
    for update := range updates {
        switch update.Type {
        case controllers.ChunkReceived:
            streamingContent += update.Content
            content := streamingContent // Capture for closure
            
            // Thread-safe UI update
            a.app.QueueUpdateDraw(func() {
                a.chatView.UpdateStreamingContent(update.StreamID, content)
            })
            
        case controllers.MessageComplete:
            finalMsg := update.Message
            
            // Thread-safe UI update
            a.app.QueueUpdateDraw(func() {
                a.streaming = false
                a.chatView.CompleteStreaming(update.StreamID, finalMsg)
                a.chatView.UpdateMessages()
            })
        }
    }
}
```

### 2. Input Handling with Event Callbacks
```go
// Set up input handling
func (cv *ChatView) setupInputHandlers() {
    cv.input.SetDoneFunc(func(key tcell.Key) {
        if key == tcell.KeyEnter {
            text := cv.input.GetText()
            if text != "" {
                cv.input.SetText("")
                if cv.sendHandler != nil {
                    cv.sendHandler(text)
                }
            }
        }
    })
    
    // Additional key handling
    cv.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        // Handle special keys
        switch event.Key() {
        case tcell.KeyCtrlC:
            // Cancel current operation
            return nil
        case tcell.KeyTab:
            // Trigger autocomplete
            return nil
        }
        return event
    })
}
```

### 3. Custom Components
```go
// Custom spinner component
type Spinner struct {
    *tview.TextView
    app        *tview.Application
    isRunning  bool
    frame      int
    text       string
    stopChan   chan bool
}

func NewSpinner(app *tview.Application) *Spinner {
    s := &Spinner{
        TextView: tview.NewTextView(),
        app:      app,
        stopChan: make(chan bool, 1),
    }
    s.SetTextAlign(tview.AlignLeft)
    return s
}

func (s *Spinner) Start(text string) {
    if s.isRunning {
        return
    }
    
    s.isRunning = true
    s.text = text
    s.frame = 0
    
    go func() {
        frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                s.frame = (s.frame + 1) % len(frames)
                // Thread-safe update
                s.app.QueueUpdateDraw(func() {
                    s.SetText(fmt.Sprintf("%s %s", frames[s.frame], s.text))
                })
            case <-s.stopChan:
                return
            }
        }
    }()
}
```

## Advanced tview Features

### 1. Modal Dialogs
```go
// Create a modal for view switching
func (a *App) showViewSwitcher() {
    list := tview.NewList().
        AddItem("Chat", "Main chat interface", '1', func() {
            a.switchToView("chat")
            a.pages.RemovePage("view-switcher")
        }).
        AddItem("Models", "Manage Ollama models", '2', func() {
            a.switchToView("models")
            a.pages.RemovePage("view-switcher")
        })
    
    list.SetBorder(true).SetTitle("Switch View (Ctrl-P)")
    
    // Create centered modal
    modal := createModal(list, 40, 15)
    a.pages.AddPage("view-switcher", modal, true, true)
}

func createModal(p tview.Primitive, width, height int) tview.Primitive {
    return tview.NewFlex().
        AddItem(nil, 0, 1, false).
        AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
            AddItem(nil, 0, 1, false).
            AddItem(p, height, 1, true).
            AddItem(nil, 0, 1, false), width, 1, true).
        AddItem(nil, 0, 1, false)
}
```

### 2. Table for Data Display
```go
// Tools view with table
type ToolsView struct {
    *tview.Flex
    table    *tview.Table
    details  *tview.TextView
    registry *tools.Registry
}

func (tv *ToolsView) populateTable() {
    tv.table.Clear()
    
    // Headers
    headers := []string{"Tool", "Description", "Category", "Status"}
    for col, header := range headers {
        tv.table.SetCell(0, col,
            tview.NewTableCell(header).
                SetTextColor(tcell.ColorYellow).
                SetAttributes(tcell.AttrBold))
    }
    
    // Data rows
    for row, tool := range tv.registry.ListTools() {
        tv.table.SetCell(row+1, 0, tview.NewTableCell(tool.Name))
        tv.table.SetCell(row+1, 1, tview.NewTableCell(tool.Description))
        tv.table.SetCell(row+1, 2, tview.NewTableCell(tool.Category))
        tv.table.SetCell(row+1, 3, tview.NewTableCell(tool.Status))
    }
    
    // Selection handler
    tv.table.SetSelectionChangedFunc(func(row, col int) {
        if row > 0 {
            tool := tv.registry.GetTool(row - 1)
            tv.details.SetText(tool.GetDetails())
        }
    })
}
```

### 3. TreeView for Hierarchical Data
```go
// Context tree view
type ContextTreeView struct {
    *tview.Flex
    tree *tview.TreeView
    root *tview.TreeNode
}

func (ctv *ContextTreeView) buildTree(conversations []Conversation) {
    ctv.root = tview.NewTreeNode("Conversations").
        SetColor(tcell.ColorYellow)
    
    for _, conv := range conversations {
        convNode := tview.NewTreeNode(conv.Title).
            SetReference(conv.ID)
        
        // Add messages as child nodes
        for _, msg := range conv.Messages {
            msgNode := tview.NewTreeNode(msg.Preview).
                SetReference(msg.ID)
            convNode.AddChild(msgNode)
        }
        
        ctv.root.AddChild(convNode)
    }
    
    ctv.tree.SetRoot(ctv.root).
        SetCurrentNode(ctv.root)
}
```

### 4. Dynamic Color and Formatting
```go
// Format messages with colors
func (cv *ChatView) formatMessage(msg chat.Message) string {
    timestamp := msg.Timestamp.Format("15:04:05")
    
    switch msg.Role {
    case "human":
        return fmt.Sprintf("[blue]%s [You]:[white] %s", timestamp, msg.Content)
    case "assistant":
        return fmt.Sprintf("[green]%s [Assistant]:[white] %s", timestamp, msg.Content)
    case "system":
        return fmt.Sprintf("[yellow]%s [System]:[white] %s", timestamp, msg.Content)
    case "error":
        return fmt.Sprintf("[red]%s [Error]:[white] %s", timestamp, msg.Content)
    default:
        return fmt.Sprintf("%s [%s]: %s", timestamp, msg.Role, msg.Content)
    }
}
```

## Event-Driven Patterns

### 1. Global Key Bindings
```go
func (a *App) setupGlobalKeyBindings() {
    a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        // Ctrl-P: Toggle command palette
        if event.Key() == tcell.KeyCtrlP {
            a.showViewSwitcher()
            return nil
        }
        
        // Ctrl-C: Cancel or quit
        if event.Key() == tcell.KeyCtrlC {
            if a.sending {
                select {
                case a.cancelSend <- true:
                default:
                }
            } else {
                a.app.Stop()
            }
            return nil
        }
        
        // Ctrl-1 through Ctrl-5: Quick view switching
        if event.Key() >= tcell.KeyCtrlA && event.Key() <= tcell.KeyCtrlE {
            views := []string{"chat", "models", "tools", "vectorstore", "context-tree"}
            index := int(event.Key() - tcell.KeyCtrlA)
            if index < len(views) {
                a.switchToView(views[index])
                return nil
            }
        }
        
        return event
    })
}
```

### 2. Context-Aware Input Handling
```go
// Different input modes
type InputMode int
const (
    InputModeNormal InputMode = iota
    InputModeVim
    InputModeSearch
)

func (cv *ChatView) handleInputMode(event *tcell.EventKey) *tcell.EventKey {
    switch cv.inputMode {
    case InputModeVim:
        return cv.handleVimKeys(event)
    case InputModeSearch:
        return cv.handleSearchKeys(event)
    default:
        return event
    }
}
```

## Performance Optimizations

### 1. Efficient Message Rendering
```go
// Only update visible messages
func (cv *ChatView) UpdateMessages() {
    cv.messages.Clear()
    
    history := cv.controller.GetHistory()
    visibleMessages := cv.getVisibleMessages(history)
    
    var content strings.Builder
    for _, msg := range visibleMessages {
        content.WriteString(cv.formatMessage(msg))
        content.WriteString("\n\n")
    }
    
    cv.messages.SetText(content.String())
    cv.messages.ScrollToEnd()
}

func (cv *ChatView) getVisibleMessages(history []chat.Message) []chat.Message {
    // Implement windowing for large histories
    _, _, _, height := cv.messages.GetRect()
    estimatedLinesPerMessage := 3
    maxMessages := height / estimatedLinesPerMessage
    
    if len(history) > maxMessages {
        return history[len(history)-maxMessages:]
    }
    return history
}
```

### 2. Debounced Updates
```go
// Debounce rapid updates during streaming
type Debouncer struct {
    timer    *time.Timer
    duration time.Duration
    fn       func()
    mu       sync.Mutex
}

func NewDebouncer(duration time.Duration, fn func()) *Debouncer {
    return &Debouncer{
        duration: duration,
        fn:       fn,
    }
}

func (d *Debouncer) Call() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.timer != nil {
        d.timer.Stop()
    }
    
    d.timer = time.AfterFunc(d.duration, d.fn)
}
```

## Testing tview Applications

### 1. Component Testing
```go
func TestChatView(t *testing.T) {
    // Create test application
    app := tview.NewApplication()
    controller := &MockController{}
    
    chatView := NewChatView(controller, app)
    
    // Test message display
    controller.messages = []chat.Message{
        {Role: "human", Content: "Hello"},
        {Role: "assistant", Content: "Hi there!"},
    }
    
    chatView.UpdateMessages()
    
    // Verify content
    content := chatView.messages.GetText(false)
    assert.Contains(t, content, "Hello")
    assert.Contains(t, content, "Hi there!")
}
```

### 2. Event Simulation
```go
func TestKeyboardShortcuts(t *testing.T) {
    app := &App{
        app:         tview.NewApplication(),
        pages:       tview.NewPages(),
        currentView: "chat",
    }
    
    // Simulate Ctrl-P
    event := tcell.NewEventKey(tcell.KeyCtrlP, 0, tcell.ModCtrl)
    handled := app.handleGlobalKey(event)
    
    assert.True(t, handled)
    assert.True(t, app.pages.HasPage("view-switcher"))
}
```

### 3. Streaming Simulation
```go
func TestStreamingUpdates(t *testing.T) {
    app := &App{
        app:      tview.NewApplication(),
        chatView: NewChatView(nil, nil),
    }
    
    updates := make(chan controllers.StreamingUpdate)
    
    go app.processStreamingUpdates(updates)
    
    // Send test updates
    updates <- controllers.StreamingUpdate{
        Type:     controllers.StreamStarted,
        StreamID: "test-123",
    }
    
    updates <- controllers.StreamingUpdate{
        Type:     controllers.ChunkReceived,
        StreamID: "test-123",
        Content:  "Hello ",
    }
    
    updates <- controllers.StreamingUpdate{
        Type:     controllers.ChunkReceived,
        StreamID: "test-123",
        Content:  "world!",
    }
    
    close(updates)
    
    // Verify final content
    time.Sleep(100 * time.Millisecond) // Allow updates to process
    assert.Equal(t, "Hello world!", app.chatView.streamingContent)
}
```

## Migration from tcell to tview

### Key Differences
1. **Event Handling**: tview provides higher-level event callbacks instead of raw event loops
2. **Layout Management**: Use Flex containers instead of manual coordinate calculations
3. **Component Lifecycle**: tview manages component rendering and updates automatically
4. **Thread Safety**: Always use QueueUpdateDraw instead of direct updates
5. **Focus Management**: tview handles focus traversal between components

### Benefits of tview
- **Reduced Code**: ~80% less code compared to raw tcell implementation
- **Built-in Components**: Rich set of widgets out of the box
- **Automatic Layout**: Responsive design without manual calculations
- **Better Abstractions**: Focus on business logic instead of terminal mechanics
- **Easier Testing**: Higher-level APIs are easier to test

---

*This document provides the foundational patterns for implementing a responsive, streaming-capable TUI using tview*