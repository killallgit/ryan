# Foundation for inner monologue
- A new configuration file needs to be managed by viper. It should be called "self.yaml" and it will contain the system prompt, personaility types that will later be used to customize system prompts and tool abilities

# AgentInnerMonologue
- Define several distinct "personality traits" each their own, separate LLM abstractions


# UI/UX
- I want to give proper feedback that a tool is being called. Lets show feedback in the chat that looks like: <TOOL>(<truncated-command>). example. "Shell(docker ps -a)" or "Search("https://www.wikipedia.com")"
