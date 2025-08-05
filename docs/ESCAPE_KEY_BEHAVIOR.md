# Escape Key Behavior

The escape key now works consistently across all views and modals in the TUI application.

## 🎯 Universal Escape Key Behavior

### **From Any View:**
- **First Press**: Returns to the previous view
- **Second Press**: If already at previous view, goes to chat (main view)
- **From Chat**: Does nothing (already at main view)

### **Navigation Flow Examples:**
```
Chat → Models → Tools
- From Tools: Esc → Models
- From Models: Esc → Chat  
- From Chat: Esc → (no change)

Chat → Models → Models (refresh)
- From Models: Esc → Chat (since current == previous)

Models → Chat → Models
- From Models: Esc → Chat
- From Chat: Esc → Models
```

## 🔧 Technical Implementation

### **Global Handler** (`app.go:167-177`)
```go
// Escape: Return to previous view or chat if nowhere else to go
if event.Key() == tcell.KeyEscape {
    if a.currentView != a.previousView {
        a.switchToView(a.previousView)
        return nil
    } else if a.currentView != "chat" {
        // If current and previous are the same but not chat, go to chat
        a.switchToView("chat")
        return nil
    }
}
```

### **View-Specific Handling:**

#### **Chat View** (`chat_view.go:212-238`)
- Fixed input handler to properly pass through unhandled events
- Special keys (PgUp, PgDn, arrows) handled locally
- Escape passed to global handler via `WrapInputHandler`

#### **Model View** (`model_view.go:99-132`)
- Table input capture returns unhandled events
- Global escape handling works seamlessly

#### **Tools View**
- No custom input handling
- Global escape works directly

#### **VectorStore View** (`vectorstore_view.go:89-103`)
- Table input capture returns unhandled events  
- Global escape handling works seamlessly

#### **Context Tree View**
- No custom input handling
- Global escape works directly

## 🪟 Modal Escape Handling

### **Download Modal** (`model_view.go:366-372`)
```go
form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
    if event.Key() == tcell.KeyEscape {
        mv.app.SetRoot(mv, true)  // Return to model view
        return nil
    }
    return event
})
```

### **Delete Confirmation Modal** (`model_view.go:279-285`)
```go
form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
    if event.Key() == tcell.KeyEscape {
        mv.app.SetRoot(mv, true)  // Cancel deletion, return to model view
        return nil
    }
    return event
})
```

### **Progress Modal** (Download in progress)
- Cancel button available
- Escape returns to model view (same as Cancel)

### **View Switcher Modal** (`app.go:248`)
- Already has escape handling to return to previous view

## ✅ Behavior Summary

| Current View | Previous View | Escape Action |
|-------------|---------------|---------------|
| Chat | Any | No change |
| Models | Chat | → Chat |
| Tools | Models | → Models |  
| VectorStore | Tools | → Tools |
| Context Tree | Chat | → Chat |
| Models | Models | → Chat |
| Any Modal | N/A | → Parent View |

## 🚀 User Experience

### **Consistent Navigation:**
- Escape always provides a way "back" or "out"
- No dead ends - always a logical escape route
- Predictable behavior across all screens

### **Modal Handling:**
- All modals respect escape key
- Escape = Cancel in all contexts
- Immediate return to parent view

### **Accessibility:**
- Standard escape key convention
- Works alongside existing keyboard shortcuts
- No conflicts with other key bindings

This implementation ensures that users can navigate intuitively and always have a clear path back to where they came from or to the main chat interface.