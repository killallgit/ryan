# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ryan is a prototype LLM repl focused on code writing assistance

## Development Commands

### Build and Run
- **Build**: `task build` or `go build -o bin/ryan main.go`
- **Run the TUI**: `task run` or `./bin/ryan`
- **Run headless**: `task build && ./bin/ryan --headless --prompt <prompt>`

### Testing
- **Unit tests**: `task test` or `go test ./pkg/... ./cmd/...`
- **Integration tests**: `task test:integration` (requires Ollama with qwen3:latest model)
- **All tests**: `task test:all`

### Code Quality
- **Full check**: `task check` (runs tidy, verify, fmt, vet, and lint)
- **Lint only**: `task lint` (runs pre-commit hooks)
- **Format code**: `go fmt ./pkg/... ./cmd/...`
- **Verify modules**: `go mod verify`

### Development Tools
- Uses Taskfile for task automation (Task version 3)
- Devbox configuration available with Go and Task pre-installed. IMPORTANT NOTE: When system packages are required, they must be installed in a devbox shell.
- Pre-commit hooks configured via `uvx pre-commit`

## Architecture

### Core Components

1. **Agent System** (`pkg/agent/`)
   - `Agent` interface defines core agent operations (Execute, ExecuteStream, ClearMemory)
   - `ExecutorAgent` implements the main agent using LangChain integration
   - Supports both blocking and streaming execution modes

2. **LLM Integration** (`pkg/llm/`)
   - `Provider` interface for LLM providers
   - `OllamaAdapter` for Ollama integration
   - `TokenTrackingAdapter` wrapper for tracking token usage
   - Registry pattern for managing multiple providers
   - Supports conversational context with message history

3. **Chat Management** (`pkg/chat/`)
   - `Manager` handles conversation state and history
   - `History` manages persistent conversation storage
   - Integrates with memory system for context retention

4. **Memory System** (`pkg/memory/`)
   - SQLite-based memory storage for LangChain
   - Session-based memory management
   - Mock implementation available for testing

5. **Streaming System** (`pkg/stream/`)
   - **Core**: Unified streaming interfaces and handlers (`pkg/stream/core/`)
   - **Providers**: LLM provider implementations (`pkg/stream/providers/`)
   - **TUI Integration**: Bubble Tea message handling (`pkg/stream/tui/`)
   - Supports console, channel, and buffer handlers
   - Middleware pipeline for stream processing

6. **UI Components** (`pkg/tui/`)
   - Bubble Tea-based terminal UI
   - Chat interface with status bar
   - Theme support via Lipgloss

7. **Headless Mode** (`pkg/headless/`)
   - CLI execution without UI
   - Supports scripting and automation
   - Streaming output support

### Entry Points
- `main.go`: Simple entry point that calls cmd.Execute()
- `cmd/root.go`: Command configuration using Cobra
  - Handles both TUI and headless modes
  - Configures LLM providers (Ollama by default)
  - Manages settings via Viper

### Key Patterns

1. **Interface-driven design**: All major components use interfaces for flexibility
2. **LangChain integration**: Uses `tmc/langchaingo` for LLM orchestration
3. **Unified streaming**: Consolidated streaming architecture with core/providers/tui separation
4. **Provider abstraction**: LLM providers are abstracted behind common interfaces
5. **Memory persistence**: SQLite-based conversation memory with session management

## Configuration

- Settings managed via Viper (supports YAML, environment variables)
- Ollama endpoint: Set via `OLLAMA_HOST` environment variable (required)
- Environment variable: `OLLAMA_DEFAULT_MODEL` for model selection
- Config file support via `--config` flag

## Dependencies

- **UI**: Bubble Tea, Bubbles, Lipgloss (Charm libraries)
- **LLM**: LangChain Go (`tmc/langchaingo`)
- **CLI**: Cobra for commands, Viper for configuration
- **Testing**: Testify for assertions
- **Database**: SQLite for memory storage

## Helpful documentation

- **Bubbletea**: https://pkg.go.dev/github.com/charmbracelet/bubbletea
- **viper**: https://github.com/spf13/viper
- **Langchain-go**: https://pkg.go.dev/github.com/tmc/langchaingo@v0.1.13
