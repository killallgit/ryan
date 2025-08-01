# Ryan - Interactive Chat TUI

## Claude's friend.

A responsive terminal-based chat interface for AI assistants, starting with Ollama integration. Built with Go and tcell for a smooth, non-blocking user experience.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.19+-blue.svg)
![Build Status](https://img.shields.io/badge/build-passing-green.svg)

## Features

🚀 **Non-Blocking Interface** - UI stays responsive during API calls  
⚡ **Real-time Feedback** - Immediate spinner visibility with progress tracking  
🎯 **Enhanced Error Display** - Clear error messages with base16 red colors  
⏰ **Progress Tracking** - Elapsed time display for long-running operations  
🔌 **Connectivity Validation** - Automatic Ollama server status checking  
⌨️ **Escape Key Cancellation** - Cancel operations with Escape key  
📱 **Responsive Layout** - Adapts gracefully to terminal resizing  
🎨 **Clean Architecture** - Functional programming with immutable data structures  

## Quick Start

```shell
# Start a chat session
ryan

# Available keyboard shortcuts:
# Enter - Send message
# Escape - Cancel operation or quit
# Arrow keys - Navigate message history
# Page Up/Down - Scroll through messages
```

## Installation

### Prerequisites

- Go 1.19 or later
- Ollama server running locally (default: http://localhost:11434)

### From Source

```shell
# Clone the repository
git clone https://github.com/your-org/ryan.git
cd ryan

# Install dependencies
go mod download

# Build and install
go build -o ryan ./cmd/ryan
```

### Configuration

Create a configuration file at `~/.ryan/config.yaml`:

```yaml
ollama:
  url: "http://localhost:11434"
  model: "llama3.1:8b"

ui:
  theme: "default"
  spinner_interval: "100ms"
```

## Architecture

Ryan follows a clean, functional architecture with clear separation of concerns:

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

### Key Components

- **Chat Domain** (`pkg/chat/`) - Core business logic, API-agnostic
- **Ollama Integration** (`pkg/ollama/`) - HTTP client for Ollama API
- **TUI Components** (`pkg/tui/`) - Terminal interface using tcell
- **Controllers** (`pkg/controllers/`) - Orchestration between layers

### UI Layout

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

## Development

### Project Structure

```
ryan/
├── cmd/ryan/           # Main CLI entry point
├── pkg/
│   ├── chat/          # Core chat domain logic
│   ├── controllers/   # Business logic orchestration  
│   ├── ollama/        # Ollama API client
│   └── tui/           # Terminal user interface
├── docs/              # Architecture and design docs
├── tests/             # Integration tests
└── Taskfile.yml       # Development tasks
```

### Development Workflow

```shell
# Run tests
task test

# Run integration tests (requires Ollama)
task test:integration  

# Run all tests
task test:all

# Build for development
task build

# Run linter
task lint
```

### Testing

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

## Implementation Phases

Ryan is built using an incremental complexity approach:

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

### 🚧 Phase 3: Streaming Infrastructure (NEXT)
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

## Contributing

We welcome contributions! Please see our development workflow:

1. **Functional Programming First** - Prefer pure functions over stateful objects
2. **Test-Driven Development** - Write tests before implementation
3. **Phase-Based Development** - Never skip phases or add complexity ahead of schedule
4. **Clean Architecture** - Maintain clear separation of concerns

### Code Style

- Use functional paradigms wherever possible
- Avoid OOP principles and "utilities" 
- Write self-documenting, descriptive names
- Follow Go conventions and idioms
- No code comments unless explicitly needed

## Configuration

### Environment Variables

- `RYAN_OLLAMA_URL` - Ollama server URL (default: http://localhost:11434)
- `RYAN_MODEL` - Default model to use (default: llama3.1:8b)
- `RYAN_CONFIG_DIR` - Configuration directory (default: ~/.ryan)

### Config File Options

```yaml
# ~/.ryan/config.yaml
ollama:
  url: "http://localhost:11434"
  model: "llama3.1:8b"
  timeout: "60s"

ui:
  theme: "default"
  spinner_interval: "100ms"
  max_history: 1000

logging:
  level: "info"
  file: "~/.ryan/logs/app.log"
```

## Troubleshooting

### Common Issues

**Spinner not visible when sending messages**
- Fixed in recent version with immediate state synchronization
- Ensure you're running the latest version

**Connection refused errors**
- Check that Ollama is running: `ollama serve`
- Verify URL in config: `~/.ryan/config.yaml`
- Test connectivity: `curl http://localhost:11434/api/tags`

**UI freezing during API calls**
- This was a known issue fixed in Phase 2
- The current version uses non-blocking architecture

**Long response times**
- Large models may take time to respond
- Progress is shown with elapsed time tracking
- Use Escape key to cancel if needed

### Getting Help

- Check the [documentation](./docs/) for detailed architecture info
- Review [TUI patterns](./docs/TUI_PATTERNS.md) for interface behavior
- See [development roadmap](./docs/DEVELOPMENT_ROADMAP.md) for current status

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [tcell](https://github.com/gdamore/tcell) - Terminal control library
- [Ollama](https://ollama.ai/) - Local AI model serving
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management

---

**Status**: Phase 2 Complete ✅ | **Next**: Streaming Implementation 🚧  
**Architecture**: Production Ready 🏗️ | **UX**: Significantly Enhanced 🚀