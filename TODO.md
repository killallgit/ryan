# NOT YET
- Streaming text doesn't seem to work. Either we're not getting it in streaming form or the events are not registering a change. I expect to see the <think><MESSAGE_STREAM...></think> which i know is somehow avail to get access to through langchain.
- chat_history.json should be saved no matter what type of log level is enabled. This is a good opportunity to setup our contextmanager.
- status bar tokens are not displayed. We need to think about how to access these from langchain
- system.log is not using the path set in the settings file -> logging.log_file
- Implement memory: https://pkg.go.dev/github.com/tmc/langchaingo@v0.1.13/memory/sqlite3. example: https://pkg.go.dev/github.com/tmc/langchaingo@v0.1.13/examples/chains-conversation-memory-sqlite Add the settings option for "langchain.memory" which has a default value of true. When true the path for the db will be the viper config root>/memory.db
- Tool usage. Scaffold out the core tools: Bash(), WebSearch(), List(), Search(), Find(), Read(), ReadMany(), Write(). Each of these tools needs to be in their own package: pkg/toole/{bash,list,read_many,etc}. For now we just need to scaffold out the common interface by following the langchain documentation around tool usage: https://python.langchain.com/docs/how_to/#tools since these docs dont exist on the golang official docs you would need to reference the pkg.go.dev pages.


- llm chaining
