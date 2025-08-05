# Package Refactoring Plan

## Current Issues
1. **pkg/agents** - Too many responsibilities in a single package (18 files)
2. **Test naming** - Some tests have non-descriptive names like "Additional" or "Coverage"
3. **Package cohesion** - Some packages mix different concerns

## Lessons Learned from Refactoring Attempt

### Why the agents package refactoring was reverted
1. **High coupling** - The agents package has many interdependencies between types
2. **Circular dependencies** - Orchestrator, Planner, Executor all reference each other
3. **Interface definitions** - Agent interface is used throughout the codebase
4. **Import complexity** - Would require updating imports in many files outside pkg/agents

### Recommendation
- Keep agents package as-is for now
- Focus on smaller, incremental improvements
- Consider refactoring only after reducing coupling between components

## Proposed Package Structure (Future Consideration)

### pkg/agents → Split into sub-packages

```
pkg/agents/
├── core/                  # Core agent interfaces and orchestration
│   ├── orchestrator.go
│   ├── executor.go
│   ├── planner.go
│   ├── protocol.go
│   ├── types.go
│   └── context.go
├── builtin/              # Built-in agent implementations
│   ├── dispatcher.go
│   ├── code_analysis.go
│   ├── code_review.go
│   ├── file_operations.go
│   ├── search.go
│   └── conversational.go
├── llm/                  # LLM-specific agents
│   ├── langchain_agent.go
│   ├── langchain_orchestrator.go
│   ├── ollama_functions_agent.go
│   └── openai_functions_agent.go
├── factory/              # Agent creation
│   └── factory.go
└── feedback/             # Feedback mechanisms
    └── feedback.go
```

### pkg/chat → Already well-organized
- ✅ Removed client_additional_test.go
- ✅ Created accumulator_test.go
- ✅ Created streaming_client_test.go
- ✅ Improved test names in conversation_test.go

### pkg/controllers → Could be simplified
```
pkg/controllers/
├── chat/                 # Chat-specific controllers
│   ├── basic.go
│   ├── langchain.go
│   └── streaming.go
├── models/               # Model management
│   └── models.go
├── vectorstore/          # Vector store integration
│   └── vectorstore.go
└── factory.go            # Controller factory
```

### pkg/langchain → Already focused (good as-is)

### pkg/tools → Could be split by functionality
```
pkg/tools/
├── core/                 # Core tool interfaces
│   ├── registry.go
│   ├── types.go
│   └── adapters.go
├── file/                 # File operations
│   ├── read.go
│   ├── write.go
│   └── tree.go
├── code/                 # Code analysis
│   ├── ast.go
│   ├── dependency_graph.go
│   └── grep.go
├── vcs/                  # Version control
│   └── git.go
├── web/                  # Web operations
│   └── webfetch.go
└── system/               # System operations
    └── bash.go
```

## Completed Refactoring

### pkg/chat Package
1. ✅ Moved tests from `client_additional_test.go` to appropriate files:
   - `accumulator_test.go` - Message accumulator tests
   - `streaming_client_test.go` - Streaming client tests
   - `conversation_test.go` - Added NewConversationFromTree test
   - `client_test.go` - Added ChatRequest with Tools test

2. ✅ Improved test naming:
   - Removed generic "Coverage" tests
   - Made test names descriptive of what they test

## Next Steps

1. **Prioritize by impact**: Start with packages that are causing the most confusion
2. **Incremental refactoring**: Move one sub-package at a time
3. **Update imports**: Ensure all imports are updated after moving files
4. **Run tests**: After each refactoring step, run `task test` to ensure nothing breaks

## Test Naming Guidelines

### Good Test Names
- `TestClient_SendMessage` - Tests specific method
- `TestMessageAccumulator_AddChunk` - Tests specific functionality
- `TestContextTree_BranchFromMessage` - Tests specific behavior

### Bad Test Names (to avoid)
- `TestAdditional` - Not descriptive
- `TestCoverage` - Implies test is only for coverage
- `TestBasic` - Too generic
- `Test_test` - Redundant

## Benefits of Refactoring

1. **Better organization** - Easier to find and understand code
2. **Single responsibility** - Each package has one clear purpose
3. **Improved testability** - Smaller, focused packages are easier to test
4. **Better maintainability** - Changes are isolated to specific areas
5. **Clearer dependencies** - Package structure reflects actual dependencies
