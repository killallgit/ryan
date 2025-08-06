# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **LLM-Based Intent Detection**: Implemented intelligent intent analysis using actual LLMs
  - Created `LLMIntentAnalyzer` to replace pattern matching with Ollama-powered intent detection
  - Orchestrator now uses LLM reasoning to determine user intent and route to appropriate agents
  - Added `GeneralAgent` for handling conversational requests with LLM integration
  - Enhanced intent classification with confidence scores and tool requirements detection
  - **BREAKING**: Removed ALL keyword matching from agents - `CanHandle` methods now trust orchestrator's LLM routing
  - General agent now uses LangChain's `llms.Model` interface for actual LLM calls
  - Planner's secondary intent detection no longer uses keyword matching
  - Search agent pattern extraction simplified without keyword dependency
- **TUI**: Component-based modal system with improved error handling
  - Created reusable BaseModal component for modal composition and lifecycle management
  - Added specialized DownloadModal for model downloads with real-time progress tracking
  - Added ErrorModal with proper text wrapping, center alignment, and flexible sizing
  - Implemented proper keyboard navigation (Enter, Escape, Tab cycling) for all modals
  - Added Ollama health check functionality with configurable timeouts
- **TUI**: Unified RenderManager for consistent text rendering across all views
  - Added RenderManager to App struct for centralized text formatting
  - Implemented helper methods for common UI patterns (lists, tables, status, progress bars, trees)
  - Updated all TUI views (ModelView, ToolsView, VectorStoreView, ContextTreeView) to use RenderManager
- **Testing**: Dedicated coverage directory for test reports
  - Created `coverage/` directory for all coverage output files
  - Added `test:coverage` task to generate HTML coverage reports
  - Updated `.gitignore` to exclude coverage files from version control

### Fixed
- **Application Startup Performance**: Resolved 60+ second freeze when Ollama is unreachable
  - Reduced Ollama version check timeout from 30s to 3s for faster failure detection
  - Made model refresh asynchronous in TUI to prevent UI blocking
  - Reduced TUI Ollama client timeout to 5s for responsive model loading
  - Added 3s timeout for LLM intent analysis to prevent request blocking
  - Application now fails fast with clear error messages instead of hanging
- **Tool Registry Requirements**: Made tool registry mandatory for proper operation
  - Removed fallback behavior when tools are unavailable (tools are core functionality)
  - Application fails early with descriptive errors when Ollama server is unreachable
  - Enhanced error messages with HTTP status codes and full request URLs for better debugging
### Changed
- **Agent Routing Architecture**: Complete removal of keyword-based routing
  - All agents now return `true/1.0` from `CanHandle()` - trusting orchestrator's LLM decision
  - Factory's `CreateBestAgent` no longer checks `CanHandle` confidence scores
  - Orchestrator passes LLM model to agents for conversational responses
  - Updated all agent initialization to support LLM model injection
- **TUI**: Refactored chat view to follow TUI.md component layout pattern
  - Restructured components to match specification: MESSAGE_NODES, STATUS_CONTAINER, CHAT_INPUT_CONTAINER, FOOTER_CONTAINER
  - Added thin border around input field with customizable color (ColorBase01)
  - Consolidated status display with left-justified text next to spinner
  - Removed separate activity view and integrated into STATUS_CONTAINER
  - Updated status components to show agent, action, and state information in a single line
  - Improved visual hierarchy and component organization

### Fixed
- **TUI**: Modal system no longer shows overlapping modals during error states
  - Fixed modal replacement logic to properly remove previous modals before showing new ones
  - Eliminated debug output bleeding through to the terminal UI during modal operations
  - Improved error modal text formatting with proper center alignment and text wrapping
- **TUI**: Modal keyboard interactions now work consistently across all modal types
  - Fixed focus management to ensure proper initial focus on input fields
  - Added proper Tab navigation between modal components
  - Fixed Enter/Escape key handling in modal contexts

### Changed
- **Configuration**: Consolidated all environment variable access through Viper
  - Replaced direct `os.Getenv()` calls with `viper.GetString()` across config hierarchy
  - Added comprehensive `viper.BindEnv()` bindings for all environment variables
  - Updated OpenAI API key handling in vectorstore to use Viper
  - Fixed MCP servers and config directory override to use Viper consistently
  - Ensured single source of truth for configuration management
- **Testing**: Updated integration tests to use LangChain controllers and Viper configuration
  - Migrated all integration tests from direct Ollama clients to LangChain controllers
  - Standardized configuration management using Viper with proper environment variable handling
  - Updated agent orchestrator tests to use `InitializeLangChainController`
  - Modernized LangChain integration tests with comprehensive controller testing
  - Fixed package naming inconsistencies across integration test files
  - Ensured tests fail appropriately when components cannot be initialized
  - Maintained full test coverage while reflecting current system architecture
- **BREAKING**: Major naming convention refactoring across the codebase
  - Removed all "enhanced" references (35+ occurrences) with specific, meaningful descriptions
  - Cleaned up interface naming by removing redundant "Interface" suffix
  - Renamed `ChatControllerInterface` → `Controller`
  - Renamed agent interfaces: `OrchestratorInterface` → `Orchestrator`, `PlannerInterface` → `Planner`, etc.
  - Renamed `OllamaClientInterface` → `OllamaClient` in tools package
  - Updated logger component from `langchain_enhanced` to `langchain_client`
  - Renamed variable `enhancedMessages` to `messagesWithContext` in ollama_tools.go
- Refactored root initialization to improve modularity and separation of concerns
- Moved application logic from `cmd/root.go` into separate modules (`cmd/app.go`, `cmd/adapter.go`)
- Created initialization helpers in `pkg/agents/init.go` and `pkg/controllers/init.go`
- Improved error handling and logging during initialization
- **BREAKING**: Refactored model configuration to be provider-specific
  - Model is now configured per provider (`ollama.model`, `openai.model`)
  - Added `provider` field to select active LLM provider
  - CLI flags changed to provider-specific: `--ollama.model`, `--openai.model`
- Refactored LangChain agent selection logic for improved tool integration
- Simplified agent type determination to always use conversational agent when tools are available
- Removed unused OllamaToolCaller in favor of unified LangChain approach
- Enhanced agent selection display in UI with confidence scores
- Improved output processing to filter special markers from responses
- Updated orchestrator integration in controller for better agent routing

### Added
- Type aliases for cleaner naming while maintaining backward compatibility
  - `pkg/controllers/aliases.go`: `Basic`, `LangChain`, `LCChat` aliases
  - `pkg/tools/aliases.go`: `Bash`, `Git`, `Tree` etc. without "Tool" suffix
  - `pkg/vectorstore/aliases.go`: `Store`, `Indexer`, `Processor` aliases
- Comprehensive naming convention documentation in `docs/NAMING_CONVENTION_REFACTOR.md`
- Naming conventions section in `CLAUDE.md` with guidelines
- Support for multiple LLM providers in configuration structure
- OpenAI configuration structure for future implementation
- Helper methods for accessing active provider configuration
- Provider selection via `--provider` CLI flag

### Removed
- Removed ScrumMaster agent and related tests (functionality can be achieved through orchestrator)
- Removed generic `--model` flag (replaced with provider-specific flags)

### Fixed
- Improved code readability by eliminating vague descriptors
- Reduced naming redundancy across packages
- Improved pre-commit hook compliance with proper file formatting
- Fixed duplicate tool mode markers appearing in UI responses
- Improved agent decision visibility in chat interface
