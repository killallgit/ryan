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

_for a better experience, install devbox_

```bash
# create the nix environment
devbox shell
# build the binary
task build
# bin can be found at ./bin/ryan
```
