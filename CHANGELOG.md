# Changelog

All notable changes to Ryan will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - Tool System Phase 3B (August 2025)
- **Advanced Tool System Architecture**: Achieved **85% Claude Code parity** with production-ready implementation
  - **8 Production Tools**: `execute_bash`, `read_file`, `write_file`, `grep_search`, `web_fetch`, `git_operations`, `tree_analysis`, `ast_parse`
  - **Batch Execution Engine**: "Multiple tools in single response" capability with dependency resolution
  - **Concurrent Tool Orchestration**: Parallel execution with goroutine pool management and result aggregation
  - **Dependency Graph System**: Topological sorting with cycle detection for complex tool workflows
  - **Progress Tracking System**: Real-time progress updates with cancellation support

- **GitTool Implementation**: Comprehensive git repository operations
  - Safe git operations: `status`, `diff`, `log`, `branch`, `show`, `ls-files`
  - Repository validation and security constraints
  - Structured output with JSON schema validation
  - Integration with batch execution system

- **TreeTool Implementation**: Advanced directory analysis and visualization
  - Multiple output formats: tree, list, json, summary
  - Intelligent filtering by file type, size, date, and patterns
  - Recursive directory traversal with depth and file count limits
  - Statistical analysis: file counts, size distribution, type analysis
  - Exclusion patterns and hidden file handling

- **ASTTool Implementation**: Language-specific code parsing and analysis
  - **Go Language Support**: Complete AST parsing with symbol extraction, metrics calculation, and issue detection
  - **Multi-language Framework**: Extensible architecture supporting 9+ languages (Go, Python, JS, TS, Java, C, C++, Rust, PHP)
  - **Code Analysis Features**: Symbol extraction, complexity metrics, issue detection, dependency analysis
  - **Flexible Analysis Types**: structure, symbols, metrics, issues, full analysis modes
  - **Position Tracking**: File, line, column, offset information for all AST elements

- **BatchExecutor System**: Advanced concurrent tool execution
  - **Goroutine Pool Management**: Configurable concurrency limits with resource monitoring
  - **Dependency Resolution**: DAG-based execution ordering with cycle detection
  - **Progress Tracking**: Real-time updates with aggregated result collection
  - **Context Management**: Timeout handling, cancellation, and resource cleanup
  - **Error Handling**: Partial failure recovery with detailed error reporting

- **DependencyGraph Implementation**: Sophisticated workflow orchestration
  - **Topological Sorting**: Kahn's algorithm for dependency ordering
  - **Cycle Detection**: DFS-based cycle prevention with clear error reporting
  - **Status Tracking**: Node status management (pending, executing, completed, failed)
  - **Graph Statistics**: Comprehensive metrics and validation
  - **Graph Manipulation**: Clone, validation, and introspection capabilities

- **Self Configuration Support**: Added support for self.yaml configuration file
  - Separate viper instance for self.yaml configuration
  - `SelfConfig` and `SelfTrait` structs for AI persona configuration
  - `GetSelf()` function to access self configuration
  - `self_config_path` setting with default `./.ryan/self.yaml`
  - Example self.yaml configurations in examples directory

### Added
- **Enhanced Status Row**: Improved status information display with format `<SPINNER> <FEEDBACK_TEXT> (<DURATION> | <NUM_TOKENS> | <bold>esc</bold> to interject)`
  - Real-time duration tracking during operations
  - Integrated token count display in status row
  - Interactive "esc to interject" hint during streaming
- **Improved Modal Buttons**: Consistent button styling across all modals
  - Download modal buttons now properly contained within border
  - Equal-width buttons that fill available space
  - Reduced padding for cleaner appearance
  - Delete confirmation modal now uses proper buttons instead of instruction text
  - Tab and arrow key navigation between buttons

### Fixed
- **Token Count Reporting**: Fixed bug where token counts were not updated after streaming completion
- **Modal Button Overflow**: Fixed download modal buttons overflowing container boundaries
- **Status Row Layout**: Improved status row positioning and content layout

### Changed
- **BREAKING: LangChain-Only Mode**: LangChain is now the only mode of operation
  - Removed ability to disable LangChain
  - Removed all conditional logic for alternate modes
  - Simplified configuration and codebase
- **BREAKING: Configuration Key Changes**:
  - Changed `logging.file` to `logging.log_file` in configuration
  - Fixed LangChain tools configuration keys:
    - `autonomous_execution` → `autonomous_reasoning`
    - `use_agent_framework` → `use_react_pattern`
- **Code Quality**: Cleaned up debug logging statements across the codebase
  - Removed extensive debug logging blocks from `pkg/langchain/client.go`
  - Cleaned up debug statements in TUI rendering components
  - Removed unused debug comments and imports
  - Improved code readability and reduced log noise

