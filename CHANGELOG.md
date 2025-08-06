# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **TUI**: Unified RenderManager for consistent text rendering across all views
  - Added RenderManager to App struct for centralized text formatting
  - Implemented helper methods for common UI patterns (lists, tables, status, progress bars, trees)
  - Updated all TUI views (ModelView, ToolsView, VectorStoreView, ContextTreeView) to use RenderManager
- **Testing**: Dedicated coverage directory for test reports
  - Created `coverage/` directory for all coverage output files
  - Added `test:coverage` task to generate HTML coverage reports
  - Updated `.gitignore` to exclude coverage files from version control

### Changed
- **TUI**: Refactored chat view to follow TUI.md component layout pattern
  - Restructured components to match specification: MESSAGE_NODES, STATUS_CONTAINER, CHAT_INPUT_CONTAINER, FOOTER_CONTAINER
  - Added thin border around input field with customizable color (ColorBase01)
  - Consolidated status display with left-justified text next to spinner
  - Removed separate activity view and integrated into STATUS_CONTAINER
  - Updated status components to show agent, action, and state information in a single line
  - Improved visual hierarchy and component organization

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
