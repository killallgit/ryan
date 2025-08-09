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

### Changed
- Renamed all references to "orchestrator" to the more generic term "agent" throughout the codebase for better clarity and consistency
- Status bar now displays dynamic icons and states during message processing
- Updated agent's ExecuteStream to use real LangChain streaming with conversation history
- Modified headless and TUI modes to use unified stream.Handler interface
- Renamed StreamSource to RegisteredSource in streaming registry to avoid naming conflicts

### Fixed
- Fixed bug where prompt flag value was incorrectly used in TUI mode
