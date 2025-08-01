# Ryan

A terminal chat interface for AI models with Ollama integration.

## Quick Start

```bash
# Start chatting
ryan

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
```bash
git clone https://github.com/killallgit/ryan.git
cd ryan
task build
```

## Configuration

Create `~/.ryan/settings.yaml`:
```yaml
ollama:
  url: "http://localhost:11434"
  model: "llama3.1:8b"
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