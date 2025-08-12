# Ryan Agent System Refactoring Plan

## Core Objective
- Build a Claude Code-like agent with visible ReAct reasoning
- Leverage existing LangChain-Go components
- Minimize custom implementations

## Current Assets to Keep
- SQLite memory system with conversation persistence
- Streaming infrastructure with unified handlers
- Tool registry pattern with permission system
- Bubble Tea TUI components
- Session management

## Critical Changes Needed

### Tool System
- Replace string-based parameters with structured JSON schemas
- Implement proper parameter validation
- Maintain tools.Tool interface compatibility
- Keep permission system intact

### Agent Architecture
- Replace deprecated ConversationalAgent with modern approach
- Implement visible ReAct loop showing thought/action/observation
- Use LangChain's OneShotAgent or Executor components
- Add streaming interceptor for formatting reasoning steps

### Operating Modes
- ExecuteMode: Direct action and execution
- PlanMode: Planning without execution
- Mode-specific prompt templates
- Runtime mode switching capability

## LangChain-Go Components to Leverage
- agents.Executor for iterative execution
- agents.NewOneShotAgent for zero-shot ReAct
- Output parsers for structured responses
- SqliteChatMessageHistory for memory
- llms.WithStreamingFunc for streaming
- Standard tools.Tool interface

## Implementation Phases

### Phase 1: Tool System Upgrade
- Create structured parameter wrapper
- Migrate FileRead tool as proof of concept
- Test with existing agent
- Roll out to remaining tools incrementally

### Phase 2: ReAct Agent Implementation
- Build agent using LangChain's OneShotAgent
- Create ReAct prompt templates
- Implement reasoning parser for display
- Wire up streaming for real-time visibility

### Phase 3: Mode Support
- Add ExecuteMode with direct action
- Add PlanMode with planning-only behavior
- Implement mode switching mechanism
- Test mode transitions

### Phase 4: Integration and Polish
- Connect new agent to existing TUI
- Maintain memory integration
- Preserve streaming capabilities
- Comprehensive testing

## Success Metrics
- Visible thought/action/observation in output
- No string parsing in tools
- Proper use of LangChain-Go components
- Maintained streaming and memory features
- Working ExecuteMode and PlanMode
- Clean TUI separation

## Risks and Mitigations
- Tool compatibility: Wrap incrementally, test each
- Streaming disruption: Keep existing handlers, add interceptor layer
- Memory loss: Preserve SQLite integration throughout
- TUI coupling: Maintain clean interfaces

## Adjustable Elements
- Tool migration order based on complexity
- Prompt templates can evolve with testing
- Streaming format can be refined based on UX
- Mode behavior can be tuned per feedback
