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
# ? - Show help modal with all shortcuts
# Ctrl+N - Switch between input and node selection modes

# Node Selection Mode (when enabled):
# j/k - Navigate up/down between message nodes
# Tab - Expand/collapse focused node  
# Space - Select/deselect focused node
# Enter - Toggle selection of focused node
# a - Select all nodes
# c - Clear all selections
# i/Esc - Return to input mode
```

## Interactive Node System

Ryan features an interactive node-based chat interface that allows you to navigate, select, and manipulate individual chat messages as discrete nodes.

### Mode System

**Input Mode (default)**: Standard text input with message history scrolling
- Status bar shows: `✏️ Input | Ctrl+N=node mode, ?=help`

**Node Selection Mode**: Interactive navigation and selection of message nodes  
- Status bar shows: `🎯 Node Select | Focused: <nodeID> | shortcuts...`
- Use `Ctrl+N` to switch between modes  
- Press `?` to show the help modal with all shortcuts
- Click on any message to automatically switch to node mode and focus that message

### Node Features

- **Visual Indicators**: Focused nodes have a gray background, selected nodes have blue background
- **Collapsible Content**: Long messages and thinking blocks can be expanded/collapsed with Tab
- **Multi-Selection**: Select multiple nodes for bulk operations
- **Vim-Style Navigation**: Use j/k keys for intuitive up/down movement

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
