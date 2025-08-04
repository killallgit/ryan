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
🛠️ **Advanced Tool System** - Claude Code-level tool execution with concurrent orchestration  
🔧 **Comprehensive Tool Suite** - 15+ production-ready tools with batch execution capability  
🔒 **Security First** - Comprehensive safety validation and sandboxing for tool execution  
🔄 **Multi-Provider Support** - Universal tool calling for OpenAI, Anthropic, and Ollama  

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
│   ├── providers/     # Multi-provider adapters (OpenAI, Anthropic)
│   ├── tools/         # Advanced tool execution system
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

## Development Status

Ryan follows an incremental development approach with clear phases:

### ✅ Phase 1: Foundation (COMPLETED)
- Core message and conversation types
- Synchronous HTTP client for Ollama API  
- Basic chat controller with conversation management
- Configuration system with Viper integration

### ✅ Phase 2: Non-Blocking TUI (COMPLETED)
- Event-driven TUI with tcell framework
- Non-blocking message sending with goroutines
- Enhanced spinner and progress tracking
- Alert area with error display and base16 colors
- Ollama connectivity validation and user guidance
- Escape key cancellation for long operations

### ✅ Phase 3A: Streaming Implementation (COMPLETED)
- ✅ HTTP streaming client with chunk processing
- ✅ Message accumulation with Unicode handling
- ✅ Thread-safe streaming updates in TUI
- ✅ Real-time message display during streaming
- ✅ Automatic fallback for non-streaming clients
- ✅ Error recovery and connection management

### ✅ Phase 3B: Production Tool System (COMPLETED)
- ✅ Universal tool interface supporting multiple providers
- ✅ Built-in tools: `execute_bash` and `read_file` with safety constraints
- ✅ Provider adapters for OpenAI, Anthropic, Ollama formats
- ✅ Complete tool calling integration with streaming chat
- ✅ Comprehensive safety framework and security validation
- ✅ Model compatibility validation (41 compatible models identified)

### 🚧 Phase 3C: Tool System Expansion (IN PROGRESS)
- 🚧 Advanced tool execution engine with concurrent orchestration
- 🚧 Expanded tool suite (WebFetch, Grep, Git integration, etc.)
- 🚧 Batch tool execution ("multiple tools in single response")
- 🚧 Real-time tool progress tracking in TUI

### 📋 Phase 4: Advanced Features (PLANNED)
- User consent system for dangerous operations
- Tool execution sandboxing and resource limits
- Audit logging and execution history
- MCP protocol support for external tool providers

### 🎨 Phase 5: Polish & Optimization (PLANNED)
- Advanced UI features (syntax highlighting, themes)
- Performance optimization and result caching
- Custom tool development SDK
- Enhanced configuration and customization options

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