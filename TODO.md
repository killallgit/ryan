# Add LLM testing framework from langchain-go
- https://tmc.github.io/langchaingo/docs/modules/model_io/models/llms/Integrations/fake
- Write a single test to get aquainted
- Create a plan to augment our current tests with this fake agent

# AgentRyan
- We need some kind of persistant document embedder that can index files for future
- Main planning agent loop.
  0. If the `--continue` flag was passed at app startup `viper.GetString("continue")` load previous context with a context loader.
  1. Receive prompt input. Create a plan of action. example: "add an agent to my cli"
  2. Create the plan by asking this of the prompt using a prompt template: "Create an initial plan of action to complete or address the following user input: {{USER_PROMPT}}
  3. Write task-list to a file

# CLEANUP
- Do a full review of the code and find any interfaces that might be duplicated, any dead code, or code that can be logically combined with existing so interfaces and logic is unified.



# UI/UX
- I want to give proper feedback that a tool is being called. Lets show feedback in the chat that looks like: <TOOL>(<truncated-command>). example. "Shell(docker ps -a)" or "Search("https://www.wikipedia.com")"
