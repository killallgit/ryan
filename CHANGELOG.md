# Changelog

All notable changes to Ryan will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Security
- **Tool Safety Features**:
  - Command validation with forbidden command blocking
  - Path restrictions for file operations
  - Resource limits (file size, execution timeout)
  - Input sanitization for all tool parameters
  - Working directory validation

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