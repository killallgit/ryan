# Changelog

## [Unreleased]

### Added
- Status bar process icons in TUI to show current system state (↑ sending, ↓ receiving, 🤔 thinking, 🔨 tool usage)
- Shared process state constants package (`pkg/process`) for consistent state management across the application
- Unit tests for process state package with 100% coverage
- Unified streaming architecture in `pkg/stream/` package with real LangChain streaming support
- Stream state tracking (IDLE, STREAMING, COMPLETE, ERROR, CANCELLED) for better UI integration
- Middleware pipeline for stream processing with processors and handlers
- Dedicated stream handlers for console, channel, and buffer outputs
- Real-time streaming using `llms.WithStreamingFunc` instead of simulated chunking
- Agent-level token tracking with `GetTokenStats()` method for decoupled token usage monitoring
- Real-time token counting in status bar during streaming responses
- Thread-safe token accumulation across multiple conversation exchanges
- Comprehensive integration tests for token tracking functionality

### Changed
- Renamed all references to "orchestrator" to the more generic term "agent" throughout the codebase for better clarity and consistency
- Status bar now displays dynamic icons and states during message processing
- Updated agent's ExecuteStream to use real LangChain streaming with conversation history
- Modified headless and TUI modes to use unified stream.Handler interface
- Renamed StreamSource to RegisteredSource in streaming registry to avoid naming conflicts
- Integrated token tracking with new streaming architecture using `tokenAndMemoryHandler`
- Refactored headless runner to use agent's centralized token statistics instead of local counting
- Enhanced status bar to display real-time token counts during streaming

### Fixed
- Fixed bug where prompt flag value was incorrectly used in TUI mode
- Fixed TUI viewport height calculation to prevent crashes when dimensions are too small
- Resolved memory reset functionality to properly clear token counts on conversation restart
