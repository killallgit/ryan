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
- **Models View** - New TUI view for managing Ollama models
  - Created dedicated models view to display available Ollama models in a table
  - Implemented Ollama API client for fetching model list from `/api/tags` endpoint
  - Table shows essential columns: Name, Size, Parameters, and Modified date
  - Detailed model information available via 'd' key press showing full metadata in modal
  - Refresh capability with 'r' key to update model list
  - Proper error handling for connection issues
  - Navigation with escape key to return to previous view
  - Replaced placeholder views (History and Settings) with functional Models view

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
  - Three initial views: Chat, Settings (shows Ollama config), and History (recent messages)
  - Navigation with j/k or arrow keys, Enter to select, Esc to cancel
  - Modal automatically sizes to content with rounded border styling
  - Proper state preservation when switching between views
- **Orchestrator Testing Framework** - Comprehensive testing infrastructure for multi-agent orchestrator system
  - Mock LLM with configurable responses, intent analysis, and behavior simulation
  - Mock agents with tool calling simulation, failure rates, and retry logic
  - Scenario-based test utilities with fluent builders for complex multi-agent workflows
  - Comprehensive assertions library for routing, tool calls, status, and execution flow validation
  - 25+ test scenarios covering simple, complex, failure, performance, and regression cases
  - Support for partial success behaviors and infinite loop detection for max iteration testing
  - Smart tool call simulation based on instruction content (file operations, bash commands, git operations)
  - Intent analysis with keyword-based classification for test scenario routing
- **TUI Testing with teatest** - Implemented golden file testing for Bubble Tea components
  - Added teatest from github.com/charmbracelet/x/exp/teatest for TUI testing
  - Created comprehensive test suite for status bar component with golden file snapshots
  - Tests cover inactive/active states, token display, and all process states
  - Fixed teatest hanging issue with proper model termination wrapper
  - Achieved 50% code coverage for status bar package

- **Centralized Configuration System** - Consolidated all configuration into a single global Settings object
  - Created `pkg/config/init.go` with strongly-typed configuration structure
  - Migrated all Viper defaults and settings to centralized package
  - Removed scattered `viper.Get*()` calls throughout the codebase
  - Global `config.Get()` function provides type-safe access to all settings
  - Improved test initialization with proper config setup
  - Environment variable support maintained (OLLAMA_HOST, OLLAMA_DEFAULT_MODEL)
  - Configuration is decoupled from UI modes and loaded upfront for better testability
- **Comprehensive Debug Logging** - Enhanced logging throughout the application for better debugging and monitoring
  - Added debug logging to agent initialization, tool setup, and RAG components
  - Added logging to Ollama client connection and model initialization
  - Added logging to chat manager for message handling and memory operations
  - Added logging to tools for permission validation and access control
  - Enabled Bubble Tea debug logging by default for UI debugging
  - Fixed configuration mismatch (logging.preserve → logging.persist)
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
- `--logging.persist` CLI flag to control system log persistence across sessions
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
