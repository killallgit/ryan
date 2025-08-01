# Ryan

A terminal chat interface for AI models with universal tool support. Built for code co-pilot functionality with industry-standard tool calling compatibility across all major LLM providers.

## Quick Start

```bash
# Start chatting with tool support
ryan

# Try the tool system demo
go run examples/tool_demo.go

# Keyboard shortcuts:
# Enter - Send message
# Escape - Cancel/quit
# Tab - Switch between chat and model management
```

## Installation

### Prerequisites
- Go 1.19+
- [Ollama](https://ollama.ai/) running locally

### Build from source
_for a better experience, install devbox_

```bash
# create the nix environment
devbox shell
# build the binary
task build
# bin can be found at ./bin/ryan
```

## Features

🚀 **Non-Blocking Interface** - UI stays responsive during API calls and tool execution  
🛠️ **Universal Tool System** - Industry-standard tool calling compatible with OpenAI, Anthropic, Ollama, and MCP  
🔧 **Built-in Tools** - Bash command execution and file reading with comprehensive safety constraints  
🔒 **Security First** - Path validation, command filtering, and resource limits  
⚡ **Real-time Feedback** - Progress tracking and error handling with visual indicators  
🎨 **Clean Architecture** - Functional programming with immutable data structures  

## Tool System

Ryan includes a universal tool system that works with all major LLM providers:

### Available Tools
- **execute_bash** - Run shell commands with safety constraints
- **read_file** - Read file contents with path validation and line range support

### Provider Compatibility
- **OpenAI** - Function calling format
- **Anthropic** - Tool use format  
- **Ollama** - Compatible with OpenAI format
- **MCP** - Model Context Protocol support

### Example Usage
```bash
# Demo the tool system
go run examples/tool_demo.go

# See provider format examples
# Tools are automatically converted to each provider's expected format
```

## Configuration

Configuration is managed through `~/.ryan/settings.yaml`:

```yaml
ollama:
  url: "http://localhost:11434"
  model: "llama3.1:8b"
  timeout: "60s"

tools:
  enabled: true
  bash:
    enabled: true
    timeout: "30s"
    allowed_paths: ["~/", "/tmp"]
  file_read:
    enabled: true
    max_file_size: "10MB"
    allowed_extensions: [".txt", ".md", ".go", ".json"]

ui:
  theme: "default"
  max_history: 1000
```

## Development

```bash
# Run tests
task test

# Build
task build

# Run checks (lint, format, etc)
task check
```

## Contributing

1. Write tests first
2. Use functional programming patterns
3. Keep code simple and readable
4. Follow the existing code style

## License

MIT