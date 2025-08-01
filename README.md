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
_for a better experience, install devbox_

```bash
# create the nix environment
devbox shell
# build the binary
task build
# bin can be found at ./bin/ryan
```

## Configuration
docs TODO

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