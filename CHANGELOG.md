# Changelog

## [Unreleased]

### Added
- **Comprehensive Debug Logging** - Enhanced logging throughout the application for better debugging and monitoring
  - Added debug logging to agent initialization, tool setup, and RAG components
  - Added logging to Ollama client connection and model initialization
  - Added logging to chat manager for message handling and memory operations
  - Added logging to tools for permission validation and access control
  - Enabled Bubble Tea debug logging by default for UI debugging
  - Fixed configuration mismatch (logging.preserve → logging.persist)

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
- Unified logging system in `pkg/logger/` package with clean interface (.Debug(), .Info(), .Warn(), .Error(), .Fatal())
- `--persist` CLI flag to control system log persistence across sessions
- Session-based logging with automatic level checking
- **Vector Store Integration** - Added chromem in-memory vector store for Retrieval Augmented Generation (RAG)
  - `pkg/vectorstore/` package with interface definitions and chromem adapter
  - `pkg/embeddings/` package with Ollama embedder support (nomic-embed-text model)
  - `pkg/retrieval/` package with retriever, augmenter, and document management
  - Mock embedder implementation for testing without external dependencies
  - Optional persistence support for vector store data
  - Comprehensive configuration via Viper with sensible defaults
  - Integration with ExecutorAgent for automatic prompt augmentation
  - Document chunking and metadata management capabilities
  - Unit and integration tests for RAG workflow (69% coverage)
- **Agent Package Unit Tests** - Comprehensive test suite for ExecutorAgent
  - Mock LLM implementation following langchain-go patterns
  - Tests for agent creation, execution, streaming, and memory management
  - Concurrent access and thread safety tests
  - Error handling and context cancellation tests
  - Achieved 70.9% code coverage for agent package (up from 0%)
- LangChain-Go tool system with 5 core tools (FileRead, FileWrite, Git, Ripgrep, WebFetch)
- Claude-style ACL permission system using `settings.json` format for tool access control
- `--skip-permissions` flag to bypass all ACL permission checks
- SecuredTool base class for consistent permission checking across all tools
- PermissionManager for pattern-based access control (e.g., `FileRead(*.go)`, `Git(status:*)`)
- Mock vectorstore implementation for future RAG capabilities
- Tool configuration via Viper with individual enable/disable flags
- Integration tests for tools with permission validation


### Changed
- Renamed all references to "orchestrator" to the more generic term "agent" throughout the codebase for better clarity and consistency
- Status bar now displays dynamic icons and states during message processing
- Updated agent's ExecuteStream to use real LangChain streaming with conversation history
- Modified headless and TUI modes to use unified stream.Handler interface
- Renamed StreamSource to RegisteredSource in streaming registry to avoid naming conflicts
- Integrated token tracking with new streaming architecture using `tokenAndMemoryHandler`
- Refactored headless runner to use agent's centralized token statistics instead of local counting
- Enhanced status bar to display real-time token counts during streaming
- Replaced all manual debug level checking with unified logger interface calls
- Updated error handling throughout codebase to use consistent logger methods
- Changed `logging.preserve` configuration to `logging.persist` with proper default (false)

### Fixed
- Fixed bug where prompt flag value was incorrectly used in TUI mode
- Fixed TUI viewport height calculation to prevent crashes when dimensions are too small
- Resolved memory reset functionality to properly clear token counts on conversation restart
- Eliminated scattered logging approaches and inconsistent error handling patterns
- Resolved duplication between manual log setup in headless mode and unified system
