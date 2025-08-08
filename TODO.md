- Implement memory: https://pkg.go.dev/github.com/tmc/langchaingo@v0.1.13/memory/sqlite3. example: https://pkg.go.dev/github.com/tmc/langchaingo@v0.1.13/examples/chains-conversation-memory-sqlite Add the settings option for "langchain.memory" which has a default value of true. When true the path for the db will be the viper config root>/memory.db
- Tool usage. Scaffold out the core tools: Bash(), WebSearch(), List(), Search(), Find(), Read(), ReadMany(), Write(). Each of these tools needs to be in their own package: pkg/toole/{bash,list,read_many,etc}. For now we just need to scaffold out the common interface by following the langchain documentation around tool usage: https://python.langchain.com/docs/how_to/#tools since these docs dont exist on the golang official docs you would need to reference the pkg.go.dev pages.


- llm chaining
