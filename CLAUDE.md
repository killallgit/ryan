# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
Ryan is an open-source alternative to Claude Code, built in Go. It provides a terminal-based AI assistant with a rich TUI interface that integrates with local LLMs through Ollama. Ryan features advanced multi-agent orchestration, comprehensive tool integration, hybrid memory management, and vector storage for semantic search.

**Current Status**: ~75% feature parity with Claude Code, focusing on local-first, privacy-focused AI development.

**Key Differentiators**:
- Local LLM inference through Ollama (no cloud dependency)
- Multi-agent architecture with specialized agents
- Rich terminal UI with multiple views
- Hybrid memory system combining conversation buffer and vector storage
- Comprehensive test coverage (58.7% overall, 90%+ for critical packages)

## Development Commands

### Build and Run
- `task build` - Build the main binary to `bin/ryan`
- `task run` - Build and run the application
- `./bin/ryan` - Run the built binary directly
- `./bin/ryan --help` - View command-line options

### Testing
- `task test` - Run all tests with coverage
- `task test:unit` - Run only unit tests (excludes integration)
- `task test:integration` - Run integration tests against Ollama API
- `task test:all` - Run both unit and integration tests
- `task check` - Run full check (tidy, verify, format, vet, unit tests)

#### Coverage Standards
- **Target**: 60%+ coverage for all critical packages
- **Current Status**: See `docs/TEST_COVERAGE_REPORT.md` for detailed analysis
- **High Coverage (≥80%)**: models (91.8%), testutil (89.7%), ollama (85.6%)
- **Good Coverage (60-79%)**: vectorstore (68.8%), logger (63.6%), tools (60.0%)
- **Improved Coverage**: agents (58.7% - recently improved from major refactoring)
- **Needs Improvement (<60%)**: chat, langchain, controllers, config, mcp, tui, cmd

### Specialized Testing
- `task test:models:primary` - Test primary models for tool compatibility
- `task test:models:all` - Test all recommended models
- `task test:embeddings` - Test embedding functionality
- `task test:file-loading` - Test file indexing pipeline
- `task test:vectorstore` - Test vector store operations
- `task test:agents` - Test agent integration

