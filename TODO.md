# Planning agent
- startup: do we have continued context (--continue was passed and a chat history exists)? if so read the last todo list
- user prompt: when the user adds a question or statement into the prompt, the planning agent will disect the query and decided what steps need to be taken in order to fulfill the request. This planning agent must have the ability to:
    - get which tools are avail and what purpose they serve
    - get which mcp servers are avail and what purpose they serve
    - be able to spawn multiple child processes and manage their lifetimes and data streams
    - cancel actions in progress