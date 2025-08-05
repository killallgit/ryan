# Agent System Documentation

## Overview

The agent system provides a flexible, runtime-configurable interface for selecting and using different Langchain agent types. This allows you to choose the best agent for your specific task or model.

## Available Agent Types

### 1. Conversational Agent (`conversational`)
- **Description**: ReAct-style conversational agent with tool usage through natural language
- **Best for**: General conversation, reasoning tasks, explanations
- **Model Requirements**: Good tool compatibility
- **Preferred Models**: Claude, GPT-4, Llama3

### 2. Ollama Functions Agent (`ollama-functions`)
- **Description**: Native Ollama function calling for efficient tool usage
- **Best for**: Tool-heavy tasks, structured operations
- **Model Requirements**: Excellent tool compatibility, Ollama-compatible
- **Preferred Models**: Llama3.1, Qwen2.5, Mistral, DeepSeek, Command-R

### 3. OpenAI Functions Agent (`openai-functions`)
- **Description**: OpenAI-style function calling for structured tool usage
- **Best for**: API integrations, structured data operations, JSON handling
- **Model Requirements**: Excellent tool compatibility, OpenAI-compatible
- **Preferred Models**: GPT-4, GPT-3.5-turbo, Claude-3

## Configuration

### CLI Flags

```bash
# Use a specific agent type
ryan --agent ollama-functions

# Set fallback chain
ryan --fallback-agents "openai-functions,conversational"

# Combine with other options
ryan --agent ollama-functions --model llama3.1
```

### Configuration File

Add to your `.ryan/settings.yaml`:

```yaml
agents:
  # Preferred agent type
  preferred: "ollama-functions"

  # Fallback chain - agents to try if preferred fails
  fallback_chain:
    - "openai-functions"
    - "conversational"

  # Auto-select best agent based on task
  auto_select: true

  # Show agent selection in output
  show_selection: true
```

## Agent Selection Strategy

The system selects agents based on:

1. **User Preference**: Explicitly specified agent via CLI or config
2. **Task Analysis**: Automatic analysis of the request to determine best agent
3. **Model Compatibility**: Matching agent capabilities with model features
4. **Fallback Chain**: Sequential fallback to alternative agents if needed

### Selection Priority

1. CLI flag `--agent` (highest priority)
2. Config file `agents.preferred`
3. Automatic selection based on task analysis
4. Fallback chain

## Architecture

### Core Components

1. **LangchainAgent Interface**: Base interface for all agents
2. **AgentFactory**: Dynamic agent creation and registration
3. **LangchainOrchestrator**: Agent selection and execution orchestration
4. **Agent Implementations**: Concrete implementations for each agent type

### Adding Custom Agents

```go
// 1. Implement the LangchainAgent interface
type MyCustomAgent struct {
    agents.BaseLangchainAgent
    // ... custom fields
}

// 2. Register with the factory
agents.GlobalFactory.Register("my-agent", NewMyCustomAgent)

// 3. Use via CLI or config
ryan --agent my-agent
```

## Examples

### Tool-Heavy Task
```bash
# Ollama Functions agent will be auto-selected for tool operations
ryan -p "List all Python files and count lines of code"
```

### Reasoning Task
```bash
# Conversational agent will be preferred for explanations
ryan --agent conversational -p "Explain how recursion works"
```

### With Fallback
```bash
# Try ollama-functions first, fall back to conversational if needed
ryan --agent ollama-functions --fallback-agents "conversational"
```

## Troubleshooting

### Agent Not Available
- Check model compatibility with `models.GetModelInfo()`
- Ensure Ollama server version supports function calling (v0.4.0+)
- Verify tool registry is properly initialized

### Agent Selection Issues
- Enable verbose logging to see selection process
- Use `--agent` flag to force specific agent
- Check fallback chain configuration

### Performance
- Ollama Functions: Best for local models with tool support
- OpenAI Functions: Optimal for cloud-based models
- Conversational: Most flexible, works with any model

## Best Practices

1. **Match Agent to Task**: Use function agents for tool-heavy tasks
2. **Configure Fallbacks**: Always set a fallback chain for reliability
3. **Test Model Compatibility**: Verify your model supports the agent type
4. **Monitor Selection**: Enable `show_selection` to understand agent choices