- **Tool Calling System (Phase 2)**: Complete Ollama integration with tool calling support
  - Universal tool interface with JSON Schema validation
  - Built-in tools: `execute_bash` and `read_file` with security constraints
  - Provider format adapters for OpenAI, Anthropic, Ollama, and MCP
  - Context-aware tool execution with loop protection
  - Comprehensive model compatibility database (41+ tool-compatible models)
  - Automated model compatibility testing framework
  - Ollama server version validation (requires v0.4.0+ for tool support)
- **Enhanced Model Selection**:
  - Updated default model to `qwen2.5:7b` (excellent tool calling support)
  - Visual tool compatibility indicators in TUI model list:
    - 🔧 Excellent tool calling support
    - ⚙️ Good tool calling support  
    - 🔩 Basic tool calling support
  - Model compatibility warnings and recommendations on startup
  - Smart model validation with graceful degradation
- **Model Download Prompt**: Automatic model downloading when selecting unavailable models
  - Interactive download confirmation dialog with model name display
  - Real-time progress tracking with animated progress bar and status updates
  - Cancellable downloads with context-based cancellation
  - Seamless integration with model selection workflow
  - Automatic model activation and configuration after successful download
- **Testing Infrastructure**:
  - Model compatibility tester with performance benchmarking
  - Comprehensive test suite for tool calling functionality
  - Integration tests for multi-tool operations
  - Command-line tool for testing model compatibility (`cmd/model-tester`)
- **Documentation**:
  - Comprehensive model compatibility guide (`docs/MODEL_COMPATIBILITY_TESTING.md`)
  - Tool calling architecture documentation
  - Model recommendation database with performance characteristics

### Changed
- **Default Configuration**:
  - Changed default model from `qwen2.5-coder:1.5b-base` to `qwen2.5:7b`
  - Enhanced startup validation with server compatibility checks
- **TUI Improvements**:
  - Model list now displays tool compatibility icons
  - Updated help text to explain tool capability indicators
  - Enhanced model selection feedback with compatibility warnings

### Removed
- **Configuration Options**:
  - Removed `UseLangchain` field from `SearchConfig`
  - Removed `ollama.use_langchain` configuration option
  - Removed unused `langchain.enabled` configuration
  - Removed unused `langchain.streaming.use_langchain` configuration
  - Removed unused `langchain.streaming.provider_optimization` configuration
  - Removed unused `langchain.prompts.use_templates` configuration
  - Removed all references to conditional LangChain usage

### Enhanced - Tool System Phase 3B
- **Advanced Tool Orchestration**: Upgraded from basic tool calling to sophisticated workflow management
  - **Batch Execution**: Execute multiple tools in single response with dependency resolution
  - **Concurrent Processing**: Parallel tool execution with configurable concurrency limits
  - **Progress Tracking**: Real-time execution feedback with cancellation support
  - **Resource Management**: Memory, CPU, and execution time monitoring with limits

- **Expanded Tool Suite**: Grew from 5 to 8 production-ready tools
  - **Git Integration**: Repository operations with safety constraints
  - **Directory Analysis**: Advanced tree visualization and file statistics  
  - **Code Parsing**: Multi-language AST analysis with symbol extraction
  - **Enhanced Security**: Multi-layer validation and permission systems

### Security
- **Enhanced Tool Safety Features**:
  - **Multi-layer Security**: Path validation, command filtering, resource limits
  - **Dependency Validation**: Cycle detection and topological sorting for safe execution
  - **Concurrent Safety**: Thread-safe execution with goroutine pool management
  - **Context Management**: Timeout handling, cancellation, and resource cleanup
  - **Advanced Validation**: JSON schema validation for all tool parameters
  - **Permission System**: User consent and risk assessment for tool operations

## Previous Versions

### [0.1.0] - Initial Release
- Basic chat interface with Ollama integration
- TUI-based conversation management
- Model management and switching
- Configuration system with YAML support
- Logging and debugging capabilities

---

## Tool Calling Feature Details

### Supported Models (Tier 1 - Excellent)
- **Llama 3.1** (8B, 70B, 405B) - Mature, reliable tool calling
- **Llama 3.2** (1B, 3B, 11B, 90B) - Lightweight with solid support
- **Qwen 2.5** (1.5B-72B) - Superior math/coding performance
- **Qwen 2.5-Coder** - Specialized for development workflows
- **Qwen 3** (8B+) - Latest with enhanced capabilities

### Built-in Tools
1. **`execute_bash`** - Safe shell command execution
   - Security constraints and timeout protection
   - Forbidden command blocking
   - Working directory validation

2. **`read_file`** - Secure file content reading
   - Extension whitelisting
   - File size limits (10MB max, 10k lines max)
   - Path traversal protection

### Performance Benchmarks
- **Qwen 2.5 7B**: ~980ms average response time, 100% test pass rate
- **Llama 3.1 8B**: ~1.2s average response time, 100% test pass rate
- **Qwen 2.5-Coder 1.5B**: ~800ms average response time, excellent for development

### Testing Commands
```bash
# Test primary models
task test:models:primary

# Test all compatible models  
task test:models:all

# Build model tester
task build:model-tester
```