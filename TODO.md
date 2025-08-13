# Refactor to single agent without routing
- Code should only have a single ReAct agent that is the core. All other code relating to routing, orchestration, or other terms that describe plural agentic paradigms need to be removed and refactored. Claudes objective is to have a single ReAct agent. The system prompt, tool definitions, and other related code need to be removed. Our goal is to distill this down to its most basic shell of an agent. In the end we should have a single langchain reAct client, an ollama client for model management, the TUI, memory and vector store.

# Part 2
- Skipped tests? There shouldn't be any. We need to review what is being skipped and if the logic needs updating or the test should be completely removed. Now that we have the core react agent logic we need to ensure we are able to test this thoroughly and completely end to end in a way that asserts that the planning and tool calling is happening.

- Some kind of observable state that the TUI can use to report on the status of an agent in progress. This should show things like which tool is being used and what the output is (truncated to a few lines) example:
```
> how many files are in this dir?

● Bash(ls -al)
  ⎿ <output of command>
     ...truncated

● <agent response>

> Summarize the file

● Read(./path/to/file)

● <agent response>
  ...if markdown show markdown formatted

● Bash(git add .)
  ⎿ no content

```

where a cheveron > denotes a user inputted message and the ● denote a tool or agent response. These messages / state need to be able to be observed in real time and displayed as they come in but still totally decoupled from the TUI. The TUI should just receive the data and display it. As little processing as possble should happen in the TUI to keep it totally decoupled and testable

- Cancel in progress
- Doublecheck our tools are following the way that langchain expects these: https://tmc.github.io/langchaingo/docs/modules/agents/ and for the chains as well
- Text processing middleware and markdown styling.
- text splitters for large text input
- memory adjustments: https://tmc.github.io/langchaingo/docs/modules/memory/ need to adjust the mode dynamically
