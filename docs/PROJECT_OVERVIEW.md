# Ryan - Project Overview

## What is Ryan?

Ryan is a responsive terminal-based chat interface for AI assistants, starting with Ollama integration. Built with Go and tcell for a smooth, non-blocking user experience.

## Features

🚀 **Non-Blocking Interface** - UI stays responsive during API calls  
⚡ **Real-time Feedback** - Immediate spinner visibility with progress tracking  
🎯 **Enhanced Error Display** - Clear error messages with base16 red colors  
⏰ **Progress Tracking** - Elapsed time display for long-running operations  
🔌 **Connectivity Validation** - Automatic Ollama server status checking  
⌨️ **Escape Key Cancellation** - Cancel operations with Escape key  
📱 **Responsive Layout** - Adapts gracefully to terminal resizing  
🎨 **Clean Architecture** - Functional programming with immutable data structures  
🛠️ **Universal Tool System** - Industry-standard tool calling compatible with all major LLM providers  
🔧 **Built-in Tools** - Bash command execution and file reading with safety constraints  
🔒 **Security First** - Comprehensive safety validation and sandboxing for tool execution  

## UI Layout

```
┌─────────────────────────────────┐
│         Message Area            │ ← Chat history with scroll
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

## Project Structure

```
ryan/
├── cmd/ryan/           # Main CLI entry point
├── pkg/
│   ├── chat/          # Core chat domain logic
│   ├── controllers/   # Business logic orchestration  
│   ├── ollama/        # Ollama API client
│   ├── tools/         # Universal tool system
│   └── tui/           # Terminal user interface
├── docs/              # Architecture and design docs
├── examples/          # Tool system demos and examples
├── integration/       # Integration tests
└── Taskfile.yaml      # Development tasks
```

## Configuration Options

### Environment Variables

- `RYAN_OLLAMA_URL` - Ollama server URL (default: http://localhost:11434)
- `RYAN_MODEL` - Default model to use (default: llama3.1:8b)
- `RYAN_CONFIG_DIR` - Configuration directory (default: ~/.ryan)

### Config File Format

```yaml
# ~/.ryan/settings.yaml
ollama:
  url: "http://localhost:11434"
  model: "llama3.1:8b"
  timeout: "60s"
  system_prompt: "You are a helpful assistant."

ui:
  theme: "default"
  spinner_interval: "100ms"
  max_history: 1000

tools:
  enabled: true
  bash:
    enabled: true
    timeout: "30s"
    allowed_paths: ["~/", "/tmp"]
  file_read:
    enabled: true
    max_file_size: "10MB"
    allowed_extensions: [".txt", ".md", ".go", ".json"]

logging:
  level: "info"
  file: "~/.ryan/logs/app.log"
```

## Development Phases

Ryan was built using an incremental complexity approach:

### ✅ Phase 1: Foundation (COMPLETED)
- Core message and conversation types
- Synchronous HTTP client for Ollama API  
- Basic chat controller
- Simple CLI interface for testing

### ✅ Phase 2: Non-Blocking TUI (COMPLETED)
- Event-driven TUI with tcell
- Non-blocking message sending
- Enhanced spinner visibility and progress tracking
- Alert area with base16 red error colors
- Ollama connectivity validation
- Escape key cancellation support

### 🚧 Phase 3: Streaming Infrastructure (CURRENT)
- HTTP streaming client
- Message chunk accumulation
- Streaming controller with channels

### 📋 Phase 4: TUI + Streaming Integration (PLANNED)
- Real-time message streaming in TUI
- Progressive message display
- Thread-safe streaming updates

### 🎨 Phase 5: Polish & Production (PLANNED)
- Advanced UI features (syntax highlighting, themes)
- Performance optimization
- Configuration and customization options

## Testing Strategy

The project follows test-driven development with comprehensive coverage:

- **Unit Tests** - Pure functions and isolated components
- **Integration Tests** - Real API communication with Ollama
- **TUI Tests** - Event simulation and screen capture
- **Concurrency Tests** - Race detection and deadlock prevention

```shell
# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

## Troubleshooting

### Common Issues

**Connection refused errors**
- Check that Ollama is running: `ollama serve`
- Verify URL in config: `~/.ryan/settings.yaml`
- Test connectivity: `curl http://localhost:11434/api/tags`

**UI freezing during API calls**
- This was a known issue fixed in Phase 2
- The current version uses non-blocking architecture

**Long response times**
- Large models may take time to respond
- Progress is shown with elapsed time tracking
- Use Escape key to cancel if needed