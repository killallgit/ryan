# Changelog

## [Unreleased]

### Added
- **Custom ReAct Agent System** - Replaced orchestrator with componentized ReAct reasoning pipeline
  - Created modular `pkg/agent/react/` package with highly componentized architecture
  - **Controller**: Manages ReAct loop with configurable max iterations
  - **ResponseParser**: Extracts Thought/Action/Action Input/Observation/Final Answer using regex
  - **PromptBuilder**: Constructs ReAct prompts with ExecuteMode and PlanMode support
  - **ToolExecutor**: Executes tools with JSON parameter support and tool name normalization
  - **StateManager**: Tracks conversation state, iterations, and potential answers
  - **DecisionMaker**: Determines when to stop loop based on iterations, stuck detection, and answer quality
  - Visible reasoning output with formatted steps (🤔 Thinking, ⚡ Action, 📝 Input, 👁️ Observation, ✅ Answer)
  - Support for ExecuteMode (direct execution) and PlanMode (planning only)
  - Stream interceptor for real-time formatting of ReAct steps
- **Structured Tool System** - Created JSON schema-based tool parameter system
  - `pkg/tools/structured/` package with Tool wrapper for JSON schemas
  - Parameter definitions with types, descriptions, and required flags
  - Builder pattern for fluent tool construction
  - Migration helpers to convert existing string-based tools to structured format
  - Maintains compatibility with LangChain's tools.Tool interface
- **MRKL Agent** - Simplified agent implementation with operating modes
  - ExecuteMode and PlanMode support with prompt loading from markdown files
  - Integration with tool registry and memory system
  - Token tracking for usage statistics
  - Clean output formatting with ReAct pattern artifact removal
- **Models View with Download/Delete Modal System** - Comprehensive TUI interface for managing Ollama models
  - Created dedicated models view to display available Ollama models in a table
  - Implemented Ollama API client for fetching model list from `/api/tags` endpoint
  - Table shows essential columns: Name, Size, Parameters, and Modified date
  - **Download Modal System** - Interactive model downloading with progress tracking
    - "Pull model" row at top of list opens download modal with text input
    - Real-time progress bar showing download status and completion percentage
    - Animated spinner (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏) with 150ms intervals for visual feedback
    - ESC closes modal but keeps download running in background
    - Downloading models show in list with spinner and dimmed gray styling
    - Ctrl+D cancels active downloads from modal or model list
    - Extended HTTP timeout (30 minutes) to prevent download timeouts
    - Auto-refresh every 5 seconds to show completed downloads
    - Press Enter on downloading model to reopen progress modal
  - **Delete Confirmation Modal** - Safe model deletion with confirmation
    - Ctrl+D shows delete confirmation modal for installed models
    - Cannot delete models while downloading (shows cancellation option instead)
    - Clear confirmation dialog with Enter/Y for confirm, N/ESC for cancel
  - **View History Navigation** - Stack-based navigation for seamless UX
    - ESC key uses view history stack to return to previous view
    - Maximum 10 views in history with LIFO behavior
    - Smart navigation prevents loops and handles edge cases
  - Detailed model information available via 'd' key press showing full metadata in modal
  - Refresh capability with 'r' key to update model list
  - Proper error handling for connection issues
  - Replaced placeholder views (History and Settings) with functional Models view
  - **Code Refactoring** - Broke down large 877-line models.go into 7 focused files
    - `models_types.go` (54 lines) - Type definitions and struct declarations
    - `models_messages.go` (40 lines) - BubbleTea message type definitions
    - `models_download.go` (226 lines) - Download functionality and progress handling
    - `models_delete.go` (72 lines) - Delete operations and confirmations
    - `models_modals.go` (186 lines) - Modal rendering functions
    - `models_table.go` (138 lines) - Table management and cursor logic
    - `models.go` (250 lines) - Main orchestration and view lifecycle
    - Improved maintainability with single-responsibility principle
    - Enhanced readability and easier debugging

### Removed
- **Orchestrator System** - Completely removed orchestrator package and all related components
  - Deleted `pkg/orchestrator/` directory with router, state, feedback, and registry
  - Removed orchestrator integration tests and testing infrastructure
  - Eliminated complex multi-agent routing in favor of single ReAct agent

### Changed
- Refactored headless mode to use ReAct interceptor for visible reasoning
- Updated root command to use new ReAct agent instead of orchestrator
- Modified agent interface to support operating modes

### Added
- **Tool Registry System** - Centralized tool registration and initialization using factory pattern
  - Created `pkg/tools/registry/` package with Registry interface and implementation
  - Factory pattern for tool creation with `ToolFactory` functions
  - Global registry singleton with thread-safe operations
  - Auto-registration of tools via `init()` function in `pkg/tools/init.go`
  - Configuration-based tool enablement with `GetEnabled()` method
  - Simplified agent initialization from ~45 lines to 3 lines
  - Comprehensive unit tests with 96.1% code coverage
  - Support for dynamic tool loading and extension
- **View Switcher with Command Palette** - Multi-view TUI navigation system
  - Implemented command palette modal accessible via `Ctrl+P`
  - Created flexible View interface for all TUI views
  - Added view switcher using bubbles/list component with highlighted selection
  - Centered modal overlay using lipgloss.Place() for proper positioning
  - Smooth navigation with keyboard support (↑/↓ for selection, Enter to switch, Esc to cancel)
  - Modal overlay with proper z-ordering and view updates
  - Integration with chat, models, history, and settings views