### Environment Variables
- `OLLAMA_URL` - Ollama server URL (default: https://ollama.kitty-tetra.ts.net)
- `OLLAMA_TEST_MODEL` - Model for integration tests (default: qwen2.5-coder:1.5b-base)
- `INTEGRATION_TEST=true` - Enable integration test mode

## Architecture

### Core Components
1. **cmd/root.go** - Main CLI entry point with LangChain controller setup
2. **pkg/controllers/** - Chat controllers (basic and LangChain-based)
3. **pkg/langchain/** - LangChain integration with tool calling
4. **pkg/agents/** - Agent orchestration system with planning and execution
5. **pkg/tools/** - Built-in tools (bash, file operations, git, grep, etc.)
6. **pkg/tui/** - Terminal UI with multiple views (chat, models, tools, vectorstore)
7. **pkg/vectorstore/** - Document indexing and semantic search
8. **pkg/chat/** - Chat conversation management and memory

### Agent System
The agent orchestrator (`pkg/agents/orchestrator.go`) coordinates multiple specialized agents:
- **Dispatcher Agent** - Routes requests to appropriate agents
- **File Operations Agent** - Handles file system operations
- **Code Analysis Agent** - Analyzes code structure and patterns
- **Code Review Agent** - Reviews code changes
- **Search Agent** - Performs code and content searches

### Tool Integration
- Tools are registered in `pkg/tools/registry.go`
- LangChain integration via `pkg/langchain/client.go`
- Agent-specific tool execution through orchestrator
- Model compatibility checking in `pkg/models/compatibility.go`

### Configuration
- Main config in `.ryan/settings.yaml`
- Model definitions in `models.yaml`
- System prompts in `examples/` directory
- Uses Viper for configuration management

### Memory and Vector Storage
- Hybrid memory system combining conversation buffer and vector storage
- ChromemDB-based vector store for document embeddings
- LangChain memory integration for conversation context
- Document chunking and indexing pipeline

## Development Patterns

### Error Handling
- Use structured logging via `pkg/logger`
- Return wrapped errors with context
- Validate inputs early in functions

### Testing
- Unit tests alongside source files (`*_test.go`)
- Integration tests in `integration/` directory
- Fake implementations in `pkg/testutil/`
- Use Ginkgo/Gomega for BDD-style tests where appropriate
- Mock systems in `pkg/testutil/mocks` for complex dependencies
- Test fixtures in `pkg/testutil/fixtures` for consistent test data
- Comprehensive coverage tracking - see `docs/TEST_COVERAGE_REPORT.md`

#### Testing Patterns
- **Unit Tests**: Focus on individual functions and components
- **Integration Tests**: Test component interactions and external dependencies
- **Mock Testing**: Use testify/mock for complex dependency injection
- **BDD Testing**: Use Ginkgo/Gomega for behavior-driven test scenarios
- **Coverage Goals**: Maintain 60%+ coverage for critical packages

### Tool Development
- Implement `tools.Tool` interface
- Register in `tools.Registry`
- Include validation and error handling
- Support both sync and async execution

### Agent Development
- Implement `agents.Agent` interface
- Register with orchestrator
- Handle execution context and progress reporting
- Support task breakdown and result aggregation

## Key File Locations
- **Main binary**: `main.go` → `cmd/root.go`
- **TUI application**: `pkg/tui/app.go`
- **LangChain client**: `pkg/langchain/client.go`
- **Agent orchestrator**: `pkg/agents/orchestrator.go`
- **Tool registry**: `pkg/tools/registry.go`
- **Vector store manager**: `pkg/vectorstore/manager.go`
- **Chat controllers**: `pkg/controllers/factory.go`

## CLI Usage Patterns
- `--config` - Custom config file path
- `--model` - Override model selection
- `--prompt` - Direct prompt execution (non-interactive)
- `--no-tui` - Run without TUI (requires --prompt)
- `--continue` - Continue from previous chat history
- `--ollama.system_prompt` - Custom system prompt file

## Testing Requirements
Always run the full test suite before considering tasks complete. The integration tests require a running Ollama server with compatible models.

## Programming patterns
Always use functional paradigms over OOP style
Always update the documentation when a task is complete
Tasks will not be accepted as complete unless the entire test suite passes with `task test`

## Naming Conventions
The codebase follows clean naming conventions to reduce redundancy and improve readability:

### Interfaces
- Interfaces should not have redundant "Interface" suffix
- Good: `Controller`, `Agent`, `Orchestrator`
- Avoid: `ControllerInterface`, `AgentInterface`

### Package-Scoped Types
- Avoid repeating the package name in type names when context is clear
- Type aliases are provided for cleaner naming while maintaining backward compatibility
- Examples:
  - `controllers` package: `Controller` interface, `Basic`, `LangChain` aliases
  - `tools` package: `Bash`, `Git`, `Tree` instead of `BashTool`, `GitTool`, `TreeTool`
  - `vectorstore` package: `Store` interface, `Indexer`, `Processor` aliases

### Descriptive Names
- Use specific descriptors instead of vague terms like "enhanced", "improved", "better"
- Be explicit about functionality in names and comments
- Example: "LangChain client with ReAct pattern" instead of "enhanced client"

See `docs/NAMING_CONVENTION_REFACTOR.md` for detailed naming guidelines and migration strategy.

## Documentation and Communication
- **README.md** - Project overview, installation, and quick start guide
- **docs/ROADMAP.md** - Detailed feature parity roadmap with Claude Code
- **docs/TEST_COVERAGE_REPORT.md** - Comprehensive test coverage analysis
- **examples/** - Configuration examples and system prompts

## Claude Code Parity Tracking
Ryan aims for feature parity with Claude Code while maintaining local-first architecture:
- ✅ **Core Chat Interface** - Complete parity with rich TUI
- ✅ **File Operations** - Full toolkit for file manipulation
- ✅ **Code Analysis** - Advanced AST parsing and symbol resolution
- ✅ **Git Integration** - Comprehensive git command support
- 🚧 **MCP Integration** - Basic client implementation, expanding
- 📋 **Enterprise Features** - Planned for 2026 (OAuth, RBAC, cloud hosting)

See `docs/ROADMAP.md` for detailed parity analysis and implementation timeline.