### Changed
- Refactored TUI architecture to use View interface for all screens
- Updated main TUI model to handle view switching logic
- Modified view registration to use map-based lookups for efficiency
- Updated CLAUDE.md with view switcher documentation
- Applied consistent rendering patterns across all views

## [2024-10-29]

### Added
- **TUI Testing Infrastructure** - Comprehensive tea test golden file testing for status bar
  - Golden file snapshots for status bar states (normal, spinner, progress, error, multi-status, token)
  - VT100 sequence support for accurate ANSI rendering in tests
  - Automated golden file regeneration with `--update` flag
  - Platform-specific test handling for CI environments
  - Mock spinner implementation for deterministic testing
- **Enhanced Status Bar System** - Comprehensive status management in TUI
  - Multi-status display support with StatusBarUpdateMsg
  - Token counter integration showing sent/received tokens
  - Progress bar rendering for long operations
  - Error state display with proper formatting
  - Connection status and model information display
  - Spinner animation for async operations
- **Streaming Infrastructure Overhaul** - Complete rewrite of streaming architecture
  - Created modular `pkg/stream/` package with core interfaces, providers, and TUI integration
  - Unified Handler interface for all streaming operations
  - Provider-specific implementations for Ollama and OpenAI streaming formats
  - TUI integration with dedicated message types (StreamStartMsg, StreamDeltaMsg, StreamEndMsg)
  - ConsoleHandler for headless mode with real-time output
  - ChannelHandler for concurrent streaming operations
  - BufferHandler for capturing complete streaming output
  - Middleware pipeline support for stream processing
  - Comprehensive test coverage with mock implementations
- **Memory System** - SQLite-based conversation persistence
  - Window buffer memory for conversation history
  - Session-based memory isolation
  - Integration with LangChain memory interface
  - Comprehensive integration tests for memory operations
- **Token Tracking System** - Real-time token usage monitoring
  - `pkg/tokens/` package with thread-safe counter
  - Integration with streaming system via TokenCountingHandler
  - Real-time token counting in status bar during streaming responses
  - Thread-safe token accumulation across multiple conversation exchanges
  - Comprehensive integration tests for token tracking functionality
- Unified logging system in `pkg/logger/` package with clean interface (.Debug(), .Info(), .Warn(), .Error(), .Fatal())
- `--logging.persist` CLI flag to control system log persistence across sessions
- Session-based logging with automatic level checking
- **Vector Store Integration** - Added chromem in-memory vector store for Retrieval Augmented Generation (RAG)
  - `pkg/vectorstore/` package with interface definitions and chromem adapter
  - `pkg/embeddings/` package with Ollama embedder support (nomic-embed-text model)
  - Document loading and chunking capabilities in `pkg/retrieval/loader.go`
  - Configuration support for vector store settings
  - Optional persistence support for vector store data
- **Config System Consolidation** - Centralized configuration using Viper
  - Unified configuration management in `pkg/config/` package
  - Support for YAML config files, environment variables, and CLI flags
  - Automatic config file generation with defaults
  - Environment variable binding (OLLAMA_HOST, OLLAMA_DEFAULT_MODEL, etc.)
  - Comprehensive settings structure covering all components

### Changed
- Refactored streaming to use new unified architecture
- Migrated from direct Ollama streaming to provider-based streaming
- Updated chat manager to use new streaming system
- Improved error handling with centralized error types
- Simplified TUI message handling with dedicated streaming messages
- Modified status bar to integrate with new streaming infrastructure
- **Improved Error Handling** - More robust error management throughout the application
  - Centralized error types in streaming package
  - Better error propagation and user feedback
  - Graceful handling of connection failures and timeouts
- Changed `logging.preserve` configuration to `logging.persist` with proper default (false)
- Updated all components to use centralized config system
- Migrated from per-package settings to unified Settings struct

### Fixed
- Status bar now properly shows real-time token counts during streaming
- Fixed race conditions in concurrent streaming operations
- Resolved memory leaks in long-running streaming sessions
- Fixed status bar updates getting stuck during errors
- Status bar spinner now properly stops on stream completion
- Error messages now properly clear from status bar after timeout
- Fixed CLI flags not properly overriding config file values
- Resolved OLLAMA_HOST environment variable not being respected
- Fixed config file generation creating invalid defaults

### Removed
- Legacy direct Ollama streaming implementation
- Duplicate streaming logic across different components
- Old message type system for streaming updates
- Redundant status update mechanisms
- Removed per-package configuration logic in favor of centralized config
- Eliminated duplicate environment variable handling

## [2024-10-26]

### Added
- Moved to langchain-go library and ollama adapters for LLM operations
- Created generic ExecutorAgent implementation with LangChain integration
- Added memory persistence using SQLite-based memory system
- Introduced Agent interface with Execute and ExecuteStream methods
- Created headless mode for CLI operations
- Added `--prompt` flag for direct prompt execution
- Implemented `--headless` flag for running without TUI
- Support for multiple LLM provider models (Ollama, OpenAI, etc.)
- Basic tool integration with LangChain tools interface

### Changed
- Migrated from custom LLM implementation to LangChain framework
- Refactored agent system to use ExecutorAgent pattern
- Updated streaming to work with LangChain streaming callbacks
- Modified TUI to work with new agent architecture
- Separated concerns between agent logic and UI presentation

### Removed
- Custom Ollama client implementation (replaced with LangChain)
- Direct API calls to Ollama (now handled by LangChain)

## [2024-10-22]

### Initial Release
- Basic TUI chat interface using Bubble Tea
- Direct Ollama API integration
- Simple message history
- Basic streaming support
- File read/write tools
- Bash command execution tool
